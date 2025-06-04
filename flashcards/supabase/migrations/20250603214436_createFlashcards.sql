CREATE SCHEMA IF NOT EXISTS gocourse;

CREATE TABLE IF NOT EXISTS gocourse.flashcards (
    id SERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    createdAt TIMESTAMP DEFAULT NOW(),
    updatedAt TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_flashcards_created_at ON gocourse.flashcards(createdAt);
