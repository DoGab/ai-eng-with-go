package models

import "time"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Quiz struct {
	ID              int                 `json:"id" db:"id"`
	Config          QuizV2Configuration `json:"config" db:"config"`
	LLMContext      string              `json:"-" db:"llm_context"`
	AskedQuestions  []string            `json:"asked_questions" db:"asked_questions"`
	CreatedAt       time.Time           `json:"createdAt" db:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt" db:"updatedAt"`
}

type CreateQuizRequest struct {
	Config QuizV2Configuration `json:"config"`
}

type QuizConfigRequest struct {
	Messages []Message `json:"messages"`
}

type QuizConfigResponse struct {
	Type    string             `json:"type"`
	Message string             `json:"message"`
	Config  *QuizConfiguration `json:"config"`
}

type QuizConfiguration struct {
	NoteIDs       []int  `json:"note_ids"`
	QuestionCount int    `json:"question_count"`
	Topic         string `json:"topic"`
}

type NoteRankRequest struct {
	NoteIDs []int    `json:"note_ids"`
	Topics  []string `json:"topics"`
}

type NoteRankResponse struct {
	RankedNotes []RankedNote `json:"ranked_notes"`
}

type RankedNote struct {
	NoteID int     `json:"note_id"`
	Score  float64 `json:"score"`
}

type QuizConductRequest struct {
	NoteIDs  []int     `json:"note_ids"`
	Topics   []string  `json:"topics"`
	Messages []Message `json:"messages"`
}

type QuizConductResponse struct {
	Type       string          `json:"type"`
	Message    string          `json:"message"`
	Evaluation *QuizEvaluation `json:"evaluation"`
}

type QuizEvaluation struct {
	Correct  bool   `json:"correct"`
	Feedback string `json:"feedback"`
}

type QuizV2ConfigRequest struct {
	Messages []Message `json:"messages"`
}

type QuizV2ConfigResponse struct {
	Type    string               `json:"type"`
	Message string               `json:"message"`
	Config  *QuizV2Configuration `json:"config"`
}

type QuizV2Configuration struct {
	QuestionCount int      `json:"question_count"`
	Topics        []string `json:"topics"`
}

type QuizV2ConductRequest struct {
	QuizID   int       `json:"quiz_id"`
	Messages []Message `json:"messages"`
}

type QuizV2ConductResponse struct {
	Type       string          `json:"type"`
	Message    string          `json:"message"`
	Evaluation *QuizEvaluation `json:"evaluation"`
}

type UpdateQuizRequest struct {
	AskedQuestions []string `json:"asked_questions"`
}