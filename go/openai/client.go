// Copyright (c) Microsoft. All rights reserved.

// Package openai provides a [ChatClient] implementation backed by the
// OpenAI Chat Completions API.
//
// Create a client with [New] and pass it to [agentframework.NewAgent]:
//
//	client := openai.New(apiKey, openai.WithModel("gpt-4o"))
//	agent  := agentframework.NewAgent(client)
package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

// Client implements [agentframework.ChatClient] using the OpenAI Chat
// Completions API. Use [New] to create one.
type Client struct {
	tp      transport
	model   string
	handler af.ChatHandler
}

// Verify interface compliance at compile time.
var _ af.ChatClient = (*Client)(nil)

// New creates an OpenAI [Client] with the given API key and options.
//
//	client := openai.New(os.Getenv("OPENAI_API_KEY"),
//	    openai.WithModel("gpt-4o"),
//	)
func New(apiKey string, opts ...Option) *Client {
	cfg := &clientConfig{}
	for _, o := range opts {
		o(cfg)
	}
	c := &Client{
		tp:    newHTTPTransport(apiKey, cfg),
		model: cfg.model,
	}
	// Set up core handler
	c.handler = c.coreResponse
	// Apply middleware in order
	for i := len(cfg.chatMiddleware) - 1; i >= 0; i-- {
		c.handler = cfg.chatMiddleware[i](c.handler)
	}
	return c
}

// newWithTransport creates a Client with a custom transport (for testing).
func newWithTransport(tp transport, model string) *Client {
	c := &Client{tp: tp, model: model}
	c.handler = c.coreResponse
	return c
}

// Response sends a non-streaming chat completion request and returns the
// complete response.
func (c *Client) Response(ctx context.Context, messages []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error) {
	return c.handler(ctx, messages, opts)
}

// coreResponse is the base implementation called by the middleware chain.
func (c *Client) coreResponse(ctx context.Context, messages []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error) {
	req := buildRequest(messages, opts, c.model)
	req.Stream = false

	resp, err := c.tp.do(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read response body: %v", af.ErrService, err)
	}

	raw, err := unmarshalChatResponse(body)
	if err != nil {
		return nil, fmt.Errorf("%w: parse response: %v", af.ErrService, err)
	}

	result := parseChatResponse(raw)
	result.Raw = raw
	return result, nil
}

// StreamResponse sends a streaming chat completion request and returns
// a [ResponseStream] that yields incremental updates via server-sent events.
func (c *Client) StreamResponse(ctx context.Context, messages []af.Message, opts *af.ChatOptions) (*af.ResponseStream[af.ChatResponseUpdate], error) {
	req := buildRequest(messages, opts, c.model)
	req.Stream = true
	req.StreamOptions = &streamOptions{IncludeUsage: true}

	resp, err := c.tp.do(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return nil, err
	}

	stream := af.NewResponseStream[af.ChatResponseUpdate](ctx, func(ctx context.Context, ch chan<- af.ChatResponseUpdate) error {
		defer resp.Body.Close()
		return parseSSEStream(ctx, resp.Body, ch)
	})

	return stream, nil
}

// parseSSEStream reads OpenAI server-sent events from r and sends parsed
// updates to ch. It returns when the stream is exhausted ([DONE]),
// the context is cancelled, or an error occurs.
func parseSSEStream(ctx context.Context, r io.Reader, ch chan<- af.ChatResponseUpdate) error {
	scanner := bufio.NewScanner(r)
	// Allow large SSE lines (some responses can be substantial).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// SSE format: lines starting with "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		data = strings.TrimSpace(data)

		// Stream terminator.
		if data == "[DONE]" {
			return nil
		}

		var chunk chatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Skip malformed chunks rather than aborting.
			continue
		}

		update := parseChunk(&chunk)
		update.Raw = &chunk

		select {
		case ch <- *update:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%w: read SSE stream: %v", af.ErrService, err)
	}

	return nil
}
