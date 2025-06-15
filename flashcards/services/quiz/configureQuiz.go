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

func (qs *QuizService) ConfigureQuiz(messages []models.Message) (*models.QuizConfigResponse, error) {
	log.Printf("[INFO] Starting quiz configuration with %d existing messages", len(messages))

	ctx := context.Background()
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, QUIZ_CONFIG_SYSTEM_PROMPT),
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
		llms.WithTools(quizConfigTools), 
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