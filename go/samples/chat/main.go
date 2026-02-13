// Copyright (c) Microsoft. All rights reserved.

// Command chat demonstrates a multi-turn conversational agent with tool use.
//
// It works with both direct OpenAI and Azure AI Foundry endpoints.
//
// Usage with OpenAI:
//
//	export OPENAI_API_KEY=sk-...
//	go run .
//
// Usage with Azure AI Foundry:
//
//	export AZURE_FOUNDRY_ENDPOINT=https://<project>.services.ai.azure.com/openai/deployments/<deployment>
//	export AZURE_FOUNDRY_KEY=<your-key>
//	export AZURE_FOUNDRY_MODEL=gpt-4o          # optional, defaults to gpt-4o
//	go run .
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joho/godotenv"
	af "github.com/microsoft/agent-framework/go/agentframework"
	"github.com/microsoft/agent-framework/go/openai"
)

func main() {
	// Load .env file if present (ignored if missing).
	_ = godotenv.Load()

	// Enable debug logging if requested
	if os.Getenv("DEBUG") != "" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	client := newChatClient()


	// Define tools.
	weatherTool := af.NewTypedTool("get_weather",
		"Get the current weather for a location.",
		func(ctx context.Context, args struct {
			Location string `json:"location" jsonschema:"description=City name or location,required"`
			Unit     string `json:"unit"     jsonschema:"description=Temperature unit,enum=celsius|fahrenheit"`
		}) (any, error) {
			// Simulated weather API
			unit := args.Unit
			if unit == "" {
				unit = "fahrenheit"
			}
			temp := 72
			if unit == "celsius" {
				temp = 22
			}
			return map[string]any{
				"location":    args.Location,
				"temperature": temp,
				"unit":        unit,
				"condition":   "sunny",
			}, nil
		},
	)

	timeTool := af.NewTool("get_time",
		"Get the current time.",
		json.RawMessage(`{"type":"object","properties":{}}`),
		func(ctx context.Context, args json.RawMessage) (any, error) {
			return "2025-01-15T10:30:00Z", nil
		},
	)

	// Create the agent.
	agent := af.NewAgent(client,
		af.WithName("assistant"),
		af.WithInstructions("You are a helpful assistant. When asked about the weather, use the get_weather tool. When asked about the time, use the get_time tool. Keep responses concise."),
		af.WithTools(weatherTool, timeTool),
		af.WithAgentMiddleware(af.LoggingMiddleware(slog.Default())),
	)

	// Create a session for multi-turn conversation.
	session := agent.NewSession()

	fmt.Println("Chat with the assistant (type 'quit' to exit, 'stream' prefix for streaming)")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "quit" || input == "exit" {
			break
		}

		ctx := context.Background()

		if strings.HasPrefix(input, "stream ") {
			// Streaming mode
			input = strings.TrimPrefix(input, "stream ")
			streamResp, err := agent.RunStream(ctx,
				[]af.Message{af.NewUserMessage(input)},
				af.WithSession(session),
			)
			if err != nil {
				log.Printf("Error: %v", err)
				continue
			}

			fmt.Print("Assistant: ")
			for {
				update, ok, err := streamResp.Next(ctx)
				if err != nil {
					log.Printf("\nStream error: %v", err)
					break
				}
				if !ok {
					break
				}
				fmt.Print(update.Text())
			}
			fmt.Println()
			streamResp.Close()
		} else {
			// Non-streaming mode
			resp, err := agent.Run(ctx,
				[]af.Message{af.NewUserMessage(input)},
				af.WithSession(session),
			)
			if err != nil {
				log.Printf("Error: %v", err)
				continue
			}

			fmt.Printf("Assistant: %s\n", resp.Text())
			if resp.Usage.TotalTokens > 0 {
				fmt.Printf("  [tokens: %d in, %d out]\n",
					resp.Usage.InputTokens, resp.Usage.OutputTokens)
			}
		}
		fmt.Println()
	}
}

// newChatClient creates an OpenAI-compatible client, choosing between Azure AI
// Foundry and direct OpenAI based on which environment variables are set.
func newChatClient() *openai.Client {
	// Azure AI Foundry — uses the OpenAI-compatible endpoint.
	if endpoint := os.Getenv("AZURE_FOUNDRY_ENDPOINT"); endpoint != "" {
		key := os.Getenv("AZURE_FOUNDRY_KEY")
		model := os.Getenv("AZURE_FOUNDRY_MODEL")
		if model == "" {
			model = "gpt-4o"
		}
		
		fmt.Printf("Using Azure AI Foundry: %s\n", endpoint)
		
		// If no key provided, use Azure AD authentication
		if key == "" {
			fmt.Println("Using Azure AD authentication (DefaultAzureCredential)")
			fmt.Println("Attempting authentication via: environment variables, managed identity, az login, etc.")
			cred, err := azidentity.NewDefaultAzureCredential(nil)
			if err != nil {
				log.Fatalf("Failed to create Azure credential: %v", err)
			}
			fmt.Println("✓ Azure AD credential initialized")
			return openai.New("", // empty key when using Azure AD
				openai.WithBaseURL(endpoint),
				openai.WithModel(model),
				openai.WithAzureCredential(cred),
			)
		}
		
		// API key authentication
		fmt.Println("Using API key authentication")
		return openai.New(key,
			openai.WithBaseURL(endpoint),
			openai.WithModel(model),
			openai.WithHeaders(map[string]string{
				"api-key": key, // Azure uses api-key header instead of Bearer token
			}),
		)
	}

	// Direct OpenAI
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Set OPENAI_API_KEY or AZURE_FOUNDRY_ENDPOINT")
	}
	return openai.New(apiKey,
		openai.WithModel("gpt-4o"),
	)
}
