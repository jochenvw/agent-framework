// Copyright (c) Microsoft. All rights reserved.

package agentframework

import "context"

// ContextProvider injects dynamic context into each agent invocation.
// Implementations can supply additional instructions, messages, or tools
// based on runtime state (e.g., RAG retrieval, memory lookup).
type ContextProvider interface {
	// Invoking is called before each agent run. The returned InvocationContext
	// is merged into the request (instructions concatenated, messages prepended,
	// tools added).
	Invoking(ctx context.Context, messages []Message) (*InvocationContext, error)

	// Invoked is called after each agent run with the request and response messages.
	Invoked(ctx context.Context, request, response []Message) error

	// SessionCreated is called when a new session is created.
	SessionCreated(ctx context.Context, sessionID string) error
}

// InvocationContext holds the dynamic context returned by a [ContextProvider].
type InvocationContext struct {
	// Instructions to append to the system prompt.
	Instructions string

	// Messages to prepend to the conversation.
	Messages []Message

	// Tools to add to the available tool set.
	Tools []Tool
}

// NoOpContextProvider is a [ContextProvider] that does nothing.
// Embed it to provide default implementations for unused hooks.
type NoOpContextProvider struct{}

func (NoOpContextProvider) Invoking(_ context.Context, _ []Message) (*InvocationContext, error) {
	return &InvocationContext{}, nil
}

func (NoOpContextProvider) Invoked(_ context.Context, _, _ []Message) error {
	return nil
}

func (NoOpContextProvider) SessionCreated(_ context.Context, _ string) error {
	return nil
}
