package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	ctx := context.Background()
	llm, err := openai.New(openai.WithModel("gpt-4o-mini"))
	if err != nil {
		log.Fatal(err)
	}
	prompt := "What would be a good company name for a company that makes colorful socks? Reply with the company name only, nothing else."
	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt, llms.WithTemperature(0.9))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(completion)
}
