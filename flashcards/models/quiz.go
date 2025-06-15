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