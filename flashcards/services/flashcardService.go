package services

import (
	"fmt"
	"strings"

	"flashcards/db"
	"flashcards/models"
)

type FlashcardService struct {
	repo db.FlashcardRepository
}

func NewFlashcardService(repo db.FlashcardRepository) *FlashcardService {
	return &FlashcardService{repo: repo}
}

func (s *FlashcardService) CreateFlashcard(req *models.CreateFlashcardRequest) (*models.Flashcard, error) {
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	flashcard := &models.Flashcard{
		Content: strings.TrimSpace(req.Content),
	}

	if err := s.repo.CreateFlashcard(flashcard); err != nil {
		return nil, fmt.Errorf("failed to create flashcard: %w", err)
	}

	return flashcard, nil
}

func (s *FlashcardService) GetFlashcardByID(id int) (*models.Flashcard, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid flashcard ID: %d", id)
	}

	flashcard, err := s.repo.GetFlashcardByID(id)
	if err != nil {
		return nil, err
	}

	return flashcard, nil
}

func (s *FlashcardService) GetAllFlashcards() ([]*models.Flashcard, error) {
	flashcards, err := s.repo.GetAllFlashcards()
	if err != nil {
		return nil, fmt.Errorf("failed to get flashcards: %w", err)
	}

	return flashcards, nil
}

func (s *FlashcardService) UpdateFlashcard(id int, req *models.UpdateFlashcardRequest) (*models.Flashcard, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid flashcard ID: %d", id)
	}

	if err := s.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	updates := make(map[string]any)

	if req.Content != nil {
		trimmedContent := strings.TrimSpace(*req.Content)
		if trimmedContent == "" {
			return nil, fmt.Errorf("content cannot be empty")
		}
		updates["content"] = trimmedContent
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no valid updates provided")
	}

	if err := s.repo.UpdateFlashcard(id, updates); err != nil {
		return nil, err
	}

	return s.repo.GetFlashcardByID(id)
}

func (s *FlashcardService) DeleteFlashcard(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid flashcard ID: %d", id)
	}

	return s.repo.DeleteFlashcard(id)
}

func (s *FlashcardService) validateCreateRequest(req *models.CreateFlashcardRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return fmt.Errorf("content is required")
	}

	return nil
}

func (s *FlashcardService) validateUpdateRequest(req *models.UpdateFlashcardRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.Content == nil {
		return fmt.Errorf("at least one field must be provided for update")
	}

	return nil
}