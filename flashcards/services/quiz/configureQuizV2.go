package quiz

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"flashcards/models"

	"github.com/tmc/langchaingo/llms"
)

const (
	configQuizV2SystemPrompt = `You are a quiz configuration assistant. Your job is to interview users to understand what kind of quiz they want to create.

Ask about:
1. How many questions they want (if not specified)
2. What topics/subjects they want to focus on (if not specified)

Be conversational and helpful. Once you have enough information to create a quiz configuration, call the finalize_quiz_config function with the appropriate parameters.

CRITICAL: When extracting topics, treat the user's input as COMPLETE TOPIC PHRASES, not individual keywords. For example:
- If user says "testing distributed systems" → use ["testing distributed systems"] as ONE topic
- If user says "database performance optimization" → use ["database performance optimization"] as ONE topic
- If user says "scalability and caching" → use ["scalability", "caching"] as TWO separate topics
- Do NOT break down topic phrases into individual words unless the user clearly mentions multiple distinct topics

Only create multiple topics if the user explicitly mentions multiple separate subjects (like "I want to study both X and Y").

If you need more information, call continue_interview to ask follow-up questions.`
)

var configQuizV2Tools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "continue_interview",
			Description: "Continue interviewing the user to gather more information about their quiz preferences",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message": map[string]any{
						"type":        "string",
						"description": "The message to send to the user to continue the interview",
					},
				},
				"required": []string{"message"},
			},
		},
	},
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "finalize_quiz_config",
			Description: "Finalize the quiz configuration based on the user's preferences",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"question_count": map[string]any{
						"type":        "integer",
						"description": "Number of questions for the quiz",
						"minimum":     1,
						"maximum":     5,
					},
					"topics": map[string]any{
						"type":        "array",
						"description": "Array of complete topic phrases that the user mentioned. Keep topic phrases intact (e.g., 'testing distributed systems' as one topic, not split into separate words). Only create multiple array items if user mentions multiple distinct subjects.",
						"items": map[string]any{
							"type": "string",
						},
					},
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Brief explanation of the configuration choices",
					},
				},
				"required": []string{"question_count", "topics", "reasoning"},
			},
		},
	},
}

func (qs *Service) ConfigureQuizV2(messages []models.Message) (*models.QuizV2ConfigResponse, error) {
	log.Printf("[INFO] Starting quiz v2 configuration with %d existing messages", len(messages))

	ctx := context.Background()
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, configQuizV2SystemPrompt),
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

	log.Printf("[INFO] Calling LLM for quiz v2 configuration")
	resp, err := qs.llm.GenerateContent(ctx, messageHistory,
		llms.WithTools(configQuizV2Tools),
		llms.WithTemperature(0.7),
		llms.WithToolChoice("required"))
	if err != nil {
		log.Printf("[ERROR] Failed to generate quiz v2 configuration response: %v", err)
		return nil, fmt.Errorf("failed to generate quiz v2 configuration response: %w", err)
	}

	if len(resp.Choices) == 0 || len(resp.Choices[0].ToolCalls) == 0 {
		log.Printf("[ERROR] No tool calls in LLM response")
		return nil, fmt.Errorf("no tool calls in LLM response")
	}

	toolCall := resp.Choices[0].ToolCalls[0]
	log.Printf("[INFO] LLM called function: %s", toolCall.FunctionCall.Name)

	switch toolCall.FunctionCall.Name {
	case "continue_interview":
		var params ContinueInterviewParams
		if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &params); err != nil {
			log.Printf("[ERROR] Failed to parse continue_interview arguments: %v", err)
			return nil, fmt.Errorf("failed to parse continue_interview arguments: %w", err)
		}

		return &models.QuizV2ConfigResponse{
			Type:    "continue",
			Message: params.Message,
			Config:  nil,
		}, nil

	case "finalize_quiz_config":
		var params FinalizeConfigParams
		if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &params); err != nil {
			log.Printf("[ERROR] Failed to parse finalize_quiz_config arguments: %v", err)
			return nil, fmt.Errorf("failed to parse finalize_quiz_config arguments: %w", err)
		}

		log.Printf("[INFO] Configuration ready for topics: %v", params.Topics)
		log.Printf("[INFO] Topics will be used to generate quiz content from document index when quiz is created")

		config := &models.QuizV2Configuration{
			QuestionCount: params.QuestionCount,
			Topics:        params.Topics,
		}

		log.Printf("[INFO] Successfully configured v2 quiz with %d topics and %d questions", len(params.Topics), params.QuestionCount)
		return &models.QuizV2ConfigResponse{
			Type:    "configure",
			Message: params.Reasoning,
			Config:  config,
		}, nil

	default:
		log.Printf("[ERROR] Unknown function call: %s", toolCall.FunctionCall.Name)
		return nil, fmt.Errorf("unknown function call: %s", toolCall.FunctionCall.Name)
	}
}