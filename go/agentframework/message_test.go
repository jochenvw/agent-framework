// Copyright (c) Microsoft. All rights reserved.

package agentframework_test

import (
	"testing"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

func TestNewUserMessage(t *testing.T) {
	m := af.NewUserMessage("hi")
	if m.Role != af.RoleUser {
		t.Errorf("role = %q, want %q", m.Role, af.RoleUser)
	}
	if m.Text() != "hi" {
		t.Errorf("text = %q, want %q", m.Text(), "hi")
	}
}

func TestNewAssistantMessage(t *testing.T) {
	m := af.NewAssistantMessage("hello")
	if m.Role != af.RoleAssistant {
		t.Errorf("role = %q", m.Role)
	}
	if m.Text() != "hello" {
		t.Errorf("text = %q", m.Text())
	}
}

func TestNewToolMessage(t *testing.T) {
	m := af.NewToolMessage("call-1", "result")
	if m.Role != af.RoleTool {
		t.Errorf("role = %q", m.Role)
	}
	if len(m.Contents) != 1 {
		t.Fatalf("contents len = %d", len(m.Contents))
	}
	fr, ok := m.Contents[0].(*af.FunctionResultContent)
	if !ok {
		t.Fatalf("type = %T", m.Contents[0])
	}
	if fr.CallID != "call-1" {
		t.Errorf("CallID = %q", fr.CallID)
	}
}

func TestMessageText_MultipleContents(t *testing.T) {
	m := af.Message{
		Role: af.RoleAssistant,
		Contents: af.Contents{
			&af.TextContent{Text: "Hello "},
			&af.FunctionCallContent{Name: "fn"}, // non-text: skipped
			&af.TextContent{Text: "World"},
		},
	}
	if got := m.Text(); got != "Hello World" {
		t.Errorf("text = %q, want %q", got, "Hello World")
	}
}

func TestNormalizeMessages(t *testing.T) {
	msgs := af.NormalizeMessages(
		"hello",
		af.NewAssistantMessage("hi"),
		[]af.Message{af.NewSystemMessage("sys")},
	)
	if len(msgs) != 3 {
		t.Fatalf("len = %d, want 3", len(msgs))
	}
	if msgs[0].Role != af.RoleUser {
		t.Errorf("[0].Role = %q", msgs[0].Role)
	}
	if msgs[1].Role != af.RoleAssistant {
		t.Errorf("[1].Role = %q", msgs[1].Role)
	}
	if msgs[2].Role != af.RoleSystem {
		t.Errorf("[2].Role = %q", msgs[2].Role)
	}
}

func TestPrependInstructions(t *testing.T) {
	msgs := []af.Message{af.NewUserMessage("hi")}

	// With instructions
	result := af.PrependInstructions(msgs, "Be helpful")
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0].Role != af.RoleSystem {
		t.Errorf("[0].Role = %q", result[0].Role)
	}
	if result[0].Text() != "Be helpful" {
		t.Errorf("[0].Text() = %q", result[0].Text())
	}

	// Empty instructions: no change
	result2 := af.PrependInstructions(msgs, "")
	if len(result2) != 1 {
		t.Errorf("empty instructions should not add message, got len=%d", len(result2))
	}

	// Already has system message: no duplicate
	withSys := []af.Message{af.NewSystemMessage("existing"), af.NewUserMessage("hi")}
	result3 := af.PrependInstructions(withSys, "new")
	if len(result3) != 2 {
		t.Errorf("should not add duplicate system message, got len=%d", len(result3))
	}
}
