# Vector Search with Pinecone Demo

This Go application demonstrates how to use Pinecone as a vector database with OpenAI embeddings via LangChain Go. It generates 20 random concepts, converts them to embeddings using OpenAI's text-embedding-ada-002 model through LangChain Go, stores them in a Pinecone index, and demonstrates semantic search capabilities.

## Features

- **Random Concept Generation**: Creates 20 diverse concepts across technology, science, and environment categories
- **LangChain Go Integration**: Uses LangChain Go with OpenAI's text-embedding-ada-002 model to generate high-quality vector embeddings
- **Pinecone Integration**: Creates and manages a serverless Pinecone index with proper error handling
- **Batch Processing**: Efficiently upserts vectors in batches for better performance
- **Semantic Search**: Demonstrates vector similarity search with and without metadata filtering
- **Rich Metadata**: Stores additional information like categories, descriptions, and timestamps
- **Index Statistics**: Shows usage statistics and index health information

## Prerequisites

1. **Pinecone Account**: Sign up at [pinecone.io](https://pinecone.io) and get your API key
2. **OpenAI Account**: Get an API key from [OpenAI](https://platform.openai.com)
3. **Go 1.24+**: Make sure you have Go installed

## Setup

1. **Clone and navigate to the project**:
   ```bash
   cd examples/vector-search
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Set environment variables**:
   ```bash
   export PINECONE_API_KEY="your-pinecone-api-key"
   export OPENAI_API_KEY="your-openai-api-key"
   ```

   Or create a `.env` file:
   ```
   PINECONE_API_KEY=your-pinecone-api-key
   OPENAI_API_KEY=your-openai-api-key
   ```

## Usage

Run the application:

```bash
go run main.go
```

## What the Application Does

### 1. Concept Generation
The app generates 20 predefined concepts covering:
- **Technology**: AI, Machine Learning, Neural Networks, Quantum Computing, etc.
- **Science**: Bioinformatics, Gene Therapy
- **Environment**: Renewable Energy, Sustainable Development

### 2. Embedding Generation
Each concept is processed through OpenAI's `text-embedding-ada-002` model via LangChain Go to create 1536-dimensional vectors that capture semantic meaning.

### 3. Pinecone Index Creation
- Creates a serverless index named `concept-search-demo`
- Uses cosine similarity metric (ideal for normalized embeddings)
- Deployed on AWS in `us-east-1` region
- Includes tags for organization

### 4. Vector Storage
- Stores vectors with rich metadata including:
  - Original text and description
  - Category classification
  - Creation timestamp
- Uses the `demo-concepts` namespace for organization

### 5. Search Demonstrations
The app performs three types of searches:

1. **General Semantic Search**: Query "machine learning algorithms"
2. **Filtered Search**: Query "advanced computing technology" filtered by "Technology" category
3. **Environmental Search**: Query "environmental sustainability"

### 6. Statistics Display
Shows index statistics including vector count and namespace information.

## Key Components

### Data Structure
```go
type Concept struct {
    ID          string
    Text        string
    Category    string
    Description string
}
```

### Search Functions
- `queryVectors()`: Basic semantic search returning top 5 results
- `queryWithCategoryFilter()`: Search with metadata filtering
- Demonstrates both inclusive and filtered search patterns

### Error Handling
- Comprehensive error checking throughout the pipeline
- Graceful handling of API failures
- Index existence checking to avoid conflicts

## Example Output

```
Generated 20 concepts
Generating embeddings using LangChain OpenAI...
Generated embeddings for 20 concepts (dimension: 1536)
Creating Pinecone index 'concept-search-demo' with dimension 1536...
Successfully created index: concept-search-demo
Index concept-search-demo is ready
Upserting vectors to Pinecone...
Successfully upserted 10 vectors (batch 1)
Successfully upserted 10 vectors (batch 2)

=== Query Results for: 'machine learning algorithms' ===
1. ID: concept-2 (Score: 0.8945)
   Text: Machine Learning
   Category: Technology
   Description: A subset of AI that enables systems to automatically learn and improve from experience.

2. ID: concept-1 (Score: 0.8721)
   Text: Artificial Intelligence
   Category: Technology
   Description: The simulation of human intelligence in machines that are programmed to think and learn.
```

## Configuration

The application uses these default settings:
- **Index Name**: `concept-search-demo`
- **Namespace**: `demo-concepts`
- **Embedding Model**: `text-embedding-ada-002`
- **Similarity Metric**: Cosine
- **Batch Size**: 10 vectors per upsert
- **Query Top-K**: 5 results (3 for filtered queries)

## Cleanup

The application creates a Pinecone index that will incur costs. To clean up:

1. Delete the index through Pinecone console, or
2. Add cleanup code to delete the index programmatically:

```go
err := pc.DeleteIndex(ctx, "concept-search-demo")
```

## Architecture Notes

This example follows the recommended patterns from both the Pinecone research document and official SDK documentation:

- Uses the latest Pinecone Go SDK v3
- Implements proper error handling and retry logic
- Utilizes serverless indexes for automatic scaling  
- Demonstrates both basic and filtered search patterns
- Includes comprehensive metadata for rich search experiences
- Uses batch processing for efficiency

## Next Steps

To extend this example:
- Add more sophisticated metadata filtering
- Implement hybrid search (dense + sparse vectors)
- Add reranking with Pinecone's reranker models
- Integrate with different embedding models
- Add real-time updates and deletions
- Implement namespace-based multi-tenancy