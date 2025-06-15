package quiz

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"flashcards/models"

	"github.com/samber/lo"
	"github.com/tmc/langchaingo/llms"
)

func (qs *QuizService) ConductQuiz(noteIDs []int, topics []string, messages []models.Message) (*models.QuizConductResponse, error) {
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

	prompt := qs.buildQuizConductPrompt(targetNotes, topics, messages)

	ctx := context.Background()
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, QUIZ_CONDUCT_SYSTEM_PROMPT),
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
		llms.WithTools(quizConductTools),
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