package quiz

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"flashcards/models"

	"github.com/samber/lo"
	"github.com/tmc/langchaingo/llms"
)

func (qs *QuizService) RankNotes(noteIDs []int, topics []string) (*models.NoteRankResponse, error) {
	log.Printf("[INFO] Starting note ranking for %d notes with topics: %v", len(noteIDs), topics)

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

	log.Printf("[INFO] Found %d notes to rank", len(targetNotes))

	prompt := qs.buildRankingPrompt(targetNotes, topics)

	ctx := context.Background()
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	log.Printf("[INFO] Calling LLM for note ranking")
	resp, err := qs.llm.GenerateContent(ctx, messageHistory,
		llms.WithTools(noteRankingTools),
		llms.WithTemperature(0.3),
		llms.WithToolChoice("required"))
	if err != nil {
		log.Printf("[ERROR] Failed to generate ranking response: %v", err)
		return nil, fmt.Errorf("failed to generate ranking response: %w", err)
	}

	if len(resp.Choices) == 0 || len(resp.Choices[0].ToolCalls) == 0 {
		log.Printf("[ERROR] No tool calls in LLM ranking response")
		return nil, fmt.Errorf("no tool calls in LLM ranking response")
	}

	toolCall := resp.Choices[0].ToolCalls[0]
	if toolCall.FunctionCall.Name != "rank_notes" {
		log.Printf("[ERROR] Unexpected function call: %s", toolCall.FunctionCall.Name)
		return nil, fmt.Errorf("unexpected function call: %s", toolCall.FunctionCall.Name)
	}

	var params RankNotesParams
	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &params); err != nil {
		log.Printf("[ERROR] Failed to parse ranking arguments: %v", err)
		return nil, fmt.Errorf("failed to parse ranking arguments: %w", err)
	}

	rankedNotes := lo.Map(params.Rankings, func(ranking NoteRanking, _ int) models.RankedNote {
		return models.RankedNote{
			NoteID: ranking.NoteID,
			Score:  ranking.Score,
		}
	})

	slices.SortFunc(rankedNotes, func(a, b models.RankedNote) int {
		if a.Score > b.Score {
			return -1
		}
		if a.Score < b.Score {
			return 1
		}
		return 0
	})

	log.Printf("[INFO] Successfully ranked %d notes", len(rankedNotes))
	return &models.NoteRankResponse{
		RankedNotes: rankedNotes,
	}, nil
}