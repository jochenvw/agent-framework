// Copyright (c) Microsoft. All rights reserved.

package agentframework

import "context"

// ChatClient is the interface for interacting with an LLM backend.
// Provider packages (e.g., openai) implement this interface.
type ChatClient interface {
	// Response sends messages to the model and returns a complete response.
	Response(ctx context.Context, messages []Message, opts *ChatOptions) (*ChatResponse, error)

	// StreamResponse sends messages and returns a stream of incremental updates.
	StreamResponse(ctx context.Context, messages []Message, opts *ChatOptions) (*ResponseStream[ChatResponseUpdate], error)
}
