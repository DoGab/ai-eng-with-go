package models

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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