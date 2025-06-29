package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"flashcards/config"
	"flashcards/db"
	"flashcards/models"
	"flashcards/services"

	"github.com/pinecone-io/go-pinecone/v3/pinecone"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"google.golang.org/protobuf/types/known/structpb"
)

type DocumentChunk struct {
	ID              string
	NoteID          int
	ChunkIndex      int
	Heading         string
	HeadingPath     []string // Tree of parent headings leading to this chunk
	Content         string
	OriginalNote    string
	EnrichedContext string
}

type EnrichChunkContextParams struct {
	EnrichedSummary string `json:"enriched_summary"`
}

var enrichmentTools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "enrich_chunk_context",
			Description: "Provide an enriched contextual summary for a document chunk",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enriched_summary": map[string]any{
						"type":        "string",
						"description": "A comprehensive summary that explains what this chunk is about, its context within the larger document, and why it's relevant. This should be self-contained so someone reading just this summary would understand the content and its significance.",
					},
				},
				"required": []string{"enriched_summary"},
			},
		},
	},
}

func main() {
	log.Printf("[INFO] Starting document indexing process")

	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		log.Fatal("[ERROR] DB_URL environment variable is required")
	}

	pineconeAPIKey := os.Getenv("PINECONE_API_KEY")
	if pineconeAPIKey == "" {
		log.Fatal("[ERROR] PINECONE_API_KEY environment variable is required")
	}

	if cfg.OpenAIAPIKey == "" {
		log.Fatal("[ERROR] OPENAI_API_KEY environment variable is required")
	}

	noteRepo, err := db.NewPostgresNoteRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[ERROR] Failed to initialize note database: %v", err)
	}
	defer noteRepo.Close()

	noteService := services.NewNoteService(noteRepo)

	llm, err := openai.New(
		openai.WithModel("gpt-4o-mini"),
		openai.WithToken(cfg.OpenAIAPIKey),
	)
	if err != nil {
		log.Fatalf("[ERROR] Failed to create OpenAI client: %v", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatalf("[ERROR] Failed to create embedder: %v", err)
	}

	pc, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: pineconeAPIKey,
	})
	if err != nil {
		log.Fatalf("[ERROR] Failed to create Pinecone client: %v", err)
	}

	indexName := "flashcards-notes-index"
	if err := ensurePineconeIndex(pc, indexName); err != nil {
		log.Fatalf("[ERROR] Failed to ensure Pinecone index: %v", err)
	}

	notes, err := noteService.GetAllNotes()
	if err != nil {
		log.Fatalf("[ERROR] Failed to retrieve notes: %v", err)
	}

	log.Printf("[INFO] Retrieved %d notes from database", len(notes))

	for i, note := range notes {
		log.Printf("[INFO] Processing note %d/%d (ID: %d)", i+1, len(notes), note.ID)

		if err := processNote(pc, indexName, note, llm, embedder); err != nil {
			log.Printf("[ERROR] Failed to process note ID %d: %v", note.ID, err)
			continue
		}

		log.Printf("[INFO] Successfully processed note ID %d", note.ID)
	}

	log.Printf("[INFO] Document indexing process completed successfully")
}

func ensurePineconeIndex(pc *pinecone.Client, indexName string) error {
	ctx := context.Background()

	indexes, err := pc.ListIndexes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list indexes: %w", err)
	}

	for _, idx := range indexes {
		if idx.Name == indexName {
			log.Printf("[INFO] Index %s already exists", indexName)
			return nil
		}
	}

	log.Printf("[INFO] Creating Pinecone index: %s", indexName)
	dimension := int32(1536) // OpenAI ada-002 embedding dimension
	deletionProtection := pinecone.DeletionProtectionDisabled
	metric := pinecone.Cosine

	_, err = pc.CreateServerlessIndex(ctx, &pinecone.CreateServerlessIndexRequest{
		Name:               indexName,
		Dimension:          &dimension,
		Metric:             &metric,
		Cloud:              pinecone.Aws,
		Region:             "us-east-1",
		DeletionProtection: &deletionProtection,
		Tags:               &pinecone.IndexTags{"environment": "development", "project": "flashcards-indexing"},
	})
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	for {
		idx, err := pc.DescribeIndex(ctx, indexName)
		if err != nil {
			return fmt.Errorf("failed to describe index: %w", err)
		}
		if idx.Status.Ready {
			log.Printf("[INFO] Index %s is ready", indexName)
			break
		}
		log.Printf("[INFO] Waiting for index %s to be ready...", indexName)
		time.Sleep(10 * time.Second)
	}

	return nil
}

func processNote(pc *pinecone.Client, indexName string, note *models.Note, llm llms.Model, embedder embeddings.Embedder) error {
	log.Printf("[INFO] Chunking note ID %d", note.ID)
	chunks := chunkMarkdownByHeadings(note)
	if len(chunks) == 0 {
		log.Printf("[INFO] No chunks created for note ID %d", note.ID)
		return nil
	}
	log.Printf("[INFO] Created %d chunks for note ID %d", len(chunks), note.ID)

	log.Printf("[INFO] Deleting existing vectors for note ID %d", note.ID)
	if err := deleteExistingVectors(pc, indexName, note.ID); err != nil {
		return fmt.Errorf("failed to delete existing vectors: %w", err)
	}

	log.Printf("[INFO] Processing and upserting chunks individually for note ID %d", note.ID)
	for i := range chunks {
		headingInfo := chunks[i].Heading
		if len(chunks[i].HeadingPath) > 0 {
			headingInfo = fmt.Sprintf("%s [Path: %s]", chunks[i].Heading, strings.Join(chunks[i].HeadingPath, " → "))
		}
		log.Printf("[INFO] Processing chunk %d/%d for note ID %d (Heading: %s)", i+1, len(chunks), note.ID, headingInfo)
		
		// Enrich the chunk
		enrichedContext, err := enrichChunkContext(llm, chunks[i])
		if err != nil {
			log.Printf("[ERROR] Failed to enrich chunk %d for note ID %d: %v", i+1, note.ID, err)
			log.Printf("[INFO] Using fallback content for chunk %d of note ID %d", i+1, note.ID)
			chunks[i].EnrichedContext = chunks[i].Content // Fallback to original content
		} else {
			log.Printf("[INFO] Successfully enriched chunk %d for note ID %d", i+1, note.ID)
			chunks[i].EnrichedContext = enrichedContext
		}

		// Generate embedding and upsert immediately
		log.Printf("[INFO] Generating embedding for chunk %d", i+1)
		vector, err := createSingleVector(chunks[i], embedder)
		if err != nil {
			log.Printf("[ERROR] Failed to create vector for chunk %d: %v", i+1, err)
			return fmt.Errorf("failed to create vector for chunk %d: %w", i+1, err)
		}

		log.Printf("[INFO] Upserting chunk %d to Pinecone", i+1)
		if err := upsertSingleVector(pc, indexName, vector); err != nil {
			log.Printf("[ERROR] Failed to upsert chunk %d: %v", i+1, err)
			return fmt.Errorf("failed to upsert chunk %d: %w", i+1, err)
		}
		log.Printf("[INFO] Successfully upserted chunk %d for note ID %d", i+1, note.ID)
	}
	log.Printf("[INFO] Completed processing all %d chunks for note ID %d", len(chunks), note.ID)

	return nil
}

func chunkMarkdownByHeadings(note *models.Note) []DocumentChunk {
	content := note.Content
	lines := strings.Split(content, "\n")

	var chunks []DocumentChunk
	var currentChunk strings.Builder
	var currentHeading string
	var headingStack []string // Stack to track heading hierarchy
	chunkIndex := 0

	headingRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

	for _, line := range lines {
		if match := headingRegex.FindStringSubmatch(line); match != nil {
			// Save previous chunk if it exists
			if currentChunk.Len() > 0 {
				chunkContent := strings.TrimSpace(currentChunk.String())
				if chunkContent != "" {
					chunks = append(chunks, DocumentChunk{
						ID:           fmt.Sprintf("note_%d_chunk_%d", note.ID, chunkIndex),
						NoteID:       note.ID,
						ChunkIndex:   chunkIndex,
						Heading:      currentHeading,
						HeadingPath:  make([]string, len(headingStack)),
						Content:      chunkContent,
						OriginalNote: content,
					})
					copy(chunks[len(chunks)-1].HeadingPath, headingStack)
					chunkIndex++
				}
				currentChunk.Reset()
			}

			// Update heading hierarchy
			headingLevel := len(match[1]) // Number of # characters
			currentHeading = match[2]

			// Adjust heading stack based on current level
			if headingLevel <= len(headingStack) {
				// We're at same level or going up - truncate stack
				headingStack = headingStack[:headingLevel-1]
			}
			// Add current heading to stack
			headingStack = append(headingStack, currentHeading)

			currentChunk.WriteString(line + "\n")
		} else {
			currentChunk.WriteString(line + "\n")
		}
	}

	// Handle final chunk
	if currentChunk.Len() > 0 {
		chunkContent := strings.TrimSpace(currentChunk.String())
		if chunkContent != "" {
			chunks = append(chunks, DocumentChunk{
				ID:           fmt.Sprintf("note_%d_chunk_%d", note.ID, chunkIndex),
				NoteID:       note.ID,
				ChunkIndex:   chunkIndex,
				Heading:      currentHeading,
				HeadingPath:  make([]string, len(headingStack)),
				Content:      chunkContent,
				OriginalNote: content,
			})
			copy(chunks[len(chunks)-1].HeadingPath, headingStack)
		}
	}

	// Handle document with no headings
	if len(chunks) == 0 && strings.TrimSpace(content) != "" {
		chunks = append(chunks, DocumentChunk{
			ID:           fmt.Sprintf("note_%d_chunk_0", note.ID),
			NoteID:       note.ID,
			ChunkIndex:   0,
			Heading:      "Document Content",
			HeadingPath:  []string{},
			Content:      content,
			OriginalNote: content,
		})
	}

	return chunks
}

func enrichChunkContext(llm llms.Model, chunk DocumentChunk) (string, error) {
	ctx := context.Background()

	log.Printf("[INFO] Starting LLM enrichment for chunk: %s (Note ID: %d, Chunk: %d)", chunk.Heading, chunk.NoteID, chunk.ChunkIndex)

	systemPrompt := `You are an expert at analyzing document chunks and providing enriched contextual summaries. 

Your task is to create a comprehensive summary that:
1. Explains what this specific chunk covers
2. Provides context about how it fits within the larger document
3. Highlights why this information is relevant or important
4. Makes the chunk self-contained and searchable

The enriched summary should help someone understand the chunk's content and significance without needing to read the entire original document.`

	headingPathStr := ""
	if len(chunk.HeadingPath) > 0 {
		headingPathStr = fmt.Sprintf("Section hierarchy: %s", strings.Join(chunk.HeadingPath, " → "))
	}

	userPrompt := fmt.Sprintf(`Please analyze this document chunk and create an enriched contextual summary.

CHUNK TO ANALYZE:
Heading: %s
%s
Content: %s

FULL DOCUMENT CONTEXT:
%s

Create a comprehensive summary that explains what this chunk is about, its context within the larger document, and why it's relevant. Pay special attention to the section hierarchy to understand how this content fits within the document structure.`,
		chunk.Heading, headingPathStr, chunk.Content, chunk.OriginalNote)

	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	log.Printf("[INFO] Calling LLM for enrichment (chunk content length: %d chars)", len(chunk.Content))
	resp, err := llm.GenerateContent(ctx, messageHistory,
		llms.WithTools(enrichmentTools),
		llms.WithTemperature(0.3),
		llms.WithToolChoice("required"))
	if err != nil {
		log.Printf("[ERROR] LLM call failed for chunk %s: %v", chunk.Heading, err)
		return "", fmt.Errorf("failed to generate enrichment: %w", err)
	}

	if len(resp.Choices) == 0 || len(resp.Choices[0].ToolCalls) == 0 {
		return "", fmt.Errorf("no tool calls in enrichment response")
	}

	toolCall := resp.Choices[0].ToolCalls[0]
	if toolCall.FunctionCall.Name != "enrich_chunk_context" {
		return "", fmt.Errorf("unexpected function call: %s", toolCall.FunctionCall.Name)
	}

	var params EnrichChunkContextParams
	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &params); err != nil {
		return "", fmt.Errorf("failed to parse enrichment arguments: %w", err)
	}

	log.Printf("[INFO] LLM enrichment completed for chunk %s (enriched summary length: %d chars)", chunk.Heading, len(params.EnrichedSummary))
	return params.EnrichedSummary, nil
}

func deleteExistingVectors(pc *pinecone.Client, indexName string, noteID int) error {
	ctx := context.Background()

	idxDesc, err := pc.DescribeIndex(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to describe index: %w", err)
	}

	idxConn, err := pc.Index(pinecone.NewIndexConnParams{
		Host:      idxDesc.Host,
		Namespace: "flashcards-docs",
	})
	if err != nil {
		return fmt.Errorf("failed to create index connection: %w", err)
	}

	// First, try to list vectors to see if there are any to delete
	log.Printf("[INFO] Checking for existing vectors for note ID %d", noteID)
	prefix := fmt.Sprintf("note_%d_", noteID)
	limit := uint32(100)
	
	listResp, err := idxConn.ListVectors(ctx, &pinecone.ListVectorsRequest{
		Prefix: &prefix,
		Limit:  &limit,
	})
	if err != nil {
		// If namespace doesn't exist, that's fine - no vectors to delete
		if strings.Contains(err.Error(), "Namespace not found") {
			log.Printf("[INFO] Namespace does not exist yet - no vectors to delete for note ID %d", noteID)
			return nil
		}
		return fmt.Errorf("failed to list vectors: %w", err)
	}

	if len(listResp.VectorIds) == 0 {
		log.Printf("[INFO] No existing vectors found for note ID %d", noteID)
		return nil
	}

	log.Printf("[INFO] Found %d existing vectors for note ID %d, deleting them", len(listResp.VectorIds), noteID)

	// Delete all vectors with this prefix using list and delete by ID approach
	for listResp.NextPaginationToken != nil || len(listResp.VectorIds) > 0 {
		vectorIdsToDelete := make([]string, 0, len(listResp.VectorIds))
		for _, vectorId := range listResp.VectorIds {
			if vectorId != nil {
				vectorIdsToDelete = append(vectorIdsToDelete, *vectorId)
			}
		}

		if len(vectorIdsToDelete) > 0 {
			err = idxConn.DeleteVectorsById(ctx, vectorIdsToDelete)
			if err != nil {
				return fmt.Errorf("failed to delete vector batch: %w", err)
			}
			log.Printf("[INFO] Deleted %d vectors for note ID %d", len(vectorIdsToDelete), noteID)
		}

		// Get next batch if there's a pagination token
		if listResp.NextPaginationToken != nil {
			listResp, err = idxConn.ListVectors(ctx, &pinecone.ListVectorsRequest{
				Prefix:          &prefix,
				Limit:           &limit,
				PaginationToken: listResp.NextPaginationToken,
			})
			if err != nil {
				return fmt.Errorf("failed to list next batch of vectors: %w", err)
			}
		} else {
			break
		}
	}

	return nil
}

func createSingleVector(chunk DocumentChunk, embedder embeddings.Embedder) (*pinecone.Vector, error) {
	ctx := context.Background()

	combinedText := fmt.Sprintf("Heading: %s\n\nContent: %s\n\nContext: %s", 
		chunk.Heading, chunk.Content, chunk.EnrichedContext)

	embeddings, err := embedder.EmbedDocuments(ctx, []string{combinedText})
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	metadata := map[string]any{
		"note_id":          chunk.NoteID,
		"chunk_index":      chunk.ChunkIndex,
		"heading":          chunk.Heading,
		"heading_path":     strings.Join(chunk.HeadingPath, " → "),
		"content":          chunk.Content,
		"enriched_context": chunk.EnrichedContext,
		"created_at":       time.Now().Format(time.RFC3339),
	}

	metadataStruct, err := structpb.NewStruct(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata struct for chunk %s: %w", chunk.ID, err)
	}

	vector := &pinecone.Vector{
		Id:       chunk.ID,
		Values:   &embeddings[0],
		Metadata: metadataStruct,
	}

	return vector, nil
}

func upsertSingleVector(pc *pinecone.Client, indexName string, vector *pinecone.Vector) error {
	ctx := context.Background()

	idxDesc, err := pc.DescribeIndex(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to describe index: %w", err)
	}

	idxConn, err := pc.Index(pinecone.NewIndexConnParams{
		Host:      idxDesc.Host,
		Namespace: "flashcards-docs",
	})
	if err != nil {
		return fmt.Errorf("failed to create index connection: %w", err)
	}

	_, err = idxConn.UpsertVectors(ctx, []*pinecone.Vector{vector})
	if err != nil {
		return fmt.Errorf("failed to upsert vector: %w", err)
	}

	return nil
}

func createVectors(chunks []DocumentChunk, embedder embeddings.Embedder) ([]*pinecone.Vector, error) {
	ctx := context.Background()

	var texts []string
	for _, chunk := range chunks {
		combinedText := fmt.Sprintf("Heading: %s\n\nContent: %s\n\nContext: %s",
			chunk.Heading, chunk.Content, chunk.EnrichedContext)
		texts = append(texts, combinedText)
	}

	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	var vectors []*pinecone.Vector
	for i, chunk := range chunks {
		metadata := map[string]any{
			"note_id":          chunk.NoteID,
			"chunk_index":      chunk.ChunkIndex,
			"heading":          chunk.Heading,
			"heading_path":     strings.Join(chunk.HeadingPath, " → "),
			"content":          chunk.Content,
			"enriched_context": chunk.EnrichedContext,
			"created_at":       time.Now().Format(time.RFC3339),
		}

		metadataStruct, err := structpb.NewStruct(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to create metadata struct for chunk %s: %w", chunk.ID, err)
		}

		vector := &pinecone.Vector{
			Id:       chunk.ID,
			Values:   &embeddings[i],
			Metadata: metadataStruct,
		}
		vectors = append(vectors, vector)
	}

	return vectors, nil
}

func upsertVectors(pc *pinecone.Client, indexName string, vectors []*pinecone.Vector) error {
	ctx := context.Background()

	idxDesc, err := pc.DescribeIndex(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to describe index: %w", err)
	}

	idxConn, err := pc.Index(pinecone.NewIndexConnParams{
		Host:      idxDesc.Host,
		Namespace: "flashcards-docs",
	})
	if err != nil {
		return fmt.Errorf("failed to create index connection: %w", err)
	}

	batchSize := 10
	for i := 0; i < len(vectors); i += batchSize {
		end := i + batchSize
		if end > len(vectors) {
			end = len(vectors)
		}

		batch := vectors[i:end]
		count, err := idxConn.UpsertVectors(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to upsert vector batch: %w", err)
		}
		log.Printf("[INFO] Successfully upserted %d vectors (batch %d)", count, i/batchSize+1)
	}

	return nil
}
