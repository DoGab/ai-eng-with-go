package services

import (
	"fmt"
	"log"
	"strings"

	"flashcards/db"
	"flashcards/models"
	"flashcards/services/pinecone"
)

type QuizStoreService struct {
	repo           db.QuizRepository
	pineconeService *pinecone.Service
}

func NewQuizStoreService(repo db.QuizRepository, pineconeService *pinecone.Service) *QuizStoreService {
	return &QuizStoreService{
		repo:           repo,
		pineconeService: pineconeService,
	}
}

func (s *QuizStoreService) CreateQuiz(req *models.CreateQuizRequest) (*models.Quiz, error) {
	log.Printf("[INFO] Starting quiz creation with topics: %v", req.Config.Topics)

	if err := s.validateCreateRequest(req); err != nil {
		log.Printf("[ERROR] Quiz creation validation failed: %v", err)
		return nil, err
	}

	log.Printf("[INFO] Generating LLM context via Pinecone for topics: %v", req.Config.Topics)
	llmContext, err := s.pineconeService.QueryTopicChunks(req.Config.Topics, 5)
	if err != nil {
		log.Printf("[ERROR] Failed to generate LLM context: %v", err)
		return nil, fmt.Errorf("failed to generate LLM context: %w", err)
	}

	combinedContext := strings.Join(llmContext, "\n\n=== CHUNK SEPARATOR ===\n\n")
	log.Printf("[INFO] Generated LLM context of length: %d characters", len(combinedContext))

	quiz := &models.Quiz{
		Config:     req.Config,
		LLMContext: combinedContext,
	}

	if err := s.repo.CreateQuiz(quiz); err != nil {
		log.Printf("[ERROR] Failed to create quiz in repository: %v", err)
		return nil, fmt.Errorf("failed to create quiz: %w", err)
	}

	log.Printf("[INFO] Successfully created quiz with ID %d", quiz.ID)
	return quiz, nil
}

func (s *QuizStoreService) GetQuizByID(id int) (*models.Quiz, error) {
	log.Printf("[INFO] Starting get quiz by ID %d", id)

	if id <= 0 {
		log.Printf("[ERROR] Invalid quiz ID provided: %d", id)
		return nil, fmt.Errorf("invalid quiz ID: %d", id)
	}

	quiz, err := s.repo.GetQuizByID(id)
	if err != nil {
		log.Printf("[ERROR] Failed to get quiz by ID %d: %v", id, err)
		return nil, err
	}

	log.Printf("[INFO] Successfully retrieved quiz with ID %d", id)
	return quiz, nil
}

func (s *QuizStoreService) GetAllQuizzes() ([]*models.Quiz, error) {
	log.Printf("[INFO] Starting get all quizzes")

	quizzes, err := s.repo.GetAllQuizzes()
	if err != nil {
		log.Printf("[ERROR] Failed to get all quizzes: %v", err)
		return nil, fmt.Errorf("failed to get quizzes: %w", err)
	}

	log.Printf("[INFO] Successfully retrieved %d quizzes", len(quizzes))
	return quizzes, nil
}

func (s *QuizStoreService) DeleteQuiz(id int) error {
	log.Printf("[INFO] Starting delete quiz with ID %d", id)

	if id <= 0 {
		log.Printf("[ERROR] Invalid quiz ID provided for deletion: %d", id)
		return fmt.Errorf("invalid quiz ID: %d", id)
	}

	if err := s.repo.DeleteQuiz(id); err != nil {
		log.Printf("[ERROR] Failed to delete quiz ID %d: %v", id, err)
		return err
	}

	log.Printf("[INFO] Successfully deleted quiz with ID %d", id)
	return nil
}

func (s *QuizStoreService) validateCreateRequest(req *models.CreateQuizRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if len(req.Config.Topics) == 0 {
		return fmt.Errorf("at least one topic is required")
	}

	if req.Config.QuestionCount <= 0 {
		return fmt.Errorf("question count must be greater than 0")
	}

	for i, topic := range req.Config.Topics {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			return fmt.Errorf("topic %d cannot be empty", i+1)
		}
		req.Config.Topics[i] = topic
	}

	return nil
}