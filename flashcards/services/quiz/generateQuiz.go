package quiz

import (
	"context"
	"fmt"
	"log"
	"strings"

	"flashcards/models"

	"github.com/tmc/langchaingo/llms"
)

func (qs *QuizService) GenerateQuizResponse(noteIDs []int, messages []models.Message) (*GenerateQuizResult, error) {
	prompt, err := qs.prepareQuizPrompt(noteIDs, messages, "quiz generation")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	log.Printf("[INFO] Calling LLM for quiz generation")
	completion, err := llms.GenerateFromSinglePrompt(ctx, qs.llm, prompt, llms.WithTemperature(0.7))
	if err != nil {
		log.Printf("[ERROR] Failed to generate LLM response: %v", err)
		return nil, fmt.Errorf("failed to generate LLM response: %w", err)
	}

	updatedMessages := make([]models.Message, len(messages))
	copy(updatedMessages, messages)

	updatedMessages = append(updatedMessages, models.Message{
		Role:    "assistant",
		Content: strings.TrimSpace(completion),
	})

	log.Printf("[INFO] Successfully generated quiz response, returning %d total messages", len(updatedMessages))
	return &GenerateQuizResult{
		Content: strings.TrimSpace(completion),
	}, nil
}

func (qs *QuizService) GenerateQuizResponseStream(noteIDs []int, messages []models.Message, tokenCallback func(string)) error {
	prompt, err := qs.prepareQuizPrompt(noteIDs, messages, "streaming quiz generation")
	if err != nil {
		return err
	}

	ctx := context.Background()
	log.Printf("[INFO] Calling LLM for streaming quiz generation")
	_, err = llms.GenerateFromSinglePrompt(ctx, qs.llm, prompt,
		llms.WithTemperature(0.7),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			tokenCallback(string(chunk))
			return nil
		}),
	)
	if err != nil {
		log.Printf("[ERROR] Failed to generate streaming LLM response: %v", err)
		return fmt.Errorf("failed to generate streaming LLM response: %w", err)
	}

	log.Printf("[INFO] Successfully completed streaming quiz generation")
	return nil
}

