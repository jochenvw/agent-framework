// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"encoding/json"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

// chatCompletionResponse is the OpenAI Chat Completions API response.
type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   *usage   `json:"usage,omitempty"`
}

type choice struct {
	Index        int         `json:"index"`
	Message      respMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type respMessage struct {
	Role      string     `json:"role"`
	Content   *string    `json:"content"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// chatCompletionChunk is a single SSE chunk in streaming mode.
type chatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []chunkChoice `json:"choices"`
	Usage   *usage        `json:"usage,omitempty"`
}

type chunkChoice struct {
	Index        int        `json:"index"`
	Delta        chunkDelta `json:"delta"`
	FinishReason *string    `json:"finish_reason"`
}

type chunkDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   *string    `json:"content,omitempty"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

// parseChatResponse converts the OpenAI response into framework types.
func parseChatResponse(raw *chatCompletionResponse) *af.ChatResponse {
	resp := &af.ChatResponse{
		ResponseID: raw.ID,
		ModelID:    raw.Model,
	}

	if raw.Usage != nil {
		resp.Usage = af.UsageDetails{
			InputTokens:  raw.Usage.PromptTokens,
			OutputTokens: raw.Usage.CompletionTokens,
			TotalTokens:  raw.Usage.TotalTokens,
		}
	}

	if len(raw.Choices) > 0 {
		c := raw.Choices[0]
		resp.FinishReason = mapFinishReason(c.FinishReason)

		msg := af.Message{
			Role: af.Role(c.Message.Role),
		}

		if c.Message.Content != nil && *c.Message.Content != "" {
			msg.Contents = append(msg.Contents, &af.TextContent{Text: *c.Message.Content})
		}

		for _, tc := range c.Message.ToolCalls {
			msg.Contents = append(msg.Contents, &af.FunctionCallContent{
				CallID:    tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}

		resp.Messages = []af.Message{msg}
	}

	return resp
}

// parseChunk converts a streaming chunk into a ChatResponseUpdate.
func parseChunk(chunk *chatCompletionChunk) *af.ChatResponseUpdate {
	update := &af.ChatResponseUpdate{
		ResponseID: chunk.ID,
		ModelID:    chunk.Model,
	}

	if chunk.Usage != nil {
		update.Usage = af.UsageDetails{
			InputTokens:  chunk.Usage.PromptTokens,
			OutputTokens: chunk.Usage.CompletionTokens,
			TotalTokens:  chunk.Usage.TotalTokens,
		}
	}

	if len(chunk.Choices) > 0 {
		c := chunk.Choices[0]

		if c.Delta.Role != "" {
			update.Role = af.Role(c.Delta.Role)
		}

		if c.FinishReason != nil {
			update.FinishReason = mapFinishReason(*c.FinishReason)
		}

		if c.Delta.Content != nil && *c.Delta.Content != "" {
			update.Contents = append(update.Contents, &af.TextContent{Text: *c.Delta.Content})
		}

		for _, tc := range c.Delta.ToolCalls {
			update.Contents = append(update.Contents, &af.FunctionCallContent{
				CallID:    tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}

	return update
}

// unmarshalChatResponse parses the JSON response body.
func unmarshalChatResponse(data []byte) (*chatCompletionResponse, error) {
	var resp chatCompletionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func mapFinishReason(s string) af.FinishReason {
	switch s {
	case "stop":
		return af.FinishReasonStop
	case "length":
		return af.FinishReasonLength
	case "tool_calls":
		return af.FinishReasonToolCalls
	case "content_filter":
		return af.FinishReasonContentFilter
	default:
		return af.FinishReason(s)
	}
}
