// Copyright (c) Microsoft. All rights reserved.

package agentframework_test

import (
	"context"
	"encoding/json"
	"testing"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

func TestNewTool_BasicInvocation(t *testing.T) {
	tool := af.NewTool("greet", "Says hello", json.RawMessage(`{"type":"object"}`),
		func(ctx context.Context, args json.RawMessage) (any, error) {
			return "hello!", nil
		},
	)

	if tool.Name() != "greet" {
		t.Errorf("Name = %q", tool.Name())
	}
	if tool.Description() != "Says hello" {
		t.Errorf("Description = %q", tool.Description())
	}
	if tool.DeclarationOnly() {
		t.Error("should not be declaration-only")
	}
	if tool.Approval() != "" {
		t.Errorf("Approval = %q", tool.Approval())
	}

	result, err := tool.Invoke(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if result != "hello!" {
		t.Errorf("result = %v", result)
	}
}

func TestNewTypedTool(t *testing.T) {
	type args struct {
		Name string `json:"name" jsonschema:"description=Person name,required"`
	}

	tool := af.NewTypedTool("greet", "Greet someone",
		func(ctx context.Context, a args) (any, error) {
			return "Hello, " + a.Name + "!", nil
		},
	)

	// Check schema was generated
	params := tool.Parameters()
	var schema map[string]any
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("schema type = %v", schema["type"])
	}

	// Invoke with valid args
	result, err := tool.Invoke(context.Background(), json.RawMessage(`{"name":"Alice"}`))
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if result != "Hello, Alice!" {
		t.Errorf("result = %v", result)
	}
}

func TestNewTypedTool_InvalidArgs(t *testing.T) {
	type args struct {
		Count int `json:"count"`
	}

	tool := af.NewTypedTool("counter", "Count things",
		func(ctx context.Context, a args) (any, error) {
			return a.Count, nil
		},
	)

	_, err := tool.Invoke(context.Background(), json.RawMessage(`{"count":"not a number"}`))
	if err == nil {
		t.Fatal("expected error for invalid args")
	}
}

func TestToolOption_ApprovalRequired(t *testing.T) {
	tool := af.NewTool("risky", "Does risky things", nil,
		func(ctx context.Context, args json.RawMessage) (any, error) { return nil, nil },
		af.WithApprovalRequired(),
	)
	if tool.Approval() != af.ApprovalAlways {
		t.Errorf("Approval = %q, want %q", tool.Approval(), af.ApprovalAlways)
	}
}

func TestToolOption_DeclarationOnly(t *testing.T) {
	tool := af.NewTool("decl", "Declaration only", nil, nil,
		af.WithDeclarationOnly(),
	)
	if !tool.DeclarationOnly() {
		t.Error("should be declaration-only")
	}

	// Invoking a nil fn should error
	_, err := tool.Invoke(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error invoking declaration-only tool with nil fn")
	}
}
