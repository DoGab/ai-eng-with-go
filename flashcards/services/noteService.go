package services

import (
	"fmt"
	"strings"

	"flashcards/db"
	"flashcards/models"
)

type NoteService struct {
	repo db.NoteRepository
}

func NewNoteService(repo db.NoteRepository) *NoteService {
	return &NoteService{repo: repo}
}

func (s *NoteService) CreateNote(req *models.CreateNoteRequest) (*models.Note, error) {
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	note := &models.Note{
		Content: strings.TrimSpace(req.Content),
	}

	if err := s.repo.CreateNote(note); err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	return note, nil
}

func (s *NoteService) GetNoteByID(id int) (*models.Note, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid note ID: %d", id)
	}

	note, err := s.repo.GetNoteByID(id)
	if err != nil {
		return nil, err
	}

	return note, nil
}

func (s *NoteService) GetAllNotes() ([]*models.Note, error) {
	notes, err := s.repo.GetAllNotes()
	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}

	return notes, nil
}

func (s *NoteService) UpdateNote(id int, req *models.UpdateNoteRequest) (*models.Note, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid note ID: %d", id)
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

	if err := s.repo.UpdateNote(id, updates); err != nil {
		return nil, err
	}

	return s.repo.GetNoteByID(id)
}

func (s *NoteService) DeleteNote(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid note ID: %d", id)
	}

	return s.repo.DeleteNote(id)
}

func (s *NoteService) validateCreateRequest(req *models.CreateNoteRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return fmt.Errorf("content is required")
	}

	return nil
}

func (s *NoteService) validateUpdateRequest(req *models.UpdateNoteRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.Content == nil {
		return fmt.Errorf("at least one field must be provided for update")
	}

	return nil
}