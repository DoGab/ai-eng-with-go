package services

import (
	"fmt"
	"log"
	"strings"

	"flashcards/db"
	"flashcards/models"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

type NoteService struct {
	repo db.NoteRepository
}

func NewNoteService(repo db.NoteRepository) *NoteService {
	return &NoteService{repo: repo}
}

func (s *NoteService) CreateNote(req *models.CreateNoteRequest) (*models.Note, error) {
	log.Printf("[INFO] Starting note creation")

	if err := s.validateCreateRequest(req); err != nil {
		log.Printf("[ERROR] Note creation validation failed: %v", err)
		return nil, err
	}

	note := &models.Note{
		Content: strings.TrimSpace(req.Content),
	}

	if err := s.repo.CreateNote(note); err != nil {
		log.Printf("[ERROR] Failed to create note in repository: %v", err)
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	log.Printf("[INFO] Successfully created note with ID %d", note.ID)
	return note, nil
}

func (s *NoteService) GetNoteByID(id int) (*models.Note, error) {
	log.Printf("[INFO] Starting get note by ID %d", id)

	if id <= 0 {
		log.Printf("[ERROR] Invalid note ID provided: %d", id)
		return nil, fmt.Errorf("invalid note ID: %d", id)
	}

	note, err := s.repo.GetNoteByID(id)
	if err != nil {
		log.Printf("[ERROR] Failed to get note by ID %d: %v", id, err)
		return nil, err
	}

	log.Printf("[INFO] Successfully retrieved note with ID %d", id)
	return note, nil
}

func (s *NoteService) GetAllNotes() ([]*models.Note, error) {
	log.Printf("[INFO] Starting get all notes")

	notes, err := s.repo.GetAllNotes()
	if err != nil {
		log.Printf("[ERROR] Failed to get all notes: %v", err)
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}

	log.Printf("[INFO] Successfully retrieved %d notes", len(notes))
	return notes, nil
}

func (s *NoteService) UpdateNote(id int, req *models.UpdateNoteRequest) (*models.Note, error) {
	log.Printf("[INFO] Starting update note with ID %d", id)

	if id <= 0 {
		log.Printf("[ERROR] Invalid note ID provided for update: %d", id)
		return nil, fmt.Errorf("invalid note ID: %d", id)
	}

	if err := s.validateUpdateRequest(req); err != nil {
		log.Printf("[ERROR] Note update validation failed for ID %d: %v", id, err)
		return nil, err
	}

	updates := make(map[string]any)

	if req.Content != nil {
		trimmedContent := strings.TrimSpace(*req.Content)
		if trimmedContent == "" {
			log.Printf("[ERROR] Empty content provided for note ID %d", id)
			return nil, fmt.Errorf("content cannot be empty")
		}
		updates["content"] = trimmedContent
	}

	if len(updates) == 0 {
		log.Printf("[ERROR] No valid updates provided for note ID %d", id)
		return nil, fmt.Errorf("no valid updates provided")
	}

	if err := s.repo.UpdateNote(id, updates); err != nil {
		log.Printf("[ERROR] Failed to update note ID %d in repository: %v", id, err)
		return nil, err
	}

	log.Printf("[INFO] Successfully updated note with ID %d", id)
	return s.repo.GetNoteByID(id)
}

func (s *NoteService) DeleteNote(id int) error {
	log.Printf("[INFO] Starting delete note with ID %d", id)

	if id <= 0 {
		log.Printf("[ERROR] Invalid note ID provided for deletion: %d", id)
		return fmt.Errorf("invalid note ID: %d", id)
	}

	if err := s.repo.DeleteNote(id); err != nil {
		log.Printf("[ERROR] Failed to delete note ID %d: %v", id, err)
		return err
	}

	log.Printf("[INFO] Successfully deleted note with ID %d", id)
	return nil
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

func (s *NoteService) SearchNotesByContent(searchTerms []string) ([]*models.Note, error) {
	log.Printf("[INFO] Starting note search with %d search terms", len(searchTerms))

	notes, err := s.GetAllNotes()
	if err != nil {
		return nil, fmt.Errorf("failed to get notes for search: %w", err)
	}

	if len(searchTerms) == 0 {
		log.Printf("[INFO] No search terms provided, returning all %d notes", len(notes))
		return notes, nil
	}

	var matchingNotes []*models.Note
	for _, note := range notes {
		if s.noteMatchesSearch(note, searchTerms) {
			matchingNotes = append(matchingNotes, note)
		}
	}

	log.Printf("[INFO] Found %d notes matching search criteria", len(matchingNotes))
	return matchingNotes, nil
}

func (s *NoteService) noteMatchesSearch(note *models.Note, searchTerms []string) bool {
	noteContent := note.Content
	words := strings.Fields(strings.ToLower(noteContent))
	
	for _, term := range searchTerms {
		// Exact match (highest priority)
		if fuzzy.MatchFold(term, noteContent) {
			return true
		}
		
		// Clean words by removing punctuation
		cleanWords := make([]string, 0, len(words))
		for _, word := range words {
			cleanWord := strings.Trim(word, ".,!?;:()[]{}\"'")
			if len(cleanWord) > 0 {
				cleanWords = append(cleanWords, cleanWord)
			}
		}
		
		// Use fuzzy search against individual words
		matches := fuzzy.Find(term, cleanWords)
		if len(matches) > 0 {
			return true
		}
		
		// Also check against the full content for partial matches
		if len(term) > 2 && fuzzy.MatchFold(term, noteContent) {
			return true
		}
	}
	
	return false
}