package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
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

IMPORTANT: When extracting topics for search, be very precise and only use the EXACT keywords the user mentioned. Do not expand or interpret their request - use only their specific words. For example:
- If user says "scalability" → use ["scalability"]
- If user says "database performance" → use ["database", "performance"] 
- If user says "caching" → use ["caching"]
- Do NOT add related terms like "distributed systems" unless the user specifically mentioned them.

If you need more information, call continue_interview to ask follow-up questions.`
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

type RankNotesParams struct {
	Rankings []NoteRanking `json:"rankings"`
}

type NoteRanking struct {
	NoteID int     `json:"note_id"`
	Score  float64 `json:"score"`
}

type ContinueQuizParams struct {
	Message string `json:"message"`
}

type EvaluateAnswerParams struct {
	Message  string `json:"message"`
	Correct  bool   `json:"correct"`
	Feedback string `json:"feedback"`
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

var noteRankingTools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "rank_notes",
			Description: "Rank the provided notes by relevance to the given topics",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"rankings": map[string]any{
						"type":        "array",
						"description": "Array of note rankings with relevance scores",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"note_id": map[string]any{
									"type":        "integer",
									"description": "The ID of the note being ranked",
								},
								"score": map[string]any{
									"type":        "number",
									"description": "Relevance score from 0.0 to 1.0, where 1.0 is most relevant",
									"minimum":     0.0,
									"maximum":     1.0,
								},
							},
							"required": []string{"note_id", "score"},
						},
					},
				},
				"required": []string{"rankings"},
			},
		},
	},
}

var quizConductTools = []llms.Tool{
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
			Description: "Evaluate the user's answer to the quiz question and provide final feedback",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message": map[string]any{
						"type":        "string",
						"description": "The response message to the user after evaluation",
					},
					"correct": map[string]any{
						"type":        "boolean",
						"description": "Whether the user's answer is correct",
					},
					"feedback": map[string]any{
						"type":        "string",
						"description": "Detailed feedback explaining why the answer is correct or incorrect",
					},
				},
				"required": []string{"message", "correct", "feedback"},
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

func (qs *QuizService) RankNotes(noteIDs []int, topics []string) (*models.NoteRankResponse, error) {
	log.Printf("[INFO] Starting note ranking for %d notes with topics: %v", len(noteIDs), topics)

	if len(noteIDs) == 0 {
		return nil, fmt.Errorf("at least one note ID is required")
	}
	
	if len(topics) == 0 {
		return nil, fmt.Errorf("at least one topic is required")
	}

	// Get all notes to filter by IDs
	allNotes, err := qs.noteService.GetAllNotes()
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve notes: %v", err)
		return nil, fmt.Errorf("failed to retrieve notes: %w", err)
	}

	// Filter notes by provided IDs using lo library
	targetNotes := lo.Filter(allNotes, func(note *models.Note, _ int) bool {
		return lo.Contains(noteIDs, note.ID)
	})

	// Check if all requested note IDs were found
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

	// Prepare prompt for AI ranking
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

	// Convert to response format using lo.Map
	rankedNotes := lo.Map(params.Rankings, func(ranking NoteRanking, _ int) models.RankedNote {
		return models.RankedNote{
			NoteID: ranking.NoteID,
			Score:  ranking.Score,
		}
	})

	// Sort by score descending using slices.SortFunc
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

func (qs *QuizService) buildRankingPrompt(notes []*models.Note, topics []string) string {
	var prompt strings.Builder
	
	prompt.WriteString("Please rank the following notes by relevance to these topics: ")
	prompt.WriteString(strings.Join(topics, ", "))
	prompt.WriteString("\n\nAssign each note a relevance score from 0.0 to 1.0 where:\n")
	prompt.WriteString("- 1.0 = Highly relevant and directly addresses the topics\n")
	prompt.WriteString("- 0.7-0.9 = Moderately relevant with some connection to topics\n")
	prompt.WriteString("- 0.4-0.6 = Somewhat relevant with tangential connection\n")
	prompt.WriteString("- 0.1-0.3 = Minimally relevant with weak connection\n")
	prompt.WriteString("- 0.0 = Not relevant to the topics\n\n")
	
	prompt.WriteString("Notes to rank:\n\n")
	for _, note := range notes {
		prompt.WriteString(fmt.Sprintf("Note ID %d:\n%s\n\n", note.ID, note.Content))
	}
	
	prompt.WriteString("Use the rank_notes function to provide your rankings.")
	
	return prompt.String()
}

func (qs *QuizService) ConductQuiz(noteIDs []int, topics []string, messages []models.Message) (*models.QuizConductResponse, error) {
	log.Printf("[INFO] Starting quiz conduct with %d notes, topics: %v, %d messages", len(noteIDs), topics, len(messages))

	if len(noteIDs) == 0 {
		return nil, fmt.Errorf("at least one note ID is required")
	}

	if len(topics) == 0 {
		return nil, fmt.Errorf("at least one topic is required")
	}

	// Get notes for quiz content
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

	// Build conversation prompt
	prompt := qs.buildQuizConductPrompt(targetNotes, topics, messages)

	ctx := context.Background()
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, qs.getQuizConductSystemPrompt()),
	}

	// Add conversation history
	for _, msg := range messages {
		var msgType llms.ChatMessageType
		if msg.Role == "user" {
			msgType = llms.ChatMessageTypeHuman
		} else {
			msgType = llms.ChatMessageTypeAI
		}
		messageHistory = append(messageHistory, llms.TextParts(msgType, msg.Content))
	}

	// Add the current prompt
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
			Correct:  params.Correct,
			Feedback: params.Feedback,
		}

		return &models.QuizConductResponse{
			Type:       "evaluate",
			Message:    params.Message,
			Evaluation: evaluation,
		}, nil

	default:
		log.Printf("[ERROR] Unknown function call: %s", toolCall.FunctionCall.Name)
		return nil, fmt.Errorf("unknown function call: %s", toolCall.FunctionCall.Name)
	}
}

func (qs *QuizService) getQuizConductSystemPrompt() string {
	return `You are an interactive quiz assistant. Your role is to conduct engaging quiz sessions based on study notes.

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
}

func (qs *QuizService) buildQuizConductPrompt(notes []*models.Note, topics []string, messages []models.Message) string {
	var prompt strings.Builder

	if len(messages) == 0 {
		// First message - generate initial question
		prompt.WriteString("Generate a thoughtful quiz question based on these study materials.\n\n")
		prompt.WriteString("Focus areas: ")
		prompt.WriteString(strings.Join(topics, ", "))
		prompt.WriteString("\n\nStudy materials:\n")
		for _, note := range notes {
			prompt.WriteString(fmt.Sprintf("- %s\n", note.Content))
		}
		prompt.WriteString("\nGenerate ONE open-ended question that tests understanding of these concepts.")
	} else {
		// Continuing conversation - provide context
		prompt.WriteString("Continue the quiz conversation based on the user's latest response.\n\n")
		prompt.WriteString("Quiz topics: ")
		prompt.WriteString(strings.Join(topics, ", "))
		prompt.WriteString("\n\nReference materials:\n")
		for _, note := range notes {
			prompt.WriteString(fmt.Sprintf("- %s\n", note.Content))
		}
		prompt.WriteString("\nEvaluate if the user has answered the question or if they need guidance to get back on track.")
	}

	return prompt.String()
}
