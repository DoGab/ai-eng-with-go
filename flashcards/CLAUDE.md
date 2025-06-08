# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

**Build and Run:**
- `go build ./cmd/main.go` - Build the application (after making changes, always run this to verify compilation)
- `make build` - Build with proper binary name (creates `todo-api`)
- `make run` - Run the application directly
- `make clean` - Clean build artifacts

**Database Operations:**
- `make db-start` - Start Supabase local PostgreSQL environment
- `make db-up` - Run database migrations
- `make db-stop` - Stop Supabase environment
- `make db-reset` - Complete database reset (stop, start, migrate)

## Architecture Overview

This is a **dual-purpose REST API** serving both **todos and flashcards** with identical architectural patterns.

**Clean Architecture Pattern:**
- `cmd/main.go` - Application entry point with dependency injection
- `models/` - Data structures and DTOs (todo.go, flashcard.go)
- `handlers/` - HTTP request/response handling (todoHandler.go, flashcardHandler.go) 
- `services/` - Business logic and validation (todoService.go, flashcardService.go)
- `db/` - Repository pattern with PostgreSQL implementation (todoDb.go, flashcardDb.go)
- `config/` - Environment-based configuration management

**Key Patterns:**
- Both todos and flashcards follow **identical architectural patterns**
- Repository interfaces for database abstraction
- Service layer handles validation and business logic
- Handlers manage HTTP concerns (JSON, status codes, error responses)
- Gorilla Mux for routing with pattern-based routes (`/todos/{id:[0-9]+}`, `/flashcards/{id:[0-9]+}`)

**Database:**
- PostgreSQL with schema `gocourse.todos` and `gocourse.flashcards`
- Migrations in `supabase/migrations/` with timestamp prefixes
- Local development via Supabase CLI (accessible at localhost:54322)

**Data Models:**
- **Todos**: ID, Title, Description, Completed, CreatedAt, UpdatedAt
- **Flashcards**: ID, Content, CreatedAt, UpdatedAt (minimal content-only design)

**When adding new entities**, follow the exact same layered pattern: model → migration → repository → service → handler → route registration in main.go.

## Environment Setup

Requires `.env` file with:
- `DB_URL` - PostgreSQL connection string
- `PORT` - Application port (defaults to 8080)

Both Docker and Supabase CLI must be installed and running for database operations.

## Logging Standards

**Required Logging Pattern for All Components:**
- **Operation Start**: Log at the beginning of each operation with relevant context
  ```go
  log.Printf("[INFO] Starting operation with %d items", count)
  ```
- **Operation Success**: Log when operations complete successfully
  ```go
  log.Printf("[INFO] Operation completed successfully, processed %d items", count)
  ```
- **Error Handling**: Always log errors at the point of failure, avoid redundant logging up the call stack
  ```go
  log.Printf("[ERROR] Operation failed: %v", err)
  ```

**Log Levels:**
- Use `[INFO]` for happy path scenarios and operation progress
- Use `[ERROR]` for error scenarios only
- Include relevant context (counts, IDs, operation types) in log messages

**Apply to All New Components:**
- Handlers: Log request start, completion, and any decode/validation errors
- Services: Log operation start, progress milestones, and successful completion
- Repositories: Log database operations and query results

## Development Workflow Guidance

- After each task is complete, go build the project to verify your work and then fix any issues which come up.