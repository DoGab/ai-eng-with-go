package db

import (
	"database/sql"
	"fmt"

	"flashcards/models"

	_ "github.com/lib/pq"
)

type FlashcardRepository interface {
	CreateFlashcard(flashcard *models.Flashcard) error
	GetFlashcardByID(id int) (*models.Flashcard, error)
	GetAllFlashcards() ([]*models.Flashcard, error)
	UpdateFlashcard(id int, updates map[string]any) error
	DeleteFlashcard(id int) error
}

type PostgresFlashcardRepository struct {
	db *sql.DB
}

func NewPostgresFlashcardRepository(databaseURL string) (*PostgresFlashcardRepository, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresFlashcardRepository{db: db}, nil
}

func (r *PostgresFlashcardRepository) CreateFlashcard(flashcard *models.Flashcard) error {
	query := `
		INSERT INTO gocourse.flashcards (content) 
		VALUES ($1) 
		RETURNING id, createdAt, updatedAt`

	row := r.db.QueryRow(query, flashcard.Content)

	err := row.Scan(&flashcard.ID, &flashcard.CreatedAt, &flashcard.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create flashcard: %w", err)
	}

	return nil
}

func (r *PostgresFlashcardRepository) GetFlashcardByID(id int) (*models.Flashcard, error) {
	query := `
		SELECT id, content, createdAt, updatedAt 
		FROM gocourse.flashcards 
		WHERE id = $1`

	flashcard := &models.Flashcard{}
	row := r.db.QueryRow(query, id)

	err := row.Scan(&flashcard.ID, &flashcard.Content, &flashcard.CreatedAt, &flashcard.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("flashcard with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get flashcard: %w", err)
	}

	return flashcard, nil
}

func (r *PostgresFlashcardRepository) GetAllFlashcards() ([]*models.Flashcard, error) {
	query := `
		SELECT id, content, createdAt, updatedAt 
		FROM gocourse.flashcards 
		ORDER BY createdAt DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query flashcards: %w", err)
	}
	defer rows.Close()

	flashcards := make([]*models.Flashcard, 0)
	for rows.Next() {
		flashcard := &models.Flashcard{}
		err := rows.Scan(&flashcard.ID, &flashcard.Content, &flashcard.CreatedAt, &flashcard.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan flashcard: %w", err)
		}
		flashcards = append(flashcards, flashcard)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over flashcards: %w", err)
	}

	return flashcards, nil
}

func (r *PostgresFlashcardRepository) UpdateFlashcard(id int, updates map[string]any) error {
	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}

	query := "UPDATE gocourse.flashcards SET "
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
		return fmt.Errorf("failed to update flashcard: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("flashcard with id %d not found", id)
	}

	return nil
}

func (r *PostgresFlashcardRepository) DeleteFlashcard(id int) error {
	query := "DELETE FROM gocourse.flashcards WHERE id = $1"

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete flashcard: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("flashcard with id %d not found", id)
	}

	return nil
}

func (r *PostgresFlashcardRepository) Close() error {
	return r.db.Close()
}