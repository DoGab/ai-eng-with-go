package services

import (
	"context"
	"encoding/json"
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

	QUIZ_CONFIG_SYSTEM_PROMPT = `You are a quiz configuration assistant. Your job is to interview users to understand what kind of quiz they want to create.

Ask about:
1. How many questions they want (if not specified)
2. What topics/subjects they want to focus on (if not specified)

Be conversational and helpful. Once you have enough information to create a quiz configuration, call the finalize_quiz_config function with the appropriate parameters.

If you need more information, call continue_interview to ask follow-up questions.

Available notes cover various topics. When the user mentions a topic, try to match it to available content.`
)

type QuizService struct {
	noteService *NoteService
	llm         llms.Model
}

type ContinueInterviewParams struct {
	Message string `json:"message"`
}

type FinalizeConfigParams struct {
	QuestionCount int      `json:"question_count"`
	Topics        []string `json:"topics"`
	Reasoning     string   `json:"reasoning"`
}

var quizConfigTools = []llms.Tool{
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
						"description": "Array of topic keywords to search for in notes",
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

		matchingNotes, err := qs.noteService.SearchNotesByContent(params.Topics)
		if err != nil {
			log.Printf("[ERROR] Failed to search notes by content: %v", err)
			return nil, fmt.Errorf("failed to search notes by content: %w", err)
		}

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
