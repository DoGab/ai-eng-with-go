package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/pinecone-io/go-pinecone/v3/pinecone"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"google.golang.org/protobuf/types/known/structpb"
)

type Concept struct {
	ID          string
	Text        string
	Category    string
	Description string
}

func prettifyStruct(obj any) string {
	bytes, _ := json.MarshalIndent(obj, "", "  ")
	return string(bytes)
}

func generateRandomConcepts() []Concept {
	concepts := []Concept{
		{"concept-1", "Artificial Intelligence", "Technology", "The simulation of human intelligence in machines that are programmed to think and learn."},
		{"concept-2", "Machine Learning", "Technology", "A subset of AI that enables systems to automatically learn and improve from experience."},
		{"concept-3", "Neural Networks", "Technology", "Computing systems inspired by biological neural networks that constitute animal brains."},
		{"concept-4", "Deep Learning", "Technology", "A subset of machine learning using multi-layered neural networks."},
		{"concept-5", "Natural Language Processing", "Technology", "AI technology that helps computers understand, interpret and manipulate human language."},
		{"concept-6", "Computer Vision", "Technology", "AI field that trains computers to interpret and understand visual information."},
		{"concept-7", "Quantum Computing", "Technology", "Computing using quantum-mechanical phenomena like superposition and entanglement."},
		{"concept-8", "Blockchain Technology", "Technology", "A distributed ledger technology that maintains a continuously growing list of records."},
		{"concept-9", "Cybersecurity", "Technology", "The practice of protecting systems, networks, and programs from digital attacks."},
		{"concept-10", "Cloud Computing", "Technology", "Delivery of computing services over the internet including servers, storage, and software."},
		{"concept-11", "Internet of Things", "Technology", "Network of physical objects embedded with sensors and software to connect and exchange data."},
		{"concept-12", "Robotics", "Technology", "Branch of engineering that deals with the design, construction, and operation of robots."},
		{"concept-13", "Augmented Reality", "Technology", "Technology that overlays digital information on the real world through devices."},
		{"concept-14", "Virtual Reality", "Technology", "Computer-generated simulation of a three-dimensional environment."},
		{"concept-15", "5G Technology", "Technology", "Fifth generation cellular network technology providing faster speeds and lower latency."},
		{"concept-16", "Edge Computing", "Technology", "Computing that brings data processing closer to where data is being generated."},
		{"concept-17", "Bioinformatics", "Science", "Application of computer technology to the management of biological information."},
		{"concept-18", "Renewable Energy", "Environment", "Energy from natural resources that are naturally replenished on a human timescale."},
		{"concept-19", "Sustainable Development", "Environment", "Development that meets present needs without compromising future generations."},
		{"concept-20", "Gene Therapy", "Science", "Technique that involves introducing genetic material into a patient's cells to treat disease."},
	}

	return concepts
}

func generateEmbeddings(embedder embeddings.Embedder, texts []string) ([][]float32, error) {
	ctx := context.Background()

	// Generate embeddings using LangChain Go
	vectors, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	return vectors, nil
}

func createPineconeIndex(pc *pinecone.Client, indexName string, dimension int32) error {
	ctx := context.Background()

	// Check if index already exists
	indexes, err := pc.ListIndexes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list indexes: %w", err)
	}

	for _, idx := range indexes {
		if idx.Name == indexName {
			log.Printf("Index %s already exists, skipping creation", indexName)
			return nil
		}
	}

	// Create the index
	deletionProtection := pinecone.DeletionProtectionDisabled
	metric := pinecone.Cosine
	_, err = pc.CreateServerlessIndex(ctx, &pinecone.CreateServerlessIndexRequest{
		Name:               indexName,
		Dimension:          &dimension,
		Metric:             &metric,
		Cloud:              pinecone.Aws,
		Region:             "us-east-1",
		DeletionProtection: &deletionProtection,
		Tags:               &pinecone.IndexTags{"environment": "development", "project": "vector-search-demo"},
	})
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	log.Printf("Successfully created index: %s", indexName)

	// Wait for index to be ready
	for {
		idx, err := pc.DescribeIndex(ctx, indexName)
		if err != nil {
			return fmt.Errorf("failed to describe index: %w", err)
		}
		if idx.Status.Ready {
			log.Printf("Index %s is ready", indexName)
			break
		}
		log.Printf("Waiting for index %s to be ready...", indexName)
		time.Sleep(10 * time.Second)
	}

	return nil
}

func upsertVectors(pc *pinecone.Client, indexName string, concepts []Concept, embeddings [][]float32) error {
	ctx := context.Background()

	// Get index connection
	idxDesc, err := pc.DescribeIndex(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to describe index: %w", err)
	}

	idxConn, err := pc.Index(pinecone.NewIndexConnParams{
		Host:      idxDesc.Host,
		Namespace: "demo-concepts",
	})
	if err != nil {
		return fmt.Errorf("failed to create index connection: %w", err)
	}

	// Prepare vectors for upsert
	var vectors []*pinecone.Vector
	for i, concept := range concepts {
		metadata := map[string]any{
			"text":        concept.Text,
			"category":    concept.Category,
			"description": concept.Description,
			"created_at":  time.Now().Format(time.RFC3339),
		}

		metadataStruct, err := structpb.NewStruct(metadata)
		if err != nil {
			return fmt.Errorf("failed to create metadata struct: %w", err)
		}

		vector := &pinecone.Vector{
			Id:       concept.ID,
			Values:   &embeddings[i],
			Metadata: metadataStruct,
		}
		vectors = append(vectors, vector)
	}

	// Upsert vectors in batches of 10
	batchSize := 10
	for i := 0; i < len(vectors); i += batchSize {
		end := int(math.Min(float64(i+batchSize), float64(len(vectors))))

		batch := vectors[i:end]
		count, err := idxConn.UpsertVectors(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to upsert vectors: %w", err)
		}
		log.Printf("Successfully upserted %d vectors (batch %d)", count, i/batchSize+1)
	}

	return nil
}

func queryVectors(pc *pinecone.Client, indexName, queryText string, embedder embeddings.Embedder) error {
	ctx := context.Background()

	// Generate embedding for query
	queryEmbeddings, err := generateEmbeddings(embedder, []string{queryText})
	if err != nil {
		return fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Get index connection
	idxDesc, err := pc.DescribeIndex(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to describe index: %w", err)
	}

	idxConn, err := pc.Index(pinecone.NewIndexConnParams{
		Host:      idxDesc.Host,
		Namespace: "demo-concepts",
	})
	if err != nil {
		return fmt.Errorf("failed to create index connection: %w", err)
	}

	// Query for similar vectors
	result, err := idxConn.QueryByVectorValues(ctx, &pinecone.QueryByVectorValuesRequest{
		Vector:          queryEmbeddings[0],
		TopK:            5,
		IncludeValues:   false,
		IncludeMetadata: true,
	})
	if err != nil {
		return fmt.Errorf("failed to query vectors: %w", err)
	}

	fmt.Printf("\n=== Query Results for: '%s' ===\n", queryText)
	for i, match := range result.Matches {
		metadata := match.Vector.Metadata.AsMap()
		fmt.Printf("%d. ID: %s (Score: %.4f)\n", i+1, match.Vector.Id, match.Score)
		fmt.Printf("   Text: %s\n", metadata["text"])
		fmt.Printf("   Category: %s\n", metadata["category"])
		fmt.Printf("   Description: %s\n\n", metadata["description"])
	}

	return nil
}

func queryWithCategoryFilter(pc *pinecone.Client, indexName, queryText, category string, embedder embeddings.Embedder) error {
	ctx := context.Background()

	// Generate embedding for query
	queryEmbeddings, err := generateEmbeddings(embedder, []string{queryText})
	if err != nil {
		return fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Get index connection
	idxDesc, err := pc.DescribeIndex(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to describe index: %w", err)
	}

	idxConn, err := pc.Index(pinecone.NewIndexConnParams{
		Host:      idxDesc.Host,
		Namespace: "demo-concepts",
	})
	if err != nil {
		return fmt.Errorf("failed to create index connection: %w", err)
	}

	// Create metadata filter
	filter := map[string]any{
		"category": map[string]any{
			"$eq": category,
		},
	}

	filterStruct, err := structpb.NewStruct(filter)
	if err != nil {
		return fmt.Errorf("failed to create filter struct: %w", err)
	}

	// Query with filter
	result, err := idxConn.QueryByVectorValues(ctx, &pinecone.QueryByVectorValuesRequest{
		Vector:          queryEmbeddings[0],
		TopK:            3,
		MetadataFilter:  filterStruct,
		IncludeValues:   false,
		IncludeMetadata: true,
	})
	if err != nil {
		return fmt.Errorf("failed to query vectors with filter: %w", err)
	}

	fmt.Printf("\n=== Filtered Query Results for: '%s' (Category: %s) ===\n", queryText, category)
	for i, match := range result.Matches {
		metadata := match.Vector.Metadata.AsMap()
		fmt.Printf("%d. ID: %s (Score: %.4f)\n", i+1, match.Vector.Id, match.Score)
		fmt.Printf("   Text: %s\n", metadata["text"])
		fmt.Printf("   Category: %s\n", metadata["category"])
		fmt.Printf("   Description: %s\n\n", metadata["description"])
	}

	return nil
}

func main() {

	// Check for required environment variables
	pineconeAPIKey := os.Getenv("PINECONE_API_KEY")
	if pineconeAPIKey == "" {
		log.Fatal("PINECONE_API_KEY environment variable is required")
	}

	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	ctx := context.Background()

	// Initialize Pinecone client
	pc, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: pineconeAPIKey,
	})
	if err != nil {
		log.Fatalf("Failed to create Pinecone client: %v", err)
	}

	// Initialize LangChain OpenAI client and embedder
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create OpenAI client: %v", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	// Generate 20 random concepts
	concepts := generateRandomConcepts()
	log.Printf("Generated %d concepts", len(concepts))

	// Create text array for embedding generation
	var texts []string
	for _, concept := range concepts {
		// Combine text and description for richer embeddings
		fullText := fmt.Sprintf("%s: %s", concept.Text, concept.Description)
		texts = append(texts, fullText)
	}

	// Generate embeddings using LangChain OpenAI
	log.Println("Generating embeddings using LangChain OpenAI...")
	embeddings, err := generateEmbeddings(embedder, texts)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}
	log.Printf("Generated embeddings for %d concepts (dimension: %d)", len(embeddings), len(embeddings[0]))

	// Create Pinecone index
	indexName := "concept-search-demo"
	dimension := int32(len(embeddings[0])) // OpenAI Ada v2 has 1536 dimensions
	log.Printf("Creating Pinecone index '%s' with dimension %d...", indexName, dimension)

	if err := createPineconeIndex(pc, indexName, dimension); err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}

	// Upsert vectors to Pinecone
	log.Println("Upserting vectors to Pinecone...")
	if err := upsertVectors(pc, indexName, concepts, embeddings); err != nil {
		log.Fatalf("Failed to upsert vectors: %v", err)
	}

	// Wait a moment for vectors to be indexed
	log.Println("Waiting for vectors to be indexed...")
	time.Sleep(5 * time.Second)

	// Demonstrate querying
	log.Println("\n=== DEMONSTRATION: Vector Search ===")

	// Query 1: General search
	if err := queryVectors(pc, indexName, "machine learning algorithms", embedder); err != nil {
		log.Printf("Query 1 failed: %v", err)
	}

	// Query 2: Category-filtered search
	if err := queryWithCategoryFilter(pc, indexName, "advanced computing technology", "Technology", embedder); err != nil {
		log.Printf("Query 2 failed: %v", err)
	}

	// Query 3: Another general search
	if err := queryVectors(pc, indexName, "environmental sustainability", embedder); err != nil {
		log.Printf("Query 3 failed: %v", err)
	}

	// Get index statistics
	log.Println("\n=== INDEX STATISTICS ===")
	idxDesc, err := pc.DescribeIndex(ctx, indexName)
	if err != nil {
		log.Printf("Failed to describe index: %v", err)
	} else {
		fmt.Printf("Index Description:\n%s\n", prettifyStruct(idxDesc))
	}

	log.Println("\n=== DEMO COMPLETED SUCCESSFULLY ===")
	log.Printf("Index '%s' contains %d vectors and is ready for further queries", indexName, len(concepts))
}
