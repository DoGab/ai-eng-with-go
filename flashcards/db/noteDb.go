package db

import (
	"database/sql"
	"fmt"

	"flashcards/models"

	_ "github.com/lib/pq"
)

type NoteRepository interface {
	CreateNote(note *models.Note) error
	GetNoteByID(id int) (*models.Note, error)
	GetAllNotes() ([]*models.Note, error)
	UpdateNote(id int, updates map[string]any) error
	DeleteNote(id int) error
}

type PostgresNoteRepository struct {
	db *sql.DB
}

func NewPostgresNoteRepository(databaseURL string) (*PostgresNoteRepository, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresNoteRepository{db: db}, nil
}

func (r *PostgresNoteRepository) CreateNote(note *models.Note) error {
	query := `
		INSERT INTO gocourse.notes (content) 
		VALUES ($1) 
		RETURNING id, createdAt, updatedAt`

	row := r.db.QueryRow(query, note.Content)

	err := row.Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}

	return nil
}

func (r *PostgresNoteRepository) GetNoteByID(id int) (*models.Note, error) {
	query := `
		SELECT id, content, createdAt, updatedAt 
		FROM gocourse.notes 
		WHERE id = $1`

	note := &models.Note{}
	row := r.db.QueryRow(query, id)

	err := row.Scan(&note.ID, &note.Content, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("note with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	return note, nil
}

func (r *PostgresNoteRepository) GetAllNotes() ([]*models.Note, error) {
	query := `
		SELECT id, content, createdAt, updatedAt 
		FROM gocourse.notes 
		ORDER BY createdAt DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query notes: %w", err)
	}
	defer rows.Close()

	notes := make([]*models.Note, 0)
	for rows.Next() {
		note := &models.Note{}
		err := rows.Scan(&note.ID, &note.Content, &note.CreatedAt, &note.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over notes: %w", err)
	}

	return notes, nil
}

func (r *PostgresNoteRepository) UpdateNote(id int, updates map[string]any) error {
	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}

	query := "UPDATE gocourse.notes SET "
	args := []any{}
	argIndex := 1

	for field, value := range updates {
		if argIndex > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", field, argIndex)
		args = append(args, value)
		argIndex++
	}

	query += fmt.Sprintf(", updatedAt = NOW() WHERE id = $%d", argIndex)
	args = append(args, id)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("note with id %d not found", id)
	}

	return nil
}

func (r *PostgresNoteRepository) DeleteNote(id int) error {
	query := "DELETE FROM gocourse.notes WHERE id = $1"

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("note with id %d not found", id)
	}

	return nil
}

func (r *PostgresNoteRepository) Close() error {
	return r.db.Close()
}