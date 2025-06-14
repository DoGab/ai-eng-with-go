package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	http.HandleFunc("/stream", streamHandler)

	fmt.Println("Server starting on :8080")
	fmt.Println("Test with: curl -N http://localhost:8080/stream")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func streamHandler(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Initialize OpenAI LLM with GPT-4 model
	llm, err := openai.New(
		openai.WithModel("gpt-4o-mini"),
	)
	if err != nil {
		fmt.Fprintf(w, "data: Error initializing LLM: %v\n\n")
		return
	}

	ctx := context.Background()
	prompt := "Write a short story about a robot learning to paint."

	// Send initial message
	fmt.Fprintf(w, "Starting GPT-4 stream...\n")
	w.(http.Flusher).Flush()

	// Stream the response via SSE
	_, err = llms.GenerateFromSinglePrompt(ctx, llm, prompt,
		llms.WithTemperature(0.7),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			// Send each chunk as SSE data
			fmt.Fprintf(w, "%s", string(chunk))
			w.(http.Flusher).Flush()
			return nil
		}),
	)

	if err != nil {
		fmt.Fprintf(w, "Error: %v\n\n", err)
		return
	}

	// Send completion message
	fmt.Fprintf(w, "\n[DONE]\n")
	w.(http.Flusher).Flush()
}
