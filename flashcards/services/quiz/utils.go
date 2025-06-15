package quiz

import (
	"fmt"
	"log"
	"strings"

	"flashcards/models"

	"github.com/samber/lo"
	"github.com/tmc/langchaingo/llms"
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

	QUIZ_CONDUCT_SYSTEM_PROMPT = `You are an interactive quiz assistant. Your role is to conduct engaging quiz sessions based on study notes.

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

func (qs *QuizService) buildRankingPrompt(notes []*models.Note, topics []string) string {
	var prompt strings.Builder
	prompt.WriteString("Rank the following notes by their relevance to these topics: ")
	prompt.WriteString(strings.Join(topics, ", "))
	prompt.WriteString("\n\nNotes to rank:\n")

	for _, note := range notes {
		prompt.WriteString(fmt.Sprintf("Note ID %d:\n%s\n\n", note.ID, note.Content))
	}

	prompt.WriteString("Please rank each note with a relevance score from 0.0 to 1.0, where 1.0 means highly relevant and 0.0 means not relevant at all.")
	return prompt.String()
}

func (qs *QuizService) buildQuizConductPrompt(notes []*models.Note, topics []string, messages []models.Message) string {
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