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
	log.Printf("[INFO] Starting note search with %d search terms: %v", len(searchTerms), searchTerms)

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

	log.Printf("[INFO] Found %d notes matching search criteria out of %d total notes", len(matchingNotes), len(notes))
	return matchingNotes, nil
}

func (s *NoteService) noteMatchesSearch(note *models.Note, searchTerms []string) bool {
	noteContent := strings.ToLower(note.Content)
	words := strings.Fields(noteContent)
	
	for _, term := range searchTerms {
		termLower := strings.ToLower(term)
		
		// 1. Exact substring match (highest priority)
		if strings.Contains(noteContent, termLower) {
			log.Printf("[DEBUG] Note %d: Exact match for term '%s'", note.ID, term)
			return true
		}
		
		// 2. Clean words by removing punctuation
		cleanWords := make([]string, 0, len(words))
		for _, word := range words {
			cleanWord := strings.Trim(word, ".,!?;:()[]{}\"'-")
			if len(cleanWord) > 2 { // Only consider words longer than 2 characters
				cleanWords = append(cleanWords, cleanWord)
			}
		}
		
		// 3. Use RankMatch with Levenshtein distance for stricter matching
		for _, cleanWord := range cleanWords {
			distance := fuzzy.RankMatchFold(termLower, cleanWord)
			
			// Only accept matches if:
			// - Distance is not -1 (means it's a valid match)
			// - Distance is within acceptable threshold based on term length
			if distance != -1 {
				maxDistance := s.calculateMaxDistance(termLower)
				if distance <= maxDistance {
					log.Printf("[DEBUG] Note %d: Fuzzy match for term '%s' -> word '%s' (distance: %d, max: %d)", 
						note.ID, term, cleanWord, distance, maxDistance)
					return true
				}
			}
		}
	}
	
	return false
}

func (s *NoteService) calculateMaxDistance(term string) int {
	termLen := len(term)
	
	// Very strict distance thresholds:
	if termLen <= 4 {
		return 1 // Very short terms: allow only 1 character difference
	} else if termLen <= 7 {
		return 2 // Medium terms: allow 2 character differences
	} else {
		return 3 // Longer terms: allow 3 character differences
	}
}