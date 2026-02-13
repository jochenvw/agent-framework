// Copyright (c) Microsoft. All rights reserved.

package agentframework

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// InvocationConfig controls the function invocation loop behavior.
type InvocationConfig struct {
	// MaxIterations is the maximum number of LLM round-trips for tool calling.
	// Default: 40.
	MaxIterations int

	// MaxConsecutiveErrors is the maximum number of consecutive tool errors
	// before aborting. Default: 3.
	MaxConsecutiveErrors int

	// TerminateOnUnknown aborts if the model calls an unknown tool.
	TerminateOnUnknown bool

	// IncludeDetailedErrors includes full error text in tool results sent
	// back to the model. When false, a generic error message is used.
	IncludeDetailedErrors bool
}

// DefaultInvocationConfig returns the default configuration.
func DefaultInvocationConfig() InvocationConfig {
	return InvocationConfig{
		MaxIterations:        40,
		MaxConsecutiveErrors: 3,
	}
}

// invokeFunctions runs the tool-calling loop: extract function_call content
// from the response, invoke matched tools, append results, and re-call the LLM.
//
// It returns the final ChatResponse after all tool calls are resolved (or limits hit).
func invokeFunctions(
	ctx context.Context,
	client ChatClient,
	messages []Message,
	opts *ChatOptions,
	config InvocationConfig,
	fnMiddleware []FunctionMiddleware,
) (*ChatResponse, error) {
	if config.MaxIterations <= 0 {
		config.MaxIterations = 40
	}
	if config.MaxConsecutiveErrors <= 0 {
		config.MaxConsecutiveErrors = 3
	}

	// Build tool lookup
	toolMap := make(map[string]Tool, len(opts.Tools))
	for _, t := range opts.Tools {
		toolMap[t.Name()] = t
	}

	consecutiveErrors := 0

	for iteration := 0; iteration < config.MaxIterations; iteration++ {
		resp, err := client.Response(ctx, messages, opts)
		if err != nil {
			return nil, err
		}

		// Extract function calls from response
		calls := extractFunctionCalls(resp)
		if len(calls) == 0 {
			return resp, nil
		}

		// Process each function call
		var resultMessages []Message
		for _, call := range calls {
			tool, ok := toolMap[call.Name]
			if !ok {
				if config.TerminateOnUnknown {
					return nil, fmt.Errorf("%w: unknown tool %q", ErrToolExecution, call.Name)
				}
				slog.WarnContext(ctx, "unknown tool called", "tool", call.Name)
				resultMessages = append(resultMessages, NewToolMessage(call.CallID, "error: unknown tool"))
				consecutiveErrors++
				continue
			}

			// Check approval
			if tool.Approval() == ApprovalAlways {
				// Return response with approval request — caller handles approval flow
				resp.Messages = append(resp.Messages, Message{
					Role: RoleAssistant,
					Contents: Contents{&ApprovalRequestContent{
						CallID:    call.CallID,
						Name:      call.Name,
						Arguments: call.Arguments,
					}},
				})
				return resp, nil
			}

			// Check declaration-only
			if tool.DeclarationOnly() {
				return resp, nil
			}

			// Invoke the tool (through middleware chain if any)
			result, invokeErr := invokeToolWithMiddleware(ctx, tool, json.RawMessage(call.Arguments), fnMiddleware)
			if invokeErr != nil {
				consecutiveErrors++
				slog.WarnContext(ctx, "tool invocation error",
					"tool", call.Name,
					"error", invokeErr,
					"consecutive_errors", consecutiveErrors,
				)
				if consecutiveErrors >= config.MaxConsecutiveErrors {
					return nil, fmt.Errorf("%w: max consecutive errors reached (%d)", ErrToolExecution, consecutiveErrors)
				}
				errMsg := "error invoking tool"
				if config.IncludeDetailedErrors {
					errMsg = invokeErr.Error()
				}
				resultMessages = append(resultMessages, NewToolMessage(call.CallID, errMsg))
				continue
			}

			consecutiveErrors = 0
			resultMessages = append(resultMessages, NewToolMessage(call.CallID, result))
		}

		// Append assistant message with tool calls and tool results
		for _, m := range resp.Messages {
			messages = append(messages, m)
		}
		messages = append(messages, resultMessages...)
	}

	return nil, fmt.Errorf("%w: max iterations reached (%d)", ErrExecution, config.MaxIterations)
}

// functionCall is an extracted function call from a response.
type functionCall struct {
	CallID    string
	Name      string
	Arguments string
}

// extractFunctionCalls finds all FunctionCallContent in a response's messages.
func extractFunctionCalls(resp *ChatResponse) []functionCall {
	var calls []functionCall
	for _, msg := range resp.Messages {
		for _, c := range msg.Contents {
			if fc, ok := c.(*FunctionCallContent); ok {
				calls = append(calls, functionCall{
					CallID:    fc.CallID,
					Name:      fc.Name,
					Arguments: fc.Arguments,
				})
			}
		}
	}
	return calls
}

// invokeToolWithMiddleware runs the tool through the function middleware chain.
func invokeToolWithMiddleware(ctx context.Context, tool Tool, args json.RawMessage, mws []FunctionMiddleware) (any, error) {
	handler := func(ctx context.Context, t Tool, a json.RawMessage) (any, error) {
		return t.Invoke(ctx, a)
	}
	final := chainFunctionMiddleware(handler, mws...)
	return final(ctx, tool, args)
}
