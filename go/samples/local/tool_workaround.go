// Copyright (c) Microsoft. All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

// toolCallPattern matches JSON arrays that look like tool calls:
// [{"function_name": {...}}] or [{"fn1": {...}}, {"fn2": {...}}]
// Also matches when wrapped in markdown code fences: ```json ... ```
var toolCallPattern = regexp.MustCompile(`(?s)^\s*(?:` + "`" + `{3}(?:json)?\s*)?\[\s*\{.*\}\s*\](?:\s*` + "`" + `{3})?\s*$`)

// ToolCallWorkaroundMiddleware detects text responses that contain tool calls
// in JSON format and converts them to proper FunctionCallContent objects.
//
// This is a workaround for local inference runtimes that don't emit structured
// tool_calls in the OpenAI wire format.
//
// IMPORTANT: This must be used as ChatMiddleware (not AgentMiddleware) so the
// conversion happens BEFORE the agent's tool execution logic runs.
func ToolCallWorkaroundMiddleware(logger *slog.Logger) af.ChatMiddleware {
	return func(next af.ChatHandler) af.ChatHandler {
		return func(ctx context.Context, messages []af.Message, opts *af.ChatOptions) (*af.ChatResponse, error) {
			resp, err := next(ctx, messages, opts)
			if err != nil || resp == nil {
				return resp, err
			}

			// Process each message in the response
			for i := range resp.Messages {
				msg := &resp.Messages[i]
				if msg.Role != af.RoleAssistant {
					continue
				}

				// Check if message has only text content (no existing tool calls)
				if !hasOnlyTextContent(msg) {
					continue
				}

				// Get the text content
				text := extractText(msg)
				if text == "" {
					continue
				}

				// Check if it matches tool call pattern
				if !toolCallPattern.MatchString(text) {
					continue
				}

				logger.Debug("detected potential tool call in text",
					"text", text)

				// Try to parse as tool calls
				toolCalls, err := parseToolCalls(text)
				if err != nil {
					logger.Debug("failed to parse tool calls",
						"error", err)
					continue
				}

				if len(toolCalls) == 0 {
					continue
				}

				logger.Info("converted text to tool calls",
					"count", len(toolCalls))

				// Replace text content with function call content
				msg.Contents = make([]af.Content, len(toolCalls))
				for j, tc := range toolCalls {
					msg.Contents[j] = tc
				}

				// Set finish reason to indicate tool calls
				resp.FinishReason = af.FinishReasonToolCalls
			}

			return resp, nil
		}
	}
}

// hasOnlyTextContent checks if a message contains only text content.
func hasOnlyTextContent(msg *af.Message) bool {
	if len(msg.Contents) == 0 {
		return false
	}
	for _, c := range msg.Contents {
		if _, ok := c.(*af.TextContent); !ok {
			return false
		}
	}
	return true
}

// extractText extracts all text from a message's text contents.
func extractText(msg *af.Message) string {
	var parts []string
	for _, c := range msg.Contents {
		if tc, ok := c.(*af.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// parseToolCalls attempts to parse text as a JSON array of tool calls.
// Expected format: [{"function_name": {"arg": "value"}}, ...]
// Handles markdown code fences: ```json ... ```
func parseToolCalls(text string) ([]*af.FunctionCallContent, error) {
	// Strip markdown code fences if present
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	// Parse as array of objects
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &arr); err != nil {
		return nil, fmt.Errorf("invalid JSON array: %w", err)
	}

	var result []*af.FunctionCallContent
	for i, obj := range arr {
		// Each object should have exactly one key (the function name)
		if len(obj) != 1 {
			return nil, fmt.Errorf("tool call %d: expected 1 key, got %d", i, len(obj))
		}

		// Extract the function name and arguments
		for funcName, argsRaw := range obj {
			// Generate a synthetic call ID
			callID := fmt.Sprintf("call_local_%d", i)

			// Convert arguments to compact JSON string
			argsCompact, err := json.Marshal(json.RawMessage(argsRaw))
			if err != nil {
				return nil, fmt.Errorf("tool call %d (%s): invalid args: %w", i, funcName, err)
			}

			result = append(result, &af.FunctionCallContent{
				CallID:    callID,
				Name:      funcName,
				Arguments: string(argsCompact),
			})
		}
	}

	return result, nil
}
