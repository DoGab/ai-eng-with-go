# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Structure

This is a **Go AI agents learning repository** containing:

1. **Examples** (`examples/`) - Standalone Go applications demonstrating LangChain integration patterns
2. **Flashcards Application** (`flashcards/`) - Go REST API backend for flashcards and todos
3. **Flashcards Frontend** (`flashcards-app/`) - React frontend for flashcards
4. **Project Starter** (`prjstart/`) - Go API template project
5. **Learning Resources** (`scripts/booknotes/`) - Extensive book notes and documentation

## Common Development Commands

### Go Applications (examples/, flashcards/, prjstart/)
**Build and Run:**
- `go run main.go` - Run Go examples directly
- `go build -o <binary-name> cmd/main.go` - Build applications with cmd/ structure
- `make build` - Build using Makefile (flashcards/, prjstart/)
- `make run` - Run using Makefile
- `make clean` - Clean build artifacts

**Dependencies:**
- `go mod tidy` - Clean up module dependencies
- `go mod download` - Download dependencies

### React Frontend (flashcards-app/)
**Development:**
- `npm start` - Start development server on localhost:3000
- `npm test` - Run test suite
- `npm run build` - Build for production
- `npm install` - Install dependencies

### Database Operations (flashcards/, prjstart/)
**Supabase Local Development:**
- `make db-start` - Start local PostgreSQL via Supabase
- `make db-stop` - Stop Supabase environment
- `make db-up` - Run database migrations
- `make db-reset` - Complete database reset

## Architecture Overview

### AI Agent Examples Pattern
The `examples/` directory demonstrates three core LangChain patterns:

1. **Basic Usage** - Simple LLM prompting with OpenAI
2. **Function Calling** - Tool integration with structured JSON responses
3. **LLM Streaming** - Real-time response streaming via Server-Sent Events

**Key Dependencies:**
- `github.com/tmc/langchaingo` - Primary LLM integration library
- Models used: `gpt-4o-mini`, `gpt-4o`

### Full-Stack Applications
Both `flashcards/` and `prjstart/` follow **Clean Architecture** with identical patterns:

**Go Backend Architecture:**
```
cmd/main.go          # Entry point with dependency injection
models/              # Data structures and DTOs
handlers/            # HTTP request/response handling
services/            # Business logic and validation
db/                  # Repository pattern with PostgreSQL
config/              # Environment-based configuration
```

**Frontend Architecture (flashcards-app/):**
- **React 19** with Create React App
- **TailwindCSS** for styling
- **Axios** for API communication
- **React Hot Toast** for notifications
- **React Markdown** for content rendering

### Database Schema
- **PostgreSQL** with Supabase local development
- Schema prefix: `gocourse.`
- Migrations in `supabase/migrations/` with timestamp prefixes
- Access: localhost:54322 (DB), localhost:54323 (Supabase Studio)

## Development Workflow

**For Go Applications:**
1. Always run `go build` after changes to verify compilation
2. Follow logging standards with `[INFO]` and `[ERROR]` prefixes
3. Use repository interfaces for database abstraction
4. Register new routes in main.go following existing patterns

**For React Frontend:**
1. Use existing component patterns in `components/`
2. Follow TailwindCSS utility-first approach
3. Use axios for API calls via `api/` directory
4. Implement proper error handling with react-hot-toast

**Environment Setup:**
- Go applications require `.env` files with `DB_URL` and `PORT`
- React app runs on port 3000, APIs typically on 8080
- Supabase CLI and Docker required for database operations

## Key Patterns to Follow

**Adding New Entities (Go APIs):**
Follow the exact layered pattern: model → migration → repository → service → handler → route registration

**LangChain Integration:**
- Use `llms.GenerateFromSinglePrompt()` for simple prompts
- Implement `llms.Tool` structs for function calling
- Use `llms.WithStreamingFunc()` for real-time responses
- Always handle tool call parsing with proper error handling

**Error Handling:**
- Log at point of failure, avoid redundant logging up call stack
- Use structured logging with relevant context (IDs, counts, operation types)
- Return appropriate HTTP status codes in handlers