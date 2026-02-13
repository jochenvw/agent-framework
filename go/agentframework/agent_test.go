// Copyright (c) Microsoft. All rights reserved.

package agentframework_test

import (
	"context"
	"testing"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

func TestAgent_BasicRun(t *testing.T) {
	client := &mockClient{
		responseFn: func(ctx context.Context, msgs []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error) {
			return &af.ChatResponse{
				Messages:   []af.Message{af.NewAssistantMessage("I'm here to help!")},
				ResponseID: "resp-1",
				Usage:      af.UsageDetails{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
			}, nil
		},
	}

	agent := af.NewAgent(client,
		af.WithName("test-agent"),
		af.WithInstructions("You are helpful."),
	)

	if agent.Name() != "test-agent" {
		t.Errorf("Name = %q", agent.Name())
	}
	if agent.ID() == "" {
		t.Error("ID should not be empty")
	}

	resp, err := agent.Run(context.Background(), []af.Message{af.NewUserMessage("hi")})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if resp.Text() != "I'm here to help!" {
		t.Errorf("Text = %q", resp.Text())
	}
	if resp.AgentID != agent.ID() {
		t.Errorf("AgentID = %q, want %q", resp.AgentID, agent.ID())
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d", resp.Usage.TotalTokens)
	}
}

func TestAgent_WithToolInvocation(t *testing.T) {
	tool := af.NewTypedTool("add", "Adds two numbers",
		func(ctx context.Context, args struct {
			A int `json:"a"`
			B int `json:"b"`
		}) (any, error) {
			return args.A + args.B, nil
		},
	)

	callCount := 0
	client := &mockClient{
		responseFn: func(ctx context.Context, msgs []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return &af.ChatResponse{
					Messages: []af.Message{{
						Role: af.RoleAssistant,
						Contents: af.Contents{
							&af.FunctionCallContent{
								CallID:    "call-1",
								Name:      "add",
								Arguments: `{"a":3,"b":4}`,
							},
						},
					}},
				}, nil
			}
			return &af.ChatResponse{
				Messages: []af.Message{af.NewAssistantMessage("The answer is 7.")},
			}, nil
		},
	}

	agent := af.NewAgent(client, af.WithTools(tool))
	resp, err := agent.Run(context.Background(), []af.Message{af.NewUserMessage("what is 3+4?")})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if callCount != 2 {
		t.Errorf("client called %d times, want 2", callCount)
	}
	if resp.Text() != "The answer is 7." {
		t.Errorf("Text = %q", resp.Text())
	}
}

func TestAgent_WithSession(t *testing.T) {
	callCount := 0
	client := &mockClient{
		responseFn: func(ctx context.Context, msgs []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error) {
			callCount++
			messageCount := len(msgs)
			return &af.ChatResponse{
				Messages: []af.Message{af.NewAssistantMessage("ok")},
				Extra:    map[string]any{"message_count": messageCount},
			}, nil
		},
	}

	agent := af.NewAgent(client, af.WithInstructions("Be helpful"))
	session := agent.NewSession()

	// First turn
	resp1, err := agent.Run(context.Background(),
		[]af.Message{af.NewUserMessage("hello")},
		af.WithSession(session),
	)
	if err != nil {
		t.Fatalf("Run 1: %v", err)
	}
	_ = resp1

	// Second turn: session should carry history
	_, err = agent.Run(context.Background(),
		[]af.Message{af.NewUserMessage("what did I say?")},
		af.WithSession(session),
	)
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}

	// Verify session stores messages
	store := session.Store()
	if store == nil {
		t.Fatal("session store should be initialized")
	}
	msgs, _ := store.ListMessages(context.Background())
	// Should have: first user msg + first assistant response + second user msg + second assistant response
	if len(msgs) < 2 {
		t.Errorf("session has %d messages, want at least 2", len(msgs))
	}
}

func TestAgent_NewSession(t *testing.T) {
	client := &mockClient{
		responseFn: func(ctx context.Context, msgs []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error) {
			return &af.ChatResponse{Messages: []af.Message{af.NewAssistantMessage("ok")}}, nil
		},
	}

	agent := af.NewAgent(client)
	s := agent.NewSession()

	if s.ID() == "" {
		t.Error("session ID should not be empty")
	}
	if s.Store() == nil {
		t.Error("session should have an in-memory store by default")
	}
}

func TestAgent_RunWithOptions(t *testing.T) {
	var receivedModel string
	client := &mockClient{
		responseFn: func(ctx context.Context, msgs []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error) {
			if opts != nil {
				receivedModel = opts.ModelID
			}
			return &af.ChatResponse{Messages: []af.Message{af.NewAssistantMessage("ok")}}, nil
		},
	}

	agent := af.NewAgent(client)
	_, err := agent.Run(context.Background(),
		[]af.Message{af.NewUserMessage("hi")},
		af.WithRunOptions(&af.ChatOptions{ModelID: "gpt-4o-mini"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	if receivedModel != "gpt-4o-mini" {
		t.Errorf("model = %q, want gpt-4o-mini", receivedModel)
	}
}
