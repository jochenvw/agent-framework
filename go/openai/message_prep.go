// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"encoding/json"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

// chatRequest is the OpenAI Chat Completions API request body.
type chatRequest struct {
	Model            string          `json:"model"`
	Messages         []chatMessage   `json:"messages"`
	Temperature      *float64        `json:"temperature,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	MaxTokens        *int            `json:"max_completion_tokens,omitempty"`
	Stop             []string        `json:"stop,omitempty"`
	Seed             *int            `json:"seed,omitempty"`
	FrequencyPenalty *float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64        `json:"presence_penalty,omitempty"`
	Tools            []toolSpec      `json:"tools,omitempty"`
	ToolChoice       any             `json:"tool_choice,omitempty"`
	User             string          `json:"user,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
	StreamOptions    *streamOptions  `json:"stream_options,omitempty"`
	ResponseFormat   any             `json:"response_format,omitempty"`
	Store            *bool           `json:"store,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatMessage struct {
	Role       string          `json:"role"`
	Content    any             `json:"content,omitempty"` // string or []contentPart
	Name       string          `json:"name,omitempty"`
	ToolCalls  []toolCall      `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolSpec struct {
	Type     string       `json:"type"`
	Function functionSpec `json:"function"`
}

type functionSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// buildRequest converts framework types into an OpenAI API request.
func buildRequest(messages []af.Message, opts *af.ChatOptions, defaultModel string) *chatRequest {
	req := &chatRequest{
		Model: defaultModel,
	}
	if opts != nil {
		if opts.ModelID != "" {
			req.Model = opts.ModelID
		}
		req.Temperature = opts.Temperature
		req.TopP = opts.TopP
		req.MaxTokens = opts.MaxTokens
		req.Stop = opts.Stop
		req.Seed = opts.Seed
		req.FrequencyPenalty = opts.FrequencyPenalty
		req.PresencePenalty = opts.PresencePenalty
		req.User = opts.User
		req.Store = opts.Store
		req.Metadata = opts.Metadata
		req.ResponseFormat = opts.ResponseFormat

		// Convert tools
		for _, t := range opts.Tools {
			req.Tools = append(req.Tools, toolSpec{
				Type: "function",
				Function: functionSpec{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  t.Parameters(),
				},
			})
		}

		// Convert tool choice
		req.ToolChoice = convertToolChoice(opts.ToolChoice)
	}

	req.Messages = convertMessages(messages)
	return req
}

// convertMessages translates framework Messages into OpenAI chat messages.
func convertMessages(messages []af.Message) []chatMessage {
	result := make([]chatMessage, 0, len(messages))

	for _, msg := range messages {
		cm := chatMessage{
			Role: string(msg.Role),
			Name: msg.AuthorName,
		}

		switch msg.Role {
		case af.RoleTool:
			// Tool messages carry a single function result
			for _, c := range msg.Contents {
				if fr, ok := c.(*af.FunctionResultContent); ok {
					cm.ToolCallID = fr.CallID
					resultStr, _ := marshalResult(fr.Result)
					cm.Content = resultStr
				}
			}

		case af.RoleAssistant:
			// Assistant messages may have text + tool calls
			var textParts []string
			for _, c := range msg.Contents {
				switch v := c.(type) {
				case *af.TextContent:
					textParts = append(textParts, v.Text)
				case *af.FunctionCallContent:
					cm.ToolCalls = append(cm.ToolCalls, toolCall{
						ID:   v.CallID,
						Type: "function",
						Function: functionCall{
							Name:      v.Name,
							Arguments: v.Arguments,
						},
					})
				}
			}
			if len(textParts) > 0 {
				cm.Content = concatStrings(textParts)
			}

		default:
			// User/system messages: simple text or multi-part
			parts := convertContentParts(msg.Contents)
			if len(parts) == 1 && parts[0].Type == "text" {
				cm.Content = parts[0].Text
			} else if len(parts) > 0 {
				cm.Content = parts
			}
		}

		result = append(result, cm)
	}

	return result
}

// convertContentParts converts framework Content items into OpenAI content parts.
func convertContentParts(contents af.Contents) []contentPart {
	var parts []contentPart
	for _, c := range contents {
		switch v := c.(type) {
		case *af.TextContent:
			parts = append(parts, contentPart{Type: "text", Text: v.Text})
		case *af.DataContent:
			parts = append(parts, contentPart{
				Type:     "image_url",
				ImageURL: &imageURL{URL: v.URI},
			})
		case *af.URIContent:
			parts = append(parts, contentPart{
				Type:     "image_url",
				ImageURL: &imageURL{URL: v.URI},
			})
		}
	}
	return parts
}

func convertToolChoice(tc af.ToolChoice) any {
	if tc == "" {
		return nil
	}
	switch tc {
	case af.ToolChoiceAuto:
		return "auto"
	case af.ToolChoiceRequired:
		return "required"
	case af.ToolChoiceNone:
		return "none"
	default:
		// Check for function: prefix
		s := string(tc)
		if len(s) > 9 && s[:9] == "function:" {
			return map[string]any{
				"type": "function",
				"function": map[string]string{
					"name": s[9:],
				},
			}
		}
		return string(tc)
	}
}

func marshalResult(v any) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}
	b, err := json.Marshal(v)
	return string(b), err
}

func concatStrings(parts []string) string {
	if len(parts) == 1 {
		return parts[0]
	}
	result := ""
	for _, p := range parts {
		result += p
	}
	return result
}
