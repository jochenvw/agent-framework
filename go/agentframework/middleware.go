// Copyright (c) Microsoft. All rights reserved.

package agentframework

import (
	"context"
	"encoding/json"
)

// AgentHandler is the function signature for processing an agent run.
type AgentHandler func(ctx context.Context, req *AgentRequest) (*AgentResponse, error)

// AgentRequest carries the inputs for an agent run through the middleware pipeline.
type AgentRequest struct {
	Messages []Message
	Session  *Session
	Options  *ChatOptions
}

// AgentMiddleware wraps an [AgentHandler] to add cross-cutting behavior.
// Middleware should call next to continue the chain, or return early to short-circuit.
type AgentMiddleware func(next AgentHandler) AgentHandler

// ChatHandler is the function signature for processing a chat request.
type ChatHandler func(ctx context.Context, messages []Message, opts *ChatOptions) (*ChatResponse, error)

// ChatMiddleware wraps a [ChatHandler] to add cross-cutting behavior.
type ChatMiddleware func(next ChatHandler) ChatHandler

// FunctionHandler is the function signature for invoking a tool.
type FunctionHandler func(ctx context.Context, tool Tool, args json.RawMessage) (any, error)

// FunctionMiddleware wraps a [FunctionHandler] to add cross-cutting behavior.
type FunctionMiddleware func(next FunctionHandler) FunctionHandler

// chainAgentMiddleware applies middleware in order (first in list = outermost wrapper).
func chainAgentMiddleware(handler AgentHandler, mws ...AgentMiddleware) AgentHandler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
}

// chainChatMiddleware applies middleware in order (first in list = outermost wrapper).
func chainChatMiddleware(handler ChatHandler, mws ...ChatMiddleware) ChatHandler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
}

// chainFunctionMiddleware applies middleware in order.
func chainFunctionMiddleware(handler FunctionHandler, mws ...FunctionMiddleware) FunctionHandler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
}
