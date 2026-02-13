// Copyright (c) Microsoft. All rights reserved.

package agentframework_test

import (
	"context"
	"encoding/json"
	"testing"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

func TestChainMiddleware_ExecutionOrder(t *testing.T) {
	var order []string

	mw1 := af.AgentMiddleware(func(next af.AgentHandler) af.AgentHandler {
		return func(ctx context.Context, req *af.AgentRequest) (*af.AgentResponse, error) {
			order = append(order, "mw1-before")
			resp, err := next(ctx, req)
			order = append(order, "mw1-after")
			return resp, err
		}
	})

	mw2 := af.AgentMiddleware(func(next af.AgentHandler) af.AgentHandler {
		return func(ctx context.Context, req *af.AgentRequest) (*af.AgentResponse, error) {
			order = append(order, "mw2-before")
			resp, err := next(ctx, req)
			order = append(order, "mw2-after")
			return resp, err
		}
	})

	// Create a mock client and agent
	client := &mockClient{
		responseFn: func(ctx context.Context, msgs []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error) {
			return &af.ChatResponse{
				Messages: []af.Message{af.NewAssistantMessage("ok")},
			}, nil
		},
	}

	agent := af.NewAgent(client, af.WithAgentMiddleware(mw1, mw2))
	_, err := agent.Run(context.Background(), []af.Message{af.NewUserMessage("hi")})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// First middleware should be outermost
	expected := []string{"mw1-before", "mw2-before", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestFunctionMiddleware(t *testing.T) {
	var interceptedToolName string

	fnMw := af.FunctionMiddleware(func(next af.FunctionHandler) af.FunctionHandler {
		return func(ctx context.Context, tool af.Tool, args json.RawMessage) (any, error) {
			interceptedToolName = tool.Name()
			return next(ctx, tool, args)
		}
	})

	// Create an agent with a tool and function middleware
	tool := af.NewTool("echo", "Echoes input", json.RawMessage(`{"type":"object"}`),
		func(ctx context.Context, args json.RawMessage) (any, error) {
			return "echoed", nil
		},
	)

	callCount := 0
	client := &mockClient{
		responseFn: func(ctx context.Context, msgs []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				// First call: model requests tool call
				return &af.ChatResponse{
					Messages: []af.Message{{
						Role: af.RoleAssistant,
						Contents: af.Contents{
							&af.FunctionCallContent{CallID: "c1", Name: "echo", Arguments: `{}`},
						},
					}},
				}, nil
			}
			// Second call: model returns final response
			return &af.ChatResponse{
				Messages: []af.Message{af.NewAssistantMessage("done")},
			}, nil
		},
	}

	agent := af.NewAgent(client,
		af.WithTools(tool),
		af.WithFunctionMiddleware(fnMw),
	)

	_, err := agent.Run(context.Background(), []af.Message{af.NewUserMessage("test")})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if interceptedToolName != "echo" {
		t.Errorf("intercepted tool = %q, want echo", interceptedToolName)
	}
}

// mockClient implements ChatClient for testing.
type mockClient struct {
	responseFn func(ctx context.Context, msgs []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error)
}

func (m *mockClient) Response(ctx context.Context, msgs []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error) {
	return m.responseFn(ctx, msgs, opts)
}

func (m *mockClient) StreamResponse(ctx context.Context, msgs []af.Message, opts *af.ChatOptions) (*af.ResponseStream[af.ChatResponseUpdate], error) {
	return af.NewResponseStream(ctx, func(ctx context.Context, ch chan<- af.ChatResponseUpdate) error {
		resp, err := m.responseFn(ctx, msgs, opts)
		if err != nil {
			return err
		}
		for _, msg := range resp.Messages {
			ch <- af.ChatResponseUpdate{
				Contents: msg.Contents,
				Role:     msg.Role,
			}
		}
		return nil
	}), nil
}
