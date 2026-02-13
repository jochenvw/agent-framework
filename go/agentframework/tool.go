// Copyright (c) Microsoft. All rights reserved.

package agentframework

import (
	"context"
	"encoding/json"
)

// ApprovalMode controls whether a tool requires human approval before invocation.
type ApprovalMode string

const (
	ApprovalNever  ApprovalMode = "never"
	ApprovalAlways ApprovalMode = "always"
)

// Tool defines a callable function that can be exposed to an LLM.
type Tool interface {
	// Name returns the function name as exposed to the model.
	Name() string

	// Description returns a human-readable description for the model.
	Description() string

	// Parameters returns the JSON Schema describing the function's input.
	Parameters() json.RawMessage

	// Invoke calls the function with the given JSON arguments.
	Invoke(ctx context.Context, args json.RawMessage) (any, error)

	// DeclarationOnly returns true if the tool should not be auto-invoked.
	DeclarationOnly() bool

	// Approval returns the approval mode for this tool.
	Approval() ApprovalMode
}

// FunctionTool is a concrete [Tool] backed by a Go function.
type FunctionTool struct {
	name            string
	description     string
	parameters      json.RawMessage
	fn              func(ctx context.Context, args json.RawMessage) (any, error)
	declarationOnly bool
	approvalMode    ApprovalMode
	maxInvocations  int
}

// ToolOption configures a [FunctionTool].
type ToolOption func(*FunctionTool)

// WithApprovalRequired sets the tool to require human approval before invocation.
func WithApprovalRequired() ToolOption {
	return func(t *FunctionTool) { t.approvalMode = ApprovalAlways }
}

// WithDeclarationOnly marks the tool as declaration-only (returned to caller, not auto-invoked).
func WithDeclarationOnly() ToolOption {
	return func(t *FunctionTool) { t.declarationOnly = true }
}

// WithMaxInvocations limits how many times this tool can be called per request.
func WithMaxInvocations(n int) ToolOption {
	return func(t *FunctionTool) { t.maxInvocations = n }
}

// NewTool creates a [FunctionTool] with raw JSON schema and handler.
func NewTool(name, description string, parameters json.RawMessage, fn func(ctx context.Context, args json.RawMessage) (any, error), opts ...ToolOption) *FunctionTool {
	t := &FunctionTool{
		name:        name,
		description: description,
		parameters:  parameters,
		fn:          fn,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// NewTypedTool creates a [FunctionTool] that automatically generates JSON Schema
// from the Args type parameter and handles JSON deserialization.
//
// The Args type should be a struct with json tags. Use the `jsonschema` struct tag
// for additional schema metadata:
//
//	type WeatherArgs struct {
//	    Location string `json:"location" jsonschema:"description=City name,required"`
//	    Unit     string `json:"unit"     jsonschema:"description=Temperature unit,enum=celsius|fahrenheit"`
//	}
func NewTypedTool[Args any](name, description string, fn func(ctx context.Context, args Args) (any, error), opts ...ToolOption) *FunctionTool {
	schema := GenerateSchema[Args]()

	wrapped := func(ctx context.Context, raw json.RawMessage) (any, error) {
		var args Args
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, &ToolError{
				ToolName: name,
				Message:  "invalid arguments: " + err.Error(),
				Err:      ErrToolExecution,
			}
		}
		return fn(ctx, args)
	}

	return NewTool(name, description, schema, wrapped, opts...)
}

func (t *FunctionTool) Name() string              { return t.name }
func (t *FunctionTool) Description() string        { return t.description }
func (t *FunctionTool) Parameters() json.RawMessage { return t.parameters }
func (t *FunctionTool) DeclarationOnly() bool       { return t.declarationOnly }
func (t *FunctionTool) Approval() ApprovalMode      { return t.approvalMode }

// Invoke calls the tool's backing function.
func (t *FunctionTool) Invoke(ctx context.Context, args json.RawMessage) (any, error) {
	if t.fn == nil {
		return nil, &ToolError{
			ToolName: t.name,
			Message:  "tool is declaration-only and cannot be invoked",
			Err:      ErrToolExecution,
		}
	}
	return t.fn(ctx, args)
}

// GenerateSchema builds a JSON Schema from a Go struct type using reflection.
// Supports struct tags: json (field name), jsonschema (description, required, enum).
func GenerateSchema[T any]() json.RawMessage {
	var zero T
	return generateSchemaFromType(zero)
}
