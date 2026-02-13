# Go Agent Framework

Go implementation of the Microsoft Agent Framework — a library for building, orchestrating, and deploying AI agents.

## Installation

```bash
go get github.com/microsoft/agent-framework/go
```

Requires Go 1.22 or later. No external dependencies — stdlib only.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"

    af "github.com/microsoft/agent-framework/go/agentframework"
    "github.com/microsoft/agent-framework/go/openai"
)

func main() {
    client := openai.New(os.Getenv("OPENAI_API_KEY"),
        openai.WithModel("gpt-4o"),
    )

    agent := af.NewAgent(client,
        af.WithName("assistant"),
        af.WithInstructions("You are helpful and concise."),
    )

    resp, err := agent.Run(context.Background(), []af.Message{
        af.NewUserMessage("What is the capital of France?"),
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(resp.Text())
}
```

## Package Structure

```
go/
├── agentframework/     Core SDK — Agent, Content, Tools, Middleware, Session, Streaming
├── openai/             OpenAI Chat Completions provider
└── samples/
    └── chat/           Multi-turn chat with tools sample
```

### `agentframework` — Core Types

| Type | Description |
|------|-------------|
| `Agent` | Top-level orchestrator composing client, tools, middleware, sessions |
| `ChatClient` | Interface for LLM backends |
| `Content` (sealed) | 18 concrete content types (text, function call/result, approval, etc.) |
| `Message` | Chat message with role + contents |
| `Tool` | Callable function exposed to the model |
| `Session` | Multi-turn conversation state (service-managed or local) |
| `ResponseStream[T]` | Generic pull-based streaming iterator |
| `AgentMiddleware` | Wraps agent runs for cross-cutting concerns |
| `ChatMiddleware` | Wraps chat requests |
| `FunctionMiddleware` | Wraps tool invocations |

### `openai` — Provider

The OpenAI provider implements `ChatClient` with full support for:
- Synchronous and streaming responses (SSE)
- Function/tool calling
- All `ChatOptions` (temperature, top_p, max_tokens, etc.)
- Content filter error detection
- Custom base URL (Azure OpenAI, proxies)

## Tools

### Typed Tools (recommended)

```go
type WeatherArgs struct {
    Location string `json:"location" jsonschema:"description=City name,required"`
    Unit     string `json:"unit"     jsonschema:"enum=celsius|fahrenheit"`
}

weatherTool := af.NewTypedTool("get_weather", "Get current weather",
    func(ctx context.Context, args WeatherArgs) (any, error) {
        return fetchWeather(args.Location, args.Unit), nil
    },
)
```

JSON Schema is generated automatically from struct tags. The `jsonschema` tag supports:
- `description=...` — field description
- `required` — marks field as required
- `enum=a|b|c` — allowed values (pipe-separated)

### Raw Tools

```go
tool := af.NewTool("greet", "Says hello",
    json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
    func(ctx context.Context, args json.RawMessage) (any, error) {
        return "Hello!", nil
    },
)
```

### Tool Options

```go
af.NewTypedTool("send_email", "Send an email", handler,
    af.WithApprovalRequired(),    // requires human approval
    af.WithDeclarationOnly(),     // not auto-invoked
    af.WithMaxInvocations(5),     // limit calls per request
)
```

## Middleware

Three levels of middleware, all using the `func(next) handler` pattern:

```go
agent := af.NewAgent(client,
    af.WithAgentMiddleware(af.LoggingMiddleware(logger)),
    af.WithAgentMiddleware(customAuthMiddleware),
    af.WithFunctionMiddleware(rateLimitMiddleware),
)
```

First middleware in the list is the outermost wrapper.

### Built-in Middleware

- `LoggingMiddleware(logger)` — logs agent runs with duration and token counts

## Sessions

Sessions enable multi-turn conversations with automatic history management:

```go
session := agent.NewSession()

resp1, _ := agent.Run(ctx, []af.Message{af.NewUserMessage("Hi")},
    af.WithSession(session))

resp2, _ := agent.Run(ctx, []af.Message{af.NewUserMessage("What did I say?")},
    af.WithSession(session))
```

Sessions support two mutually exclusive modes:
- **Local mode**: messages stored in a `MessageStore` (default: in-memory)
- **Service mode**: identified by a service-managed thread ID

## Streaming

```go
stream, err := agent.RunStream(ctx, messages, af.WithSession(session))
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    update, ok, err := stream.Next(ctx)
    if err != nil { log.Fatal(err) }
    if !ok { break }
    fmt.Print(update.Text())
}
```

Or collect all at once:

```go
final, err := stream.FinalResponse(ctx)
```

## Error Handling

Errors use Go's standard `errors.Is` / `errors.As` patterns with sentinel chains:

```go
resp, err := agent.Run(ctx, messages)
if errors.Is(err, af.ErrContentFilter) {
    // Content was filtered
} else if errors.Is(err, af.ErrAuth) {
    // Authentication failed
} else if errors.Is(err, af.ErrService) {
    // Any service error
    var svcErr *af.ServiceError
    if errors.As(err, &svcErr) {
        log.Printf("HTTP %d: %s", svcErr.StatusCode, svcErr.Message)
    }
}
```

Error hierarchy:
```
ErrAgent → ErrExecution, ErrInitialization, ErrSession → ErrSessionModeLocked
ErrService → ErrContentFilter, ErrAuth, ErrInvalidRequest, ErrInvalidResponse
ErrTool → ErrToolExecution
```

## Content Types

The `Content` interface is sealed (18 concrete types):

| Type | Description |
|------|-------------|
| `TextContent` | Plain text |
| `TextReasoningContent` | Chain-of-thought reasoning |
| `FunctionCallContent` | Tool call from model |
| `FunctionResultContent` | Tool result |
| `ApprovalRequestContent` | Human approval request |
| `ApprovalResponseContent` | Human approval decision |
| `DataContent` | Binary data (data URI) |
| `URIContent` | External URI reference |
| `ErrorContent` | Error as content |
| `UsageContent` | Token usage |
| `HostedFileContent` | Service-hosted file |
| `HostedVectorStoreContent` | Service-hosted vector store |
| `CodeInterpreterCallContent` | Code interpreter invocation |
| `CodeInterpreterResultContent` | Code interpreter output |
| `ImageGenCallContent` | Image generation invocation |
| `ImageGenResultContent` | Image generation output |
| `MCPServerCallContent` | MCP server tool invocation |
| `MCPServerResultContent` | MCP server tool output |

JSON serialization uses a `$type` discriminator aligned with the cross-language schema.

## Running Tests

```bash
cd go
go test ./...
```

## Running the Sample

```bash
cd go/samples/chat
export OPENAI_API_KEY=sk-...
go run .
```

## Design Principles

- **Idiomatic Go**: functional options, `context.Context`, `errors.Is/As`, `slog`
- **No external dependencies**: stdlib only
- **Composable**: middleware, context providers, pluggable stores
- **Type-safe**: generics for tools and streaming
- **Cross-language alignment**: JSON wire format matches Python/C# implementations
