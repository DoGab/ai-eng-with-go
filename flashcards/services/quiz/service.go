package quiz

import (
	"fmt"

	"flashcards/services"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Service struct {
	noteService      *services.NoteService
	quizStoreService *services.QuizStoreService
	llm              llms.Model
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
	IsCorrect      bool   `json:"is_correct"`
	Feedback       string `json:"feedback"`
	CorrectAnswer  string `json:"correct_answer,omitempty"`
	Encouragement  string `json:"encouragement,omitempty"`
}

type GenerateQuizResult struct {
	Content string `json:"content"`
}

func NewService(noteService *services.NoteService, quizStoreService *services.QuizStoreService, apiKey string) *Service {
	llm, err := openai.New(
		openai.WithModel("gpt-4o-mini"),
		openai.WithToken(apiKey),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to create OpenAI client: %v", err))
	}

	return &Service{
		noteService:      noteService,
		quizStoreService: quizStoreService,
		llm:              llm,
	}
}