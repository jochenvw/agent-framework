# Go Agent Framework Samples

This directory contains example applications demonstrating the Agent Framework.

## Samples

### [chat](chat/) - Cloud Chat Agent

Multi-turn conversational agent using cloud models (OpenAI, Azure Foundry).

**Features:**
- Azure AD authentication support
- API key authentication
- Typed and raw tool definitions
- Streaming responses
- Session-based conversations
- Environment variable configuration

**Use when:** You need production-ready tool calling with cloud models.

### [local](local/) - Local Agent with Foundry Local

On-device conversational agent using Foundry Local for complete privacy.

**Features:**
- 100% local inference
- Auto-download and load models
- Device selection (CPU/GPU/NPU)
- Tool calling workaround for local models
- Session-based conversations

**Use when:** You need privacy, offline capability, or want to experiment with local models.

## Quick Start

### Cloud Agent (Recommended for tool calling)

```bash
cd chat
cp .env.sample .env
# Edit .env with your credentials
go run .
```

### Local Agent (Privacy-focused)

```bash
cd local
go run .  # Auto-downloads phi-4-mini
```

## Comparison

| Aspect | Chat (Cloud) | Local |
|--------|--------------|-------|
| **Setup** | API key or Azure AD | Auto-downloads model |
| **Privacy** | Data sent to cloud | 100% on-device |
| **Tool Calling** | Native support ✓ | Workaround (experimental) |
| **Performance** | Fast, consistent | Hardware-dependent |
| **Cost** | Pay per token | Free after model download |
| **Reliability** | Production-ready | Model-dependent |
| **Best For** | Production apps | Prototyping, privacy |

## Prerequisites

- Go 1.22 or later
- For `chat`: OpenAI/Azure API credentials
- For `local`: Foundry Local installed

## Architecture

Both samples use the same framework:

```
Agent → ChatClient → HTTP API
  ↓
Tools (get_weather, get_time)
  ↓
Middleware (Logging, Tool Workaround)
  ↓
Session (Multi-turn context)
```

## Next Steps

1. Start with `chat` to understand the framework basics
2. Experiment with `local` for privacy-focused scenarios
3. Review tool definitions in each sample to see both typed and raw approaches
4. Explore middleware patterns for logging and custom behavior

## Support

- Framework README: [../README.md](../README.md)
- Tool Calling Guide: [../README.md#tools](../README.md#tools)
- Middleware Guide: [../README.md#middleware](../README.md#middleware)
