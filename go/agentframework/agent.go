// Copyright (c) Microsoft. All rights reserved.

package agentframework

import (
	"context"
	"fmt"
	"log/slog"
)

// Agent is the top-level conversational agent. It composes a [ChatClient] with
// tools, middleware, session management, and context providers.
//
// Create one with [NewAgent] and functional options:
//
//	agent := agentframework.NewAgent(client,
//	    agentframework.WithName("assistant"),
//	    agentframework.WithInstructions("You are helpful."),
//	    agentframework.WithTools(weatherTool),
//	)
type Agent struct {
	id                   string
	name                 string
	description          string
	client               ChatClient
	instructions         string
	tools                []Tool
	defaultOptions       *ChatOptions
	messageStoreFactory  func() MessageStore
	contextProvider      ContextProvider
	agentMiddleware      []AgentMiddleware
	chatMiddleware       []ChatMiddleware
	functionMiddleware   []FunctionMiddleware
	invocationConfig     InvocationConfig
}

// AgentOption configures an [Agent] via [NewAgent].
type AgentOption func(*Agent)

// WithName sets the agent's display name.
func WithName(name string) AgentOption {
	return func(a *Agent) { a.name = name }
}

// WithDescription sets the agent's description.
func WithDescription(desc string) AgentOption {
	return func(a *Agent) { a.description = desc }
}

// WithInstructions sets the system instructions for the agent.
func WithInstructions(instructions string) AgentOption {
	return func(a *Agent) { a.instructions = instructions }
}

// WithTools adds tools to the agent's default tool set.
func WithTools(tools ...Tool) AgentOption {
	return func(a *Agent) { a.tools = append(a.tools, tools...) }
}

// WithDefaultOptions sets default [ChatOptions] for all requests.
func WithDefaultOptions(opts *ChatOptions) AgentOption {
	return func(a *Agent) { a.defaultOptions = opts }
}

// WithMessageStoreFactory sets a factory for creating message stores
// when a session is initialized in local mode.
func WithMessageStoreFactory(f func() MessageStore) AgentOption {
	return func(a *Agent) { a.messageStoreFactory = f }
}

// WithContextProvider attaches a [ContextProvider] for dynamic context injection.
func WithContextProvider(cp ContextProvider) AgentOption {
	return func(a *Agent) { a.contextProvider = cp }
}

// WithAgentMiddleware adds [AgentMiddleware] to the agent pipeline.
func WithAgentMiddleware(mws ...AgentMiddleware) AgentOption {
	return func(a *Agent) { a.agentMiddleware = append(a.agentMiddleware, mws...) }
}

// WithChatMiddleware adds [ChatMiddleware] to the chat pipeline.
func WithChatMiddleware(mws ...ChatMiddleware) AgentOption {
	return func(a *Agent) { a.chatMiddleware = append(a.chatMiddleware, mws...) }
}

// WithFunctionMiddleware adds [FunctionMiddleware] to the tool invocation pipeline.
func WithFunctionMiddleware(mws ...FunctionMiddleware) AgentOption {
	return func(a *Agent) { a.functionMiddleware = append(a.functionMiddleware, mws...) }
}

// WithInvocationConfig overrides the default [InvocationConfig] for the
// function calling loop.
func WithInvocationConfig(cfg InvocationConfig) AgentOption {
	return func(a *Agent) { a.invocationConfig = cfg }
}

// NewAgent creates an Agent with the given [ChatClient] and options.
func NewAgent(client ChatClient, opts ...AgentOption) *Agent {
	a := &Agent{
		id:               newUUID(),
		client:           client,
		invocationConfig: DefaultInvocationConfig(),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// ID returns the agent's unique identifier.
func (a *Agent) ID() string { return a.id }

// Name returns the agent's display name.
func (a *Agent) Name() string { return a.name }

// Description returns the agent's description.
func (a *Agent) Description() string { return a.description }

// RunOption configures a single [Run] or [RunStream] call.
type RunOption func(*runConfig)

type runConfig struct {
	session *Session
	tools   []Tool
	options *ChatOptions
}

// WithSession attaches a [Session] for multi-turn conversation.
func WithSession(s *Session) RunOption {
	return func(c *runConfig) { c.session = s }
}

// WithRunTools provides per-call tool overrides (merged with agent defaults).
func WithRunTools(tools ...Tool) RunOption {
	return func(c *runConfig) { c.tools = tools }
}

// WithRunOptions provides per-call [ChatOptions] overrides.
func WithRunOptions(opts *ChatOptions) RunOption {
	return func(c *runConfig) { c.options = opts }
}

// Run sends messages to the agent and returns a complete response.
func (a *Agent) Run(ctx context.Context, messages []Message, opts ...RunOption) (*AgentResponse, error) {
	cfg := a.buildRunConfig(opts)

	// Build the inner handler
	handler := a.buildHandler(cfg)

	// Wrap with agent middleware
	wrapped := chainAgentMiddleware(handler, a.agentMiddleware...)

	req := &AgentRequest{
		Messages: messages,
		Session:  cfg.session,
		Options:  cfg.options,
	}

	return wrapped(ctx, req)
}

// RunStream sends messages to the agent and returns a streaming response.
func (a *Agent) RunStream(ctx context.Context, messages []Message, opts ...RunOption) (*AgentResponseStream, error) {
	cfg := a.buildRunConfig(opts)

	// For streaming, we produce an AgentResponseStream that maps from ChatResponseUpdate
	chatOpts := a.prepareChatOptions(cfg)
	allMessages, err := a.prepareMessages(ctx, messages, cfg, chatOpts)
	if err != nil {
		return nil, err
	}

	// Apply chat middleware to the streaming path
	chatStream, err := a.client.StreamResponse(ctx, allMessages, chatOpts)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrExecution, err)
	}

	// Map ChatResponseUpdate → AgentResponseUpdate
	agentStream := MapStream(ctx, chatStream, func(u ChatResponseUpdate) AgentResponseUpdate {
		return AgentResponseUpdate{
			Contents:   u.Contents,
			Role:       u.Role,
			AgentID:    a.id,
			ResponseID: u.ResponseID,
			Usage:      u.Usage,
			Raw:        u.Raw,
		}
	})

	return NewAgentResponseStream(agentStream), nil
}

// NewSession creates a new [Session] pre-configured for this agent.
func (a *Agent) NewSession() *Session {
	var store MessageStore
	if a.messageStoreFactory != nil {
		store = a.messageStoreFactory()
	} else {
		store = NewInMemoryStore()
	}
	return NewSession(
		WithSessionStore(store),
		WithSessionContextProvider(a.contextProvider),
	)
}

func (a *Agent) buildRunConfig(opts []RunOption) *runConfig {
	cfg := &runConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func (a *Agent) prepareChatOptions(cfg *runConfig) *ChatOptions {
	// Start with default options
	opts := MergeChatOptions(a.defaultOptions, cfg.options)

	// Merge tools: agent defaults + per-call overrides
	allTools := make([]Tool, 0, len(a.tools)+len(cfg.tools))
	allTools = append(allTools, a.tools...)
	allTools = append(allTools, cfg.tools...)
	if len(allTools) > 0 {
		opts.Tools = allTools
	}

	// Set instructions
	if a.instructions != "" {
		if opts.Instructions != "" {
			opts.Instructions = a.instructions + "\n" + opts.Instructions
		} else {
			opts.Instructions = a.instructions
		}
	}

	return opts
}

func (a *Agent) prepareMessages(ctx context.Context, messages []Message, cfg *runConfig, opts *ChatOptions) ([]Message, error) {
	var allMessages []Message

	// Load history from session store
	if cfg.session != nil {
		if store := cfg.session.Store(); store != nil {
			history, err := store.ListMessages(ctx)
			if err != nil {
				return nil, fmt.Errorf("load session history: %w", err)
			}
			allMessages = append(allMessages, history...)
		}
		// Set conversation ID from session
		if sid := cfg.session.ServiceID(); sid != "" {
			opts.ConversationID = sid
		}
	}

	allMessages = append(allMessages, messages...)

	// Apply context provider
	cp := a.contextProvider
	if cfg.session != nil && cfg.session.ContextProvider() != nil {
		cp = cfg.session.ContextProvider()
	}
	if cp != nil {
		invCtx, err := cp.Invoking(ctx, allMessages)
		if err != nil {
			return nil, fmt.Errorf("context provider: %w", err)
		}
		if invCtx != nil {
			if invCtx.Instructions != "" {
				if opts.Instructions != "" {
					opts.Instructions += "\n" + invCtx.Instructions
				} else {
					opts.Instructions = invCtx.Instructions
				}
			}
			if len(invCtx.Messages) > 0 {
				allMessages = append(invCtx.Messages, allMessages...)
			}
			if len(invCtx.Tools) > 0 {
				opts.Tools = append(opts.Tools, invCtx.Tools...)
			}
		}
	}

	// Prepend system instructions
	allMessages = PrependInstructions(allMessages, opts.Instructions)

	return allMessages, nil
}

func (a *Agent) buildHandler(cfg *runConfig) AgentHandler {
	return func(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
		chatOpts := a.prepareChatOptions(cfg)
		allMessages, err := a.prepareMessages(ctx, req.Messages, cfg, chatOpts)
		if err != nil {
			return nil, err
		}

		slog.DebugContext(ctx, "agent run",
			"agent_id", a.id,
			"agent_name", a.name,
			"message_count", len(allMessages),
			"tool_count", len(chatOpts.Tools),
		)

		// If tools are present, use the function invocation loop
		var chatResp *ChatResponse
		if len(chatOpts.Tools) > 0 {
			chatResp, err = invokeFunctions(ctx, a.client, allMessages, chatOpts, a.invocationConfig, a.functionMiddleware)
		} else {
			chatResp, err = a.client.Response(ctx, allMessages, chatOpts)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrExecution, err)
		}

		// Update session
		if cfg.session != nil {
			if err := a.updateSession(ctx, cfg.session, req.Messages, chatResp); err != nil {
				slog.WarnContext(ctx, "failed to update session", "error", err)
			}
		}

		// Notify context provider
		cp := a.contextProvider
		if cfg.session != nil && cfg.session.ContextProvider() != nil {
			cp = cfg.session.ContextProvider()
		}
		if cp != nil {
			if err := cp.Invoked(ctx, req.Messages, chatResp.Messages); err != nil {
				slog.WarnContext(ctx, "context provider invoked hook failed", "error", err)
			}
		}

		return &AgentResponse{
			Messages:   chatResp.Messages,
			ResponseID: chatResp.ResponseID,
			AgentID:    a.id,
			Usage:      chatResp.Usage,
			Extra:      chatResp.Extra,
			Raw:        chatResp.Raw,
		}, nil
	}
}

func (a *Agent) updateSession(ctx context.Context, session *Session, request []Message, resp *ChatResponse) error {
	store := session.Store()
	if store == nil {
		// Check if response has a conversation ID → switch to service mode
		if resp.ConversationID != "" {
			return session.SetServiceID(resp.ConversationID)
		}
		// Initialize local store
		if a.messageStoreFactory != nil {
			store = a.messageStoreFactory()
		} else {
			store = NewInMemoryStore()
		}
		if err := session.SetStore(store); err != nil {
			return err
		}
	}

	// Persist messages
	if err := store.AddMessages(ctx, request); err != nil {
		return err
	}
	return store.AddMessages(ctx, resp.Messages)
}
