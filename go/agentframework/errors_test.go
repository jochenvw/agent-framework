// Copyright (c) Microsoft. All rights reserved.

package agentframework_test

import (
	"errors"
	"testing"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

func TestErrorSentinelChain(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		match  bool
	}{
		{"ErrExecution wraps ErrAgent", af.ErrExecution, af.ErrAgent, true},
		{"ErrSession wraps ErrAgent", af.ErrSession, af.ErrAgent, true},
		{"ErrSessionModeLocked wraps ErrSession", af.ErrSessionModeLocked, af.ErrSession, true},
		{"ErrSessionModeLocked wraps ErrAgent", af.ErrSessionModeLocked, af.ErrAgent, true},
		{"ErrContentFilter wraps ErrService", af.ErrContentFilter, af.ErrService, true},
		{"ErrAuth wraps ErrService", af.ErrAuth, af.ErrService, true},
		{"ErrToolExecution wraps ErrTool", af.ErrToolExecution, af.ErrTool, true},
		{"ErrAgent does not wrap ErrService", af.ErrAgent, af.ErrService, false},
		{"ErrTool does not wrap ErrAgent", af.ErrTool, af.ErrAgent, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := errors.Is(tc.err, tc.target); got != tc.match {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", tc.err, tc.target, got, tc.match)
			}
		})
	}
}

func TestServiceError(t *testing.T) {
	svcErr := &af.ServiceError{
		StatusCode: 429,
		Message:    "rate limited",
		Code:       "rate_limit_exceeded",
		Err:        af.ErrService,
	}

	// Check error message
	msg := svcErr.Error()
	if msg == "" {
		t.Fatal("error message should not be empty")
	}

	// errors.Is should match ErrService
	if !errors.Is(svcErr, af.ErrService) {
		t.Error("ServiceError should wrap ErrService")
	}

	// errors.As should extract ServiceError
	var extracted *af.ServiceError
	if !errors.As(svcErr, &extracted) {
		t.Fatal("errors.As should extract ServiceError")
	}
	if extracted.StatusCode != 429 {
		t.Errorf("StatusCode = %d", extracted.StatusCode)
	}
}

func TestToolError(t *testing.T) {
	toolErr := &af.ToolError{
		ToolName: "get_weather",
		Message:  "API timeout",
		Err:      af.ErrToolExecution,
	}

	if !errors.Is(toolErr, af.ErrToolExecution) {
		t.Error("ToolError should wrap ErrToolExecution")
	}
	if !errors.Is(toolErr, af.ErrTool) {
		t.Error("ToolError should transitively wrap ErrTool")
	}

	var extracted *af.ToolError
	if !errors.As(toolErr, &extracted) {
		t.Fatal("errors.As should extract ToolError")
	}
	if extracted.ToolName != "get_weather" {
		t.Errorf("ToolName = %q", extracted.ToolName)
	}
}
