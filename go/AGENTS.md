# Go Agent Framework — AGENTS.md

Instructions for AI agents working with the Go implementation of the Microsoft Agent Framework.

## Repository Layout

```
go/
├── go.mod                      Module definition (github.com/microsoft/agent-framework/go)
├── agentframework/             Core SDK package
│   ├── doc.go                  Package documentation
│   ├── agent.go                Agent struct, NewAgent, Run, RunStream
│   ├── client.go               ChatClient interface
│   ├── content.go              Sealed Content interface + 18 concrete types
│   ├── content_json.go         JSON marshal/unmarshal with $type discriminator
│   ├── context_provider.go     ContextProvider interface, InvocationContext
│   ├── errors.go               Sentinel errors, ServiceError, ToolError
│   ├── message.go              Message struct, Role, FinishReason, helpers
│   ├── middleware.go           Three-level middleware types and chaining
│   ├── options.go              ChatOptions, ToolChoice, MergeChatOptions
│   ├── response.go             ChatResponse, AgentResponse, update merging
│   ├── schema.go               Reflection-based JSON Schema generation
│   ├── session.go              Session with dual-mode (service/local)
│   ├── store.go                MessageStore interface, InMemoryStore
│   ├── stream.go               ResponseStream[T], AgentResponseStream, MapStream
│   ├── telemetry.go            LoggingMiddleware (slog-based)
│   ├── tool.go                 Tool interface, FunctionTool, NewTypedTool
│   ├── tool_invoke.go          invokeFunctions loop
│   └── usage.go                UsageDetails
├── openai/                     OpenAI provider package
│   ├── doc.go                  Package documentation
│   ├── client.go               Client struct implementing ChatClient
│   ├── message_prep.go         Framework → OpenAI request conversion
│   ├── option.go               Client functional options
│   ├── response_parse.go       OpenAI response → framework types
│   └── transport.go            HTTP transport with error mapping
└── samples/
    └── chat/                   Multi-turn chat sample with tools
        └── main.go
```

## Conventions

### Language & Style
- Go 1.22, `github.com/microsoft/agent-framework/go` module
- No external dependencies — stdlib only
- Functional options pattern for configuration
- `context.Context` as first parameter on all I/O operations
- No `Get`/`Set` prefixes on exported methods (Go convention)
- Receiver names are single letters (`a` for Agent, `s` for Session)

### Error Handling
- Sentinel errors with `errors.Is`/`errors.As` support
- Error chains via `fmt.Errorf("%w: ...", sentinel, ...)`
- Typed errors (`ServiceError`, `ToolError`) implement `Unwrap()`

### Testing
- Table-driven tests in `*_test.go` files
- Use `_test` package suffix (black-box testing)
- Mock via unexported interfaces (e.g., `transport` in openai)
- For HTTP mocking: use `WithHTTPClient` with custom `RoundTripper`

### JSON Serialization
- Content uses `$type` discriminator (aligned with `schemas/durable-agent-entity-state.json`)
- `Contents` type has custom `MarshalJSON`/`UnmarshalJSON`
- JSON tags on all serialized structs

### Key Patterns
- **Sealed interface**: `Content` uses a private `sealed()` marker method
- **Generics**: `ResponseStream[T]`, `NewTypedTool[Args]`, `GenerateSchema[T]`, `MapStream[A,B]`
- **Middleware chain**: `func(next Handler) Handler` pattern, first = outermost
- **Dual-mode session**: service-managed (thread ID) vs local (MessageStore), locked after first set

## Common Tasks

### Adding a new Content type
1. Add `ContentType` constant in `content.go`
2. Add struct with embedded `base` and `Type()` method
3. Add marshal case in `content_json.go` `MarshalContentJSON`
4. Add unmarshal case in `content_json.go` `UnmarshalContentJSON`
5. Add test in `content_test.go`

### Adding a new provider
1. Create new package under `go/` (e.g., `go/azure/`)
2. Implement `agentframework.ChatClient` interface
3. Use functional options for client configuration
4. Map provider errors to `agentframework.ServiceError` with appropriate sentinel

### Adding middleware
1. Choose level: `AgentMiddleware`, `ChatMiddleware`, or `FunctionMiddleware`
2. Return `func(next Handler) Handler`
3. Call `next` to continue the chain, or return early to short-circuit

## Build & Test

```bash
cd go
go build ./...
go test ./...
go vet ./...
```
