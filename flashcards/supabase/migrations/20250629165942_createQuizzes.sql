CREATE TABLE IF NOT EXISTS gocourse.quizzes (
    id SERIAL PRIMARY KEY,
    config JSONB NOT NULL,
    llm_context TEXT NOT NULL,
    createdAt TIMESTAMP DEFAULT NOW(),
    updatedAt TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quizzes_created_at ON gocourse.quizzes(createdAt);