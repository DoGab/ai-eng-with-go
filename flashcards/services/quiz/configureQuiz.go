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
	configQuizSystemPrompt = `You are a quiz configuration assistant. Your job is to interview users to understand what kind of quiz they want to create.

Ask about:
1. How many questions they want (if not specified)
2. What topics/subjects they want to focus on (if not specified)

Be conversational and helpful. Once you have enough information to create a quiz configuration, call the finalize_quiz_config function with the appropriate parameters.

IMPORTANT: When extracting topics for search, be very precise and only use the EXACT keywords the user mentioned. Do not expand or interpret their request - use only their specific words. For example:
- If user says "scalability" → use ["scalability"]
- If user says "database performance" → use ["database", "performance"] 
- If user says "caching" → use ["caching"]
- Do NOT add related terms like "distributed systems" unless the user specifically mentioned them.

If you need more information, call continue_interview to ask follow-up questions.`
)

var configQuizTools = []llms.Tool{
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
						"maximum":     50,
					},
					"topics": map[string]any{
						"type":        "array",
						"description": "Array of EXACT topic keywords that the user specifically mentioned. Do not add related or interpreted terms - only use the user's exact words.",
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

func (qs *Service) ConfigureQuiz(messages []models.Message) (*models.QuizConfigResponse, error) {
	log.Printf("[INFO] Starting quiz configuration with %d existing messages", len(messages))

	ctx := context.Background()
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, configQuizSystemPrompt),
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

	log.Printf("[INFO] Calling LLM for quiz configuration")
	resp, err := qs.llm.GenerateContent(ctx, messageHistory, 
		llms.WithTools(configQuizTools), 
		llms.WithTemperature(0.7),
		llms.WithToolChoice("required"))
	if err != nil {
		log.Printf("[ERROR] Failed to generate quiz configuration response: %v", err)
		return nil, fmt.Errorf("failed to generate quiz configuration response: %w", err)
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

		return &models.QuizConfigResponse{
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

		log.Printf("[INFO] Searching for notes with topics: %v", params.Topics)
		matchingNotes, err := qs.noteService.SearchNotesByContent(params.Topics)
		if err != nil {
			log.Printf("[ERROR] Failed to search notes by content: %v", err)
			return nil, fmt.Errorf("failed to search notes by content: %w", err)
		}
		log.Printf("[INFO] Found %d matching notes for topics %v", len(matchingNotes), params.Topics)

		if len(matchingNotes) == 0 {
			log.Printf("[ERROR] No notes found matching topics: %v", params.Topics)
			return &models.QuizConfigResponse{
				Type:    "continue",
				Message: fmt.Sprintf("I couldn't find any notes about %s. Could you specify different topics or be more specific about what you'd like to study?", strings.Join(params.Topics, ", ")),
				Config:  nil,
			}, nil
		}

		noteIDs := make([]int, len(matchingNotes))
		for i, note := range matchingNotes {
			noteIDs[i] = note.ID
		}

		config := &models.QuizConfiguration{
			NoteIDs:       noteIDs,
			QuestionCount: params.QuestionCount,
			Topic:         strings.Join(params.Topics, ", "),
		}

		log.Printf("[INFO] Successfully configured quiz with %d notes and %d questions", len(noteIDs), params.QuestionCount)
		return &models.QuizConfigResponse{
			Type:    "configure",
			Message: params.Reasoning,
			Config:  config,
		}, nil

	default:
		log.Printf("[ERROR] Unknown function call: %s", toolCall.FunctionCall.Name)
		return nil, fmt.Errorf("unknown function call: %s", toolCall.FunctionCall.Name)
	}
}