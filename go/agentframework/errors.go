// Copyright (c) Microsoft. All rights reserved.

package agentframework

import (
	"errors"
	"fmt"
)

// Sentinel errors for use with errors.Is.
var (
	// ErrAgent is the base error for agent-related failures.
	ErrAgent = errors.New("agent error")

	// ErrExecution indicates a runtime failure during agent execution.
	ErrExecution = fmt.Errorf("%w: execution", ErrAgent)

	// ErrInitialization indicates an agent configuration or setup failure.
	ErrInitialization = fmt.Errorf("%w: initialization", ErrAgent)

	// ErrSession indicates a session lifecycle failure.
	ErrSession = fmt.Errorf("%w: session", ErrAgent)

	// ErrSessionModeLocked is returned when attempting to change a session's
	// mode (service-managed vs local) after it has been set.
	ErrSessionModeLocked = fmt.Errorf("%w: mode already set", ErrSession)

	// ErrChatClient is the base error for chat client failures.
	ErrChatClient = errors.New("chat client error")

	// ErrService is the base error for backend service failures.
	ErrService = errors.New("service error")

	// ErrContentFilter indicates the request was rejected by a content filter.
	ErrContentFilter = fmt.Errorf("%w: content filter", ErrService)

	// ErrInvalidRequest indicates the request was malformed or invalid.
	ErrInvalidRequest = fmt.Errorf("%w: invalid request", ErrService)

	// ErrInvalidResponse indicates the service returned an unexpected response.
	ErrInvalidResponse = fmt.Errorf("%w: invalid response", ErrService)

	// ErrAuth indicates an authentication or authorization failure.
	ErrAuth = fmt.Errorf("%w: authentication", ErrService)

	// ErrTool is the base error for tool-related failures.
	ErrTool = errors.New("tool error")

	// ErrToolExecution indicates a failure during tool invocation.
	ErrToolExecution = fmt.Errorf("%w: execution", ErrTool)

	// ErrMiddleware is the base error for middleware failures.
	ErrMiddleware = errors.New("middleware error")
)

// ServiceError provides rich context for backend service failures.
// Use errors.As to extract it from a wrapped error chain.
type ServiceError struct {
	StatusCode int
	Message    string
	Code       string
	Err        error
}

func (e *ServiceError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("service error %d (%s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("service error %d: %s", e.StatusCode, e.Message)
}

func (e *ServiceError) Unwrap() error { return e.Err }

// ToolError provides context for tool invocation failures.
type ToolError struct {
	ToolName string
	Message  string
	Err      error
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("tool %q: %s", e.ToolName, e.Message)
}

func (e *ToolError) Unwrap() error { return e.Err }
