package pinecone

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/pinecone-io/go-pinecone/v3/pinecone"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

type Service struct {
	client    *pinecone.Client
	embedder  embeddings.Embedder
	indexName string
}

func NewService(apiKey, openaiAPIKey string) (*Service, error) {
	log.Printf("[INFO] Initializing Pinecone service")

	pc, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Pinecone client: %w", err)
	}

	llm, err := openai.New(
		openai.WithModel("gpt-4o-mini"),
		openai.WithToken(openaiAPIKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	service := &Service{
		client:    pc,
		embedder:  embedder,
		indexName: "flashcards-notes-index",
	}

	log.Printf("[INFO] Pinecone service initialized successfully")
	return service, nil
}

func (s *Service) QueryTopicChunks(topics []string, limit int) ([]string, error) {
	log.Printf("[INFO] Starting Pinecone query for topics: %v with limit: %d", topics, limit)

	ctx := context.Background()

	idxDesc, err := s.client.DescribeIndex(ctx, s.indexName)
	if err != nil {
		return nil, fmt.Errorf("failed to describe index: %w", err)
	}

	idxConn, err := s.client.Index(pinecone.NewIndexConnParams{
		Host:      idxDesc.Host,
		Namespace: "flashcards-docs",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create index connection: %w", err)
	}

	var allChunks []string

	for _, topic := range topics {
		log.Printf("[INFO] Querying topic: %s", topic)

		queryEmbeddings, err := s.embedder.EmbedDocuments(ctx, []string{topic})
		if err != nil {
			log.Printf("[ERROR] Failed to generate embedding for topic '%s': %v", topic, err)
			continue
		}

		result, err := idxConn.QueryByVectorValues(ctx, &pinecone.QueryByVectorValuesRequest{
			Vector:          queryEmbeddings[0],
			TopK:            20,
			IncludeValues:   false,
			IncludeMetadata: true,
		})
		if err != nil {
			log.Printf("[ERROR] Failed to query vectors for topic '%s': %v", topic, err)
			continue
		}

		log.Printf("[INFO] Retrieved %d chunks for topic '%s'", len(result.Matches), topic)

		for _, match := range result.Matches {
			if match.Vector.Metadata != nil {
				metadata := match.Vector.Metadata.AsMap()

				var chunkParts []string

				// Add heading information if available
				if heading, ok := metadata["heading"].(string); ok && heading != "" {
					headingInfo := "Section: " + heading
					if headingPath, ok := metadata["heading_path"].(string); ok && headingPath != "" {
						headingInfo += " (Path: " + headingPath + ")"
					}
					chunkParts = append(chunkParts, headingInfo)
				}

				// Get original content
				if content, ok := metadata["content"].(string); ok && content != "" {
					chunkParts = append(chunkParts, "Content: "+content)
				}

				// Get enriched context
				if enrichedContext, ok := metadata["enriched_context"].(string); ok && enrichedContext != "" {
					chunkParts = append(chunkParts, "Context: "+enrichedContext)
				}

				// Combine all available information
				if len(chunkParts) > 0 {
					combinedChunk := chunkParts[0]
					for i := 1; i < len(chunkParts); i++ {
						combinedChunk = fmt.Sprintf("\n\n%s\n--------------\n%s\n\n", combinedChunk, chunkParts[i])
					}
					allChunks = append(allChunks, combinedChunk)
				}
			}
		}
	}

	if len(allChunks) == 0 {
		log.Printf("[WARN] No chunks found for topics: %v", topics)
		return []string{}, nil
	}

	log.Printf("[INFO] Total chunks collected: %d", len(allChunks))

	shuffleStrings(allChunks)

	if len(allChunks) > limit {
		allChunks = allChunks[:limit]
		log.Printf("[INFO] Limited chunks to top %d", limit)
	}

	log.Printf("[INFO] Final chunks being returned: %d", len(allChunks))
	for i, chunk := range allChunks {
		log.Printf("[INFO] Chunk %d: %.150s...", i+1, chunk)
	}

	return allChunks, nil
}

func shuffleStrings(slice []string) {
	for i := range slice {
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}
