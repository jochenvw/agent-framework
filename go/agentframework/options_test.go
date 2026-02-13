// Copyright (c) Microsoft. All rights reserved.

package agentframework_test

import (
	"testing"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

func TestMergeChatOptions_NilBase(t *testing.T) {
	temp := 0.7
	override := &af.ChatOptions{Temperature: &temp, ModelID: "gpt-4o"}
	merged := af.MergeChatOptions(nil, override)

	if merged.ModelID != "gpt-4o" {
		t.Errorf("ModelID = %q", merged.ModelID)
	}
	if merged.Temperature == nil || *merged.Temperature != 0.7 {
		t.Errorf("Temperature = %v", merged.Temperature)
	}
}

func TestMergeChatOptions_NilOverride(t *testing.T) {
	base := &af.ChatOptions{ModelID: "gpt-3.5"}
	merged := af.MergeChatOptions(base, nil)

	if merged.ModelID != "gpt-3.5" {
		t.Errorf("ModelID = %q", merged.ModelID)
	}
}

func TestMergeChatOptions_BothNil(t *testing.T) {
	merged := af.MergeChatOptions(nil, nil)
	if merged == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestMergeChatOptions_OverrideWins(t *testing.T) {
	baseTemp := 0.5
	overTemp := 0.9
	base := &af.ChatOptions{
		ModelID:     "base-model",
		Temperature: &baseTemp,
		User:        "user1",
	}
	override := &af.ChatOptions{
		ModelID:     "override-model",
		Temperature: &overTemp,
	}
	merged := af.MergeChatOptions(base, override)

	if merged.ModelID != "override-model" {
		t.Errorf("ModelID = %q, want override-model", merged.ModelID)
	}
	if *merged.Temperature != 0.9 {
		t.Errorf("Temperature = %f, want 0.9", *merged.Temperature)
	}
	if merged.User != "user1" {
		t.Errorf("User = %q, want user1 (preserved from base)", merged.User)
	}
}

func TestMergeChatOptions_InstructionsConcatenate(t *testing.T) {
	base := &af.ChatOptions{Instructions: "Be helpful"}
	override := &af.ChatOptions{Instructions: "Be concise"}
	merged := af.MergeChatOptions(base, override)

	expected := "Be helpful\nBe concise"
	if merged.Instructions != expected {
		t.Errorf("Instructions = %q, want %q", merged.Instructions, expected)
	}
}

func TestMergeChatOptions_MetadataMerge(t *testing.T) {
	base := &af.ChatOptions{
		Metadata: map[string]string{"a": "1", "b": "2"},
	}
	override := &af.ChatOptions{
		Metadata: map[string]string{"b": "override", "c": "3"},
	}
	merged := af.MergeChatOptions(base, override)

	if merged.Metadata["a"] != "1" {
		t.Errorf("metadata[a] = %q", merged.Metadata["a"])
	}
	if merged.Metadata["b"] != "override" {
		t.Errorf("metadata[b] = %q, want override", merged.Metadata["b"])
	}
	if merged.Metadata["c"] != "3" {
		t.Errorf("metadata[c] = %q", merged.Metadata["c"])
	}
}

func TestToolChoiceFunction(t *testing.T) {
	tc := af.ToolChoiceFunction("get_weather")
	expected := af.ToolChoice("function:get_weather")
	if tc != expected {
		t.Errorf("ToolChoiceFunction = %q, want %q", tc, expected)
	}
}
