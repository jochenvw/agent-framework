// Copyright (c) Microsoft. All rights reserved.

// Command local demonstrates a multi-turn conversational agent running
// entirely on your machine using Foundry Local.
//
// Foundry Local must be installed (https://github.com/microsoft/Foundry-Local).
// The sample will auto-start the service, download, and load the model.
//
// Usage:
//
//	go run .                              # defaults to phi-4-mini
//	go run . --model phi-4-mini           # explicit model alias
//	go run . --model phi-4-mini --gpu     # force GPU
//	go run . --model phi-4-mini --serve   # HTTP server mode on :8080
package main

import (
	"bufio"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/microsoft/foundry-local/sdk/go/foundrylocal"

	af "github.com/microsoft/agent-framework/go/agentframework"
	"github.com/microsoft/agent-framework/go/openai"
)

//go:embed tool_calling_prompt.md
var toolCallingPrompt string

func main() {
	modelAlias := flag.String("model", "phi-4-mini", "Foundry Local model alias to use")
	useGPU := flag.Bool("gpu", false, "prefer GPU execution provider")
	serve := flag.Bool("serve", false, "run as HTTP server instead of interactive CLI")
	port := flag.String("port", "8080", "HTTP listen port (serve mode)")
	flag.Parse()

	ctx := context.Background()

	// ── Bootstrap Foundry Local ──────────────────────────────────────
	fmt.Printf("Starting Foundry Local with model %q...\n", *modelAlias)

	flOpts := []foundrylocal.Option{
		foundrylocal.WithModel(*modelAlias),
		foundrylocal.WithProgress(func(pct float64) {
			fmt.Printf("\r  downloading: %.0f%%", pct)
			if pct >= 100 {
				fmt.Println()
			}
		}),
	}
	if *useGPU {
		flOpts = append(flOpts, foundrylocal.WithDevice(foundrylocal.GPU))
	}

	mgr, err := foundrylocal.New(ctx, flOpts...)
	if err != nil {
		log.Fatalf("foundry local: %v", err)
	}

	// Resolve model to get the actual model ID for API calls.
	var lookupOpts []foundrylocal.LookupOption
	if *useGPU {
		lookupOpts = append(lookupOpts, foundrylocal.ForDevice(foundrylocal.GPU))
	}
	info, err := mgr.LookupModel(ctx, *modelAlias, lookupOpts...)
	if err != nil {
		log.Fatalf("model lookup: %v", err)
	}

	fmt.Printf("Model loaded: %s\n", info)
	fmt.Printf("Endpoint:     %s\n\n", mgr.Endpoint())

	// Default options to avoid context length issues.
	maxTokens := 512
	defaultOpts := &af.ChatOptions{
		MaxTokens: &maxTokens,
	}

	// ── Get tools ────────────────────────────────────────────────────
	tools := GetTools()

	// ── Create client with tool call workaround middleware ───────────
	// The middleware intercepts chat responses and converts text-based
	// tool calls (e.g., `[{"get_weather": {...}}]`) into proper
	// FunctionCallContent objects before the agent processes them.
	logger := slog.Default()

	clientWithWorkaround := openai.New(mgr.APIKey(),
		openai.WithBaseURL(mgr.Endpoint()),
		openai.WithModel(info.ID),
		openai.WithChatMiddleware(ToolCallWorkaroundMiddleware(logger)),
	)

	// ── Create the agent ─────────────────────────────────────────────
	agent := af.NewAgent(clientWithWorkaround,
		af.WithName("local-assistant"),
		af.WithInstructions(toolCallingPrompt),
		af.WithTools(tools...),
		af.WithAgentMiddleware(af.LoggingMiddleware(logger)),
		af.WithFunctionMiddleware(ToolCallLoggingMiddleware()),
	)

	// ── HTTP server mode ─────────────────────────────────────────────
	if *serve {
		apiKey := os.Getenv("AGENT_API_KEY")
		if apiKey == "" {
			log.Printf("[agent] WARNING: AGENT_API_KEY not set, /invoke is unauthenticated")
		}

		srv := newAgentServer(agent, apiKey, *port)
		addr := fmt.Sprintf(":%s", *port)

		baseURL := fmt.Sprintf("http://localhost:%s", *port)
		tunnelURL := os.Getenv("DEVTUNNEL_URL")
		if tunnelURL != "" {
			baseURL = strings.TrimRight(tunnelURL, "/")
		}

		log.Printf("[agent] listening on http://localhost:%s", *port)
		log.Printf("[agent] endpoints:")
		log.Printf("[agent]   Health:     %s/health", baseURL)
		log.Printf("[agent]   Agent Card: %s/.well-known/agent.json", baseURL)
		log.Printf("[agent]   Agent Card: %s/.well-known/agent-card.json", baseURL)
		log.Printf("[agent]   Invoke:     %s/invoke  (POST)", baseURL)

		if err := http.ListenAndServe(addr, srv); err != nil {
			log.Fatalf("[agent] server error: %v", err)
		}
		return
	}

	// ── Chat loop ────────────────────────────────────────────────────
	session := agent.NewSession()

	fmt.Println("Chat with the local assistant (type 'quit' to exit, 'stream' prefix for streaming)")
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

		if strings.HasPrefix(input, "stream ") {
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
			resp, err := agent.Run(ctx,
				[]af.Message{af.NewUserMessage(input)},
				af.WithSession(session),
				af.WithRunOptions(defaultOpts),
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
