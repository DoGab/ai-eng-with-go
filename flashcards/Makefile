.PHONY: help build run clean db-start db-stop db-up db-down db-reset

# Default target
help:
	@echo "Available commands:"
	@echo "  build     - Build the application"
	@echo "  run       - Run the application"
	@echo "  clean     - Clean build artifacts"
	@echo "  db-start  - Start Supabase local development"
	@echo "  db-stop   - Stop Supabase local development"
	@echo "  db-up     - Run database migrations"
	@echo "  db-down   - Rollback database migrations"
	@echo "  db-reset  - Reset database (stop, start, migrate)"

# Application commands
build:
	go build -o flashcards-api cmd/server/server.go

run:
	go run cmd/server/server.go

clean:
	rm -f flashcards-api

# Database commands
db-start:
	@echo "Starting Supabase local development..."
	supabase start

db-stop:
	@echo "Stopping Supabase local development..."
	supabase stop

db-up:
	@echo "Running database migrations..."
	supabase migration up
