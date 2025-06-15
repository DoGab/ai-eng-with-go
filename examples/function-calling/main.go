package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Struct for AI's choice parameters
type RPSChoice struct {
	Choice string `json:"choice"`
}

// Function to determine winner
func determineWinner(humanChoice, aiChoice string) string {
	if humanChoice == aiChoice {
		return "tie"
	}

	if (humanChoice == "rock" && aiChoice == "scissors") ||
		(humanChoice == "paper" && aiChoice == "rock") ||
		(humanChoice == "scissors" && aiChoice == "paper") {
		return "human"
	}

	return "ai"
}

// Available tools for the AI
var availableTools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "make_rps_choice",
			Description: "Make your choice for rock, paper, scissors game",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"choice": map[string]any{
						"type":        "string",
						"description": "Your choice: rock, paper, or scissors",
						"enum":        []string{"rock", "paper", "scissors"},
					},
				},
				"required": []string{"choice"},
			},
		},
	},
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go [rock|paper|scissors]")
		os.Exit(1)
	}

	humanChoice := os.Args[1]
	if humanChoice != "rock" && humanChoice != "paper" && humanChoice != "scissors" {
		fmt.Println("Please choose rock, paper, or scissors")
		os.Exit(1)
	}

	// Initialize OpenAI LLM
	llm, err := openai.New(openai.WithModel("gpt-4o"))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "We're playing rock, paper, scissors. Please make your choice by calling the make_rps_choice function. Make sure your choice is as random as possible"),
	}

	fmt.Printf("You chose: %s\n", humanChoice)
	fmt.Println("AI is thinking...")

	// Call the LLM with tools
	resp, err := llm.GenerateContent(ctx, messageHistory, llms.WithTools(availableTools), llms.WithTemperature(0.9), llms.WithToolChoice("required"))
	if err != nil {
		log.Fatal(err)
	}

	// Check if there are tool calls in the response
	if len(resp.Choices) > 0 && len(resp.Choices[0].ToolCalls) > 0 {
		toolCall := resp.Choices[0].ToolCalls[0]

		if toolCall.FunctionCall.Name == "make_rps_choice" {
			// Parse the function call arguments into the struct
			var rpsChoice RPSChoice
			if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &rpsChoice); err != nil {
				log.Fatal("Failed to parse function arguments:", err)
			}

			fmt.Printf("AI chose: %s\n", rpsChoice.Choice)

			// Determine winner
			winner := determineWinner(humanChoice, rpsChoice.Choice)

			fmt.Println("\n--- RESULT ---")
			switch winner {
			case "human":
				fmt.Println("🎉 You win!")
			case "ai":
				fmt.Println("🤖 AI wins!")
			case "tie":
				fmt.Println("🤝 It's a tie!")
			}
		}
	} else {
		fmt.Println("AI didn't make a choice via function call")
		if len(resp.Choices) > 0 {
			fmt.Printf("AI response: %s\n", resp.Choices[0].Content)
		}
	}
}
