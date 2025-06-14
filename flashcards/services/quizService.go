package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"flashcards/models"

	"github.com/samber/lo"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	SYSTEM_PROMPT = `You are a focused quiz assistant that helps users study from their notes. Your task is to ask one thoughtful, non-multiple-choice question based on provided notes. After a user answers, you must clearly say if the answer is correct or not, explain why if it's incorrect, and provide the correct answer. Then, allow the user to ask follow-up questions *only* about that specific topic.

If the user asks anything unrelated to the current question or topic, politely decline to answer. Do not reveal the correct answer or provide hints. Instead, remind the user to answer the original question or ask a follow-up related to the topic at hand.

Never respond with metadata, formatting, or explain who you are. Just give direct, human-like responses.`

	INITIAL_QUIZ_PROMPT = `Based on the following study notes, generate one open-ended, thought-provoking question that tests the user's understanding. The question should not be multiple choice. Keep it focused and relevant.

Notes:
%s`

	CONVERSATION_PROMPT = `Continue the quiz conversation. Use the notes and the conversation so far to guide your response.

If the last user answer is correct, acknowledge it simply and briefly.

If the answer is incorrect, clearly explain why it's wrong, then provide the correct answer.

If the user asks something unrelated to the current question or topic, do NOT give the correct answer or any hints. Instead, respond that you only answer questions about the current topic and ask the user to answer the original question or stay on topic.

Notes:
%s

Conversation:
%s`
)

type QuizService struct {
	noteService *NoteService
	llm         llms.Model
}

func NewQuizService(noteService *NoteService, apiKey string) *QuizService {
	llm, err := openai.New(
		openai.WithModel("gpt-4o-mini"),
		openai.WithToken(apiKey),
	)
	if err != nil {
		log.Fatalf("Failed to initialize OpenAI client: %v", err)
	}

	return &QuizService{
		noteService: noteService,
		llm:         llm,
	}
}

type GenerateQuizResult struct {
	NoteIDs  []int
	Messages []models.Message
}

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
		NoteIDs:  noteIDs,
		Messages: updatedMessages,
	}, nil
}

func (qs *QuizService) formatNotesContent(notes []*models.Note) string {
	if len(notes) == 0 {
		return "No notes available for quiz generation."
	}

	var content strings.Builder
	for i, note := range notes {
		content.WriteString(fmt.Sprintf("Note %d: %s\n", i+1, note.Content))
	}
	return content.String()
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

func (qs *QuizService) prepareQuizPrompt(noteIDs []int, messages []models.Message, operationType string) (string, error) {
	log.Printf("[INFO] Starting %s with %d existing messages", operationType, len(messages))

	log.Printf("[INFO] Retrieving notes for %s", operationType)
	notes, err := qs.noteService.GetAllNotes()
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve notes: %v", err)
		return "", fmt.Errorf("failed to retrieve notes: %w", err)
	}
	log.Printf("[INFO] Retrieved %d notes for %s", len(notes), operationType)

	filteredNotes := lo.Filter(notes, func(note *models.Note, index int) bool {
		return lo.Contains(noteIDs, note.ID)
	})
	if len(filteredNotes) == 0 {
		return "", fmt.Errorf("at least one valid note id is required")
	}

	notesContent := qs.formatNotesContent(filteredNotes)

	var prompt string
	if len(messages) == 0 {
		log.Printf("[INFO] Generating initial quiz question for %s", operationType)
		prompt = fmt.Sprintf(INITIAL_QUIZ_PROMPT, notesContent)
	} else {
		log.Printf("[INFO] Generating follow-up quiz question for existing conversation in %s", operationType)
		conversationHistory := qs.formatConversationHistory(messages)
		prompt = fmt.Sprintf(CONVERSATION_PROMPT, notesContent, conversationHistory)
	}

	return prompt, nil
}

func (qs *QuizService) formatConversationHistory(messages []models.Message) string {
	var history strings.Builder
	for _, msg := range messages {
		history.WriteString(fmt.Sprintf("%s: %s\n", strings.Title(msg.Role), msg.Content))
	}

	return history.String()
}
