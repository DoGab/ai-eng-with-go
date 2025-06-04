package models

import "time"

type Flashcard struct {
	ID        int       `json:"id" db:"id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" db:"updatedAt"`
}

type CreateFlashcardRequest struct {
	Content string `json:"content"`
}

type UpdateFlashcardRequest struct {
	Content *string `json:"content,omitempty"`
}