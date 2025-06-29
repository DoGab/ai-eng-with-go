package quiz

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"flashcards/models"

	"github.com/tmc/langchaingo/llms"
)

const (
	conductQuizV2SystemPrompt = `You are an interactive quiz assistant. Your role is to conduct engaging quiz sessions based on study content.

BEHAVIOR GUIDELINES:
1. If this is the start of a conversation (no previous messages), generate ONE thoughtful, open-ended question based on the provided content and topics.

2. If the user responds to your question:
   - If they give a genuine attempt to answer the quiz question, use evaluate_answer to provide feedback
   - If they indicate they want to give up (e.g., "I don't know", "I give up", "move to the next question", "skip this", "no idea", or similar), immediately use evaluate_answer and mark their response as incorrect
   - If they go off-topic, ask for clarification, or seem confused, use continue_quiz to guide them back

3. When evaluating answers:
   - Be fair and thorough in your assessment
   - Provide detailed feedback explaining why the answer is correct or incorrect
   - Give constructive guidance for improvement if the answer is wrong
   - If the user gave up, acknowledge their decision and provide the correct answer with explanation
   - DO NOT ask follow-up questions or invite further discussion - the quiz is complete at this point

4. When continuing the conversation:
   - Be supportive and encouraging
   - Help clarify the question if the user seems confused
   - Gently redirect off-topic discussions back to the quiz
   - CRITICAL: When providing clarifications, do NOT reveal or hint at the correct answer
   - Explain concepts or terms without giving away what the user should say in their response

5. Keep responses conversational and engaging, not robotic or formal.

IMPORTANT: Call evaluate_answer when the user makes a genuine attempt to answer OR when they explicitly give up/surrender. Use continue_quiz for everything else.`
)

var conductQuizV2Tools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "continue_quiz",
			Description: "Continue the quiz conversation, provide clarifications, or steer user back to answering the question",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message": map[string]any{
						"type":        "string",
						"description": "The message to continue the conversation with the user",
					},
				},
				"required": []string{"message"},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "evaluate_answer",
			Description: "Evaluate the user's answer and provide detailed feedback",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"is_correct": map[string]any{
						"type":        "boolean",
						"description": "Whether the user's answer is correct",
					},
					"feedback": map[string]any{
						"type":        "string",
						"description": "Detailed feedback explaining the correctness of the answer",
					},
					"correct_answer": map[string]any{
						"type":        "string",
						"description": "The correct answer if the user's answer was incorrect",
					},
					"encouragement": map[string]any{
						"type":        "string",
						"description": "Optional encouragement or additional context",
					},
				},
				"required": []string{"is_correct", "feedback"},
			},
		},
	},
}

func buildConductQuizV2Prompt(content []string, topics []string, messages []models.Message) string {
	var prompt strings.Builder

	if len(messages) == 0 {
		prompt.WriteString("Generate one thoughtful quiz question based on the following study materials")
		if len(topics) > 0 {
			prompt.WriteString(" focusing on: ")
			prompt.WriteString(strings.Join(topics, ", "))
		}
		prompt.WriteString(".\n\n")
	} else {
		prompt.WriteString("Continue the quiz conversation based on the study materials and conversation history")
		if len(topics) > 0 {
			prompt.WriteString(" (topics: ")
			prompt.WriteString(strings.Join(topics, ", "))
			prompt.WriteString(")")
		}
		prompt.WriteString(".\n\n")
	}

	prompt.WriteString("Study Materials:\n")
	for i, chunk := range content {
		prompt.WriteString(fmt.Sprintf("%d. %s\n\n", i+1, chunk))
	}

	if len(messages) > 0 {
		prompt.WriteString("Conversation History:\n")
		for _, msg := range messages {
			prompt.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
	}

	return prompt.String()
}

func (qs *Service) ConductQuizV2(config models.QuizV2Configuration, messages []models.Message) (*models.QuizV2ConductResponse, error) {
	log.Printf("[INFO] Starting quiz v2 conduct with topics: %v, %d messages", config.Topics, len(messages))

	if len(config.Topics) == 0 {
		return nil, fmt.Errorf("at least one topic is required")
	}

	log.Printf("[INFO] Retrieving content for topics: %v", config.Topics)
	content := GetContentForTopics(config.Topics)
	if len(content) == 0 {
		log.Printf("[ERROR] No content available for topics: %v", config.Topics)
		return nil, fmt.Errorf("no content available for topics: %v", config.Topics)
	}
	log.Printf("[INFO] Retrieved %d content chunks for topics %v", len(content), config.Topics)

	prompt := buildConductQuizV2Prompt(content, config.Topics, messages)

	ctx := context.Background()
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, conductQuizV2SystemPrompt),
	}

	for _, msg := range messages {
		var msgType llms.ChatMessageType
		if msg.Role == "user" {
			msgType = llms.ChatMessageTypeHuman
		} else {
			msgType = llms.ChatMessageTypeAI
		}
		messageHistory = append(messageHistory, llms.TextParts(msgType, msg.Content))
	}

	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, prompt))

	log.Printf("[INFO] Calling LLM for quiz v2 conduct")
	resp, err := qs.llm.GenerateContent(ctx, messageHistory,
		llms.WithTools(conductQuizV2Tools),
		llms.WithTemperature(0.7),
		llms.WithToolChoice("required"))
	if err != nil {
		log.Printf("[ERROR] Failed to generate quiz v2 conduct response: %v", err)
		return nil, fmt.Errorf("failed to generate quiz v2 conduct response: %w", err)
	}

	if len(resp.Choices) == 0 || len(resp.Choices[0].ToolCalls) == 0 {
		log.Printf("[ERROR] No tool calls in LLM quiz v2 conduct response")
		return nil, fmt.Errorf("no tool calls in LLM quiz v2 conduct response")
	}

	toolCall := resp.Choices[0].ToolCalls[0]
	log.Printf("[INFO] LLM called function: %s", toolCall.FunctionCall.Name)

	switch toolCall.FunctionCall.Name {
	case "continue_quiz":
		var params ContinueQuizParams
		if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &params); err != nil {
			log.Printf("[ERROR] Failed to parse continue_quiz arguments: %v", err)
			return nil, fmt.Errorf("failed to parse continue_quiz arguments: %w", err)
		}

		return &models.QuizV2ConductResponse{
			Type:       "continue",
			Message:    params.Message,
			Evaluation: nil,
		}, nil

	case "evaluate_answer":
		var params EvaluateAnswerParams
		if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &params); err != nil {
			log.Printf("[ERROR] Failed to parse evaluate_answer arguments: %v", err)
			return nil, fmt.Errorf("failed to parse evaluate_answer arguments: %w", err)
		}

		evaluation := &models.QuizEvaluation{
			Correct:  params.IsCorrect,
			Feedback: params.Feedback,
		}

		return &models.QuizV2ConductResponse{
			Type:       "evaluate",
			Message:    params.Feedback,
			Evaluation: evaluation,
		}, nil

	default:
		log.Printf("[ERROR] Unknown function call: %s", toolCall.FunctionCall.Name)
		return nil, fmt.Errorf("unknown function call: %s", toolCall.FunctionCall.Name)
	}
}