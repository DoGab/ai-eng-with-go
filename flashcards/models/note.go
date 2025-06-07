package models

import "time"

type Note struct {
	ID        int       `json:"id" db:"id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" db:"updatedAt"`
}

type CreateNoteRequest struct {
	Content string `json:"content"`
}

type UpdateNoteRequest struct {
	Content *string `json:"content,omitempty"`
}