package quiz

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"flashcards/models"

	"github.com/samber/lo"
	"github.com/tmc/langchaingo/llms"
)

const (
	conductQuizSystemPrompt = `You are an interactive quiz assistant. Your role is to conduct engaging quiz sessions based on study notes.

BEHAVIOR GUIDELINES:
1. If this is the start of a conversation (no previous messages), generate ONE thoughtful, open-ended question based on the provided notes and topics.

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

var conductQuizTools = []llms.Tool{
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

func buildConductQuizPrompt(notes []*models.Note, topics []string, messages []models.Message) string {
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
	for _, note := range notes {
		prompt.WriteString(fmt.Sprintf("- %s\n", note.Content))
	}

	if len(messages) > 0 {
		prompt.WriteString("\nConversation History:\n")
		for _, msg := range messages {
			prompt.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
	}

	return prompt.String()
}

func (qs *Service) ConductQuiz(noteIDs []int, topics []string, messages []models.Message) (*models.QuizConductResponse, error) {
	log.Printf("[INFO] Starting quiz conduct with %d notes, topics: %v, %d messages", len(noteIDs), topics, len(messages))

	if len(noteIDs) == 0 {
		return nil, fmt.Errorf("at least one note ID is required")
	}

	if len(topics) == 0 {
		return nil, fmt.Errorf("at least one topic is required")
	}

	allNotes, err := qs.noteService.GetAllNotes()
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve notes: %v", err)
		return nil, fmt.Errorf("failed to retrieve notes: %w", err)
	}

	targetNotes := lo.Filter(allNotes, func(note *models.Note, _ int) bool {
		return lo.Contains(noteIDs, note.ID)
	})

	if len(targetNotes) != len(noteIDs) {
		foundIDs := lo.Map(targetNotes, func(note *models.Note, _ int) int {
			return note.ID
		})
		missingIDs := lo.Filter(noteIDs, func(noteID int, _ int) bool {
			return !lo.Contains(foundIDs, noteID)
		})
		log.Printf("[ERROR] Note IDs not found: %v", missingIDs)
		return nil, fmt.Errorf("note IDs not found: %v", missingIDs)
	}

	prompt := buildConductQuizPrompt(targetNotes, topics, messages)

	ctx := context.Background()
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, conductQuizSystemPrompt),
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

	log.Printf("[INFO] Calling LLM for quiz conduct")
	resp, err := qs.llm.GenerateContent(ctx, messageHistory,
		llms.WithTools(conductQuizTools),
		llms.WithTemperature(0.7),
		llms.WithToolChoice("required"))
	if err != nil {
		log.Printf("[ERROR] Failed to generate quiz conduct response: %v", err)
		return nil, fmt.Errorf("failed to generate quiz conduct response: %w", err)
	}

	if len(resp.Choices) == 0 || len(resp.Choices[0].ToolCalls) == 0 {
		log.Printf("[ERROR] No tool calls in LLM quiz conduct response")
		return nil, fmt.Errorf("no tool calls in LLM quiz conduct response")
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

		return &models.QuizConductResponse{
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

		return &models.QuizConductResponse{
			Type:       "evaluate",
			Message:    params.Feedback,
			Evaluation: evaluation,
		}, nil

	default:
		log.Printf("[ERROR] Unknown function call: %s", toolCall.FunctionCall.Name)
		return nil, fmt.Errorf("unknown function call: %s", toolCall.FunctionCall.Name)
	}
}