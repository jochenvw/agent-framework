# Local Agent Sample

This sample demonstrates running an AI agent entirely on your local machine using [Foundry Local](https://github.com/microsoft/Foundry-Local).

## Features

- **On-device inference**: No cloud API calls, runs completely locally
- **Auto-setup**: Automatically starts service, downloads and loads models
- **Tool calling**: Weather, time, and file listing tools with workaround for local model limitations
- **HTTP server mode**: Expose the agent via `POST /invoke`, `/health`, and `/.well-known/agent-card.json`
- **Device selection**: Choose between CPU, GPU, or NPU
- **Streaming support**: Prefix queries with `stream` for streaming responses
- **Session management**: Multi-turn conversations with context retention

## Prerequisites

1. **Foundry Local**: Must be installed and available
   - See: https://github.com/microsoft/Foundry-Local
   
2. **Go 1.22+**: Required for building and running

## Usage

```bash
# Default - uses phi-4-mini on auto-detected device
go run .

# Explicit model selection
go run . --model phi-4-mini

# Force GPU execution
go run . --model phi-4-mini --gpu

# Streaming responses
You: stream what's the weather in paris?
```

## HTTP Server Mode

The sample can also run as an HTTP server, exposing the agent via `POST /invoke`,
`GET /health`, and `GET /.well-known/agent-card.json`.

```bash
# Start in HTTP mode (default port 8080)
go run . --model phi-4-mini --serve

# Custom port
go run . --model phi-4-mini --serve --port 9090

# With bearer token authentication
AGENT_API_KEY=mysecret go run . --model phi-4-mini --serve
```

### Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/health` | GET | Health check, returns `{"status":"ok"}` |
| `/.well-known/agent-card.json` | GET | Agent card with capabilities, tools, auth info |
| `/invoke` | POST | Send user input, get agent response |

### Testing with curl

```bash
# Health check
curl http://localhost:8080/health

# Agent card
curl http://localhost:8080/.well-known/agent-card.json

# Invoke (unauthenticated)
curl -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d '{"input":"list files"}'

# Invoke (with auth)
curl -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mysecret" \
  -d '{"input":"list files", "conversationId":"conv1"}'
```

### Environment Variables

| Variable | Description |
|---|---|
| `AGENT_API_KEY` | Bearer token for `/invoke` auth (optional; unauthenticated if empty) |

### Console Output (HTTP mode)

```
[agent] listening on http://localhost:8080
[agent] invoke received
[agent] auth=OK
[agent] user input="list files"
[agent] tool call requested: list_local_files
[agent] tool executed successfully (4 files)
[agent] response sent (189 bytes)
```

### Exposing via Dev Tunnel

Use the included `devtunnel.ps1` to make the HTTP server reachable externally:

```powershell
# Start the agent server first, then in another terminal:
.\devtunnel.ps1 -LocalPort 8080
```

Then register the tunnel URL with Foundry Control Plane using `registration.ps1`.

## Tool Calling Workaround

Local inference runtimes may not fully implement the OpenAI tool calling wire format. This sample includes a workaround:

### The Problem

When a model should return structured tool calls like:
```json
{
  "finish_reason": "tool_calls",
  "tool_calls": [{"id": "call_123", "function": {"name": "get_weather", ...}}]
}
```

Some local runtimes instead return:
```json
{
  "finish_reason": "stop",
  "content": "[{\"get_weather\": {\"location\": \"Paris\"}}]"
}
```

### The Solution

Three components work together:

1. **Enhanced System Prompt** (tool_calling_prompt.md)
   - Teaches the model the expected JSON format explicitly
   - Provides examples of parameter gathering
   - Emphasizes asking for missing required parameters
   - Instructs model to return ONLY the JSON (no explanations)

2. **Chat Middleware** (tool_workaround.go)
   - Intercepts responses at the chat level (before agent processes them)
   - Detects JSON arrays in text responses that match tool call patterns
   - Uses regex to identify `[{"function_name": {...}}]` patterns
   - Strips markdown code fences
   - Sets `finish_reason` to "tool_calls" to trigger agent's execution logic

3. **Parser**
   - Converts detected JSON to proper `FunctionCallContent` objects
   - Generates synthetic call IDs (`call_local_0`, `call_local_1`, etc.)
   - Replaces text content with function call content

### Limitations

- **Model-dependent**: Success depends on the model's ability to follow the prompt
- **Pattern matching**: Only detects exact JSON array format
- **No guarantee**: Smaller models may still struggle with complex tool scenarios
- **Experimental**: This is a workaround, not a proper solution

For production use cases requiring reliable tool calling, use cloud models (see ../chat/).

## Architecture

```
main.go              - Entry point, chat loop, and HTTP server mode
server.go            - HTTP handler (/invoke, /health, agent card)
tools.go             - Tool definitions (weather, time, list_local_files)
tool_calling_prompt.md - System prompt with tool calling instructions
tool_workaround.go   - Middleware that converts text to tool calls
devtunnel.ps1        - Dev Tunnel helper for external access
registration.ps1     - Foundry Control Plane registration helper
```

## Files

- **tool_calling_prompt.md**: Markdown system prompt embedded via `//go:embed` that teaches the model to output tool calls as JSON arrays
- **tools.go**: `GetTools()` returns weather, time, and list_local_files tool definitions; also provides `ToolCallLoggingMiddleware()`
- **tool_workaround.go**: `ToolCallWorkaroundMiddleware()` intercepts chat responses and converts text-based tool calls to `FunctionCallContent`
- **server.go**: `agentServer` implements the HTTP hosting layer with `/invoke`, `/health`, and `/.well-known/agent-card.json` endpoints
- **main.go**: Wires everything together; `--serve` flag switches between CLI and HTTP mode
- **devtunnel.ps1**: Creates a Dev Tunnel to expose the local HTTP port externally
- **registration.ps1**: Validates reachability and prints fields for Foundry Control Plane registration

## Example Session

```
Starting Foundry Local with model "phi-4-mini"...
  downloading: 100%
Model loaded: phi-4-mini (GPU)
Endpoint:     http://localhost:5000/v1

Chat with the local assistant (type 'quit' to exit, 'stream' prefix for streaming)

You: what's the weather in seattle?
2025/01/15 10:30:00 INFO converted text to tool calls count=1
Assistant: The weather in Seattle is currently sunny and 72F.
  [tokens: 150 in, 25 out]

You: what time is it?
2025/01/15 10:30:00 INFO converted text to tool calls count=1
Assistant: The current time is 2025-01-15T10:30:00Z.
  [tokens: 175 in, 20 out]

You: quit
```

## Comparison with Cloud Sample

| Feature | Local Sample | Chat Sample (Cloud) |
|---------|--------------|---------------------|
| Model Location | On-device | Cloud API |
| Tool Calling | Workaround required | Native support |
| HTTP Server | --serve flag | N/A |
| Setup | Auto-downloads model | API key only |
| Latency | Depends on hardware | Network dependent |
| Privacy | 100% local | Data sent to cloud |
| Reliability | Model-dependent | Production-ready |

## Troubleshooting

**Model doesn't call tools correctly:**
- Try the chat sample with Azure Foundry/OpenAI to verify tool definitions
- Check that model supports instruction following
- Inspect logs for "converted text to tool calls" messages

**Out of memory errors:**
- Try a smaller model
- Reduce `maxTokens` in main.go
- Use CPU instead of GPU if VRAM is limited

**Service won't start:**
- Verify Foundry Local is installed correctly
- Check if port 5000 is already in use
- See Foundry Local documentation for diagnostics
