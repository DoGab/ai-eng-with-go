package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"flashcards/models"

	_ "github.com/lib/pq"
)

type QuizRepository interface {
	CreateQuiz(quiz *models.Quiz) error
	GetQuizByID(id int) (*models.Quiz, error)
	GetAllQuizzes() ([]*models.Quiz, error)
	DeleteQuiz(id int) error
}

type PostgresQuizRepository struct {
	db *sql.DB
}

func NewPostgresQuizRepository(databaseURL string) (*PostgresQuizRepository, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresQuizRepository{db: db}, nil
}

func (r *PostgresQuizRepository) CreateQuiz(quiz *models.Quiz) error {
	configJSON, err := json.Marshal(quiz.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO gocourse.quizzes (config, llm_context) 
		VALUES ($1, $2) 
		RETURNING id, createdAt, updatedAt`

	row := r.db.QueryRow(query, configJSON, quiz.LLMContext)

	err = row.Scan(&quiz.ID, &quiz.CreatedAt, &quiz.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create quiz: %w", err)
	}

	return nil
}

func (r *PostgresQuizRepository) GetQuizByID(id int) (*models.Quiz, error) {
	query := `
		SELECT id, config, llm_context, createdAt, updatedAt 
		FROM gocourse.quizzes 
		WHERE id = $1`

	quiz := &models.Quiz{}
	var configJSON []byte
	row := r.db.QueryRow(query, id)

	err := row.Scan(&quiz.ID, &configJSON, &quiz.LLMContext, &quiz.CreatedAt, &quiz.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quiz with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get quiz: %w", err)
	}

	if err := json.Unmarshal(configJSON, &quiz.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return quiz, nil
}

func (r *PostgresQuizRepository) GetAllQuizzes() ([]*models.Quiz, error) {
	query := `
		SELECT id, config, llm_context, createdAt, updatedAt 
		FROM gocourse.quizzes 
		ORDER BY createdAt DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query quizzes: %w", err)
	}
	defer rows.Close()

	quizzes := make([]*models.Quiz, 0)
	for rows.Next() {
		quiz := &models.Quiz{}
		var configJSON []byte
		err := rows.Scan(&quiz.ID, &configJSON, &quiz.LLMContext, &quiz.CreatedAt, &quiz.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quiz: %w", err)
		}

		if err := json.Unmarshal(configJSON, &quiz.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		quizzes = append(quizzes, quiz)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over quizzes: %w", err)
	}

	return quizzes, nil
}

func (r *PostgresQuizRepository) DeleteQuiz(id int) error {
	query := "DELETE FROM gocourse.quizzes WHERE id = $1"

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete quiz: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("quiz with id %d not found", id)
	}

	return nil
}

func (r *PostgresQuizRepository) Close() error {
	return r.db.Close()
}