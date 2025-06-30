-- Add asked_questions field to quizzes table
ALTER TABLE gocourse.quizzes 
ADD COLUMN asked_questions JSONB DEFAULT '[]'::jsonb;

-- Add index for faster querying on asked_questions
CREATE INDEX idx_quizzes_asked_questions ON gocourse.quizzes USING gin(asked_questions);