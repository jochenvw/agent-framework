// Copyright (c) Microsoft. All rights reserved.

// Package agentframework provides the core types and abstractions for building
// AI agents in Go. It includes a composable Agent with tool calling, middleware
// pipelines, session management, and streaming support.
//
// # Quick Start
//
// Create a ChatClient (e.g., from the openai package) and build an Agent:
//
//	client := openai.New(os.Getenv("OPENAI_API_KEY"), openai.WithModel("gpt-4o"))
//
//	agent := agentframework.NewAgent(client,
//	    agentframework.WithName("assistant"),
//	    agentframework.WithInstructions("You are helpful."),
//	    agentframework.WithTools(myTool),
//	)
//
//	resp, err := agent.Run(ctx, []agentframework.Message{
//	    agentframework.NewUserMessage("Hello!"),
//	})
//
// # Architecture
//
// The package is organized around these key abstractions:
//
//   - [Agent]: the top-level orchestrator that composes a client with tools,
//     middleware, and session management.
//   - [ChatClient]: interface for LLM backends (implemented by provider packages).
//   - [Tool]: callable functions exposed to the model via function calling.
//   - [Content]: sealed interface with 18 concrete types representing message parts.
//   - [Session]: manages multi-turn conversation state (service-managed or local).
//   - [ResponseStream]: generic pull-based iterator for streaming responses.
//   - Middleware: three levels (Agent, Chat, Function) for cross-cutting concerns.
//
// # Tools
//
// Use [NewTypedTool] for type-safe tools with automatic JSON Schema generation:
//
//	type WeatherArgs struct {
//	    Location string `json:"location" jsonschema:"description=City name,required"`
//	    Unit     string `json:"unit"     jsonschema:"enum=celsius|fahrenheit"`
//	}
//
//	tool := agentframework.NewTypedTool("get_weather", "Get current weather",
//	    func(ctx context.Context, args WeatherArgs) (any, error) {
//	        return fetchWeather(args.Location, args.Unit)
//	    },
//	)
//
// # Middleware
//
// Add cross-cutting behavior at three levels:
//
//	agent := agentframework.NewAgent(client,
//	    agentframework.WithAgentMiddleware(agentframework.LoggingMiddleware(logger)),
//	    agentframework.WithFunctionMiddleware(rateLimitMiddleware),
//	)
//
// # Sessions
//
// Use sessions for multi-turn conversations:
//
//	session := agent.NewSession()
//	resp1, _ := agent.Run(ctx, msgs1, agentframework.WithSession(session))
//	resp2, _ := agent.Run(ctx, msgs2, agentframework.WithSession(session))
package agentframework
