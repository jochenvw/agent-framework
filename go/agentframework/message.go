// Copyright (c) Microsoft. All rights reserved.

package agentframework

import "strings"

// Role identifies the author of a [Message].
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// FinishReason indicates why the model stopped generating.
type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonContentFilter FinishReason = "content_filter"
)

// Message represents a single chat message exchanged with an agent or model.
type Message struct {
	Role       Role     `json:"role"`
	Contents   Contents `json:"contents,omitempty"`
	AuthorName string   `json:"authorName,omitempty"`
	MessageID  string   `json:"messageId,omitempty"`

	// Extra holds provider-specific metadata not covered by standard fields.
	Extra map[string]any `json:"-"`

	// Raw holds the original provider-specific representation, if any.
	Raw any `json:"-"`
}

// Text returns the concatenated text of all [TextContent] items in this message.
func (m *Message) Text() string {
	var b strings.Builder
	for _, c := range m.Contents {
		if tc, ok := c.(*TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// NewUserMessage creates a user-role [Message] from a text string.
func NewUserMessage(text string) Message {
	return Message{
		Role:     RoleUser,
		Contents: Contents{&TextContent{Text: text}},
	}
}

// NewAssistantMessage creates an assistant-role [Message] from a text string.
func NewAssistantMessage(text string) Message {
	return Message{
		Role:     RoleAssistant,
		Contents: Contents{&TextContent{Text: text}},
	}
}

// NewSystemMessage creates a system-role [Message] from a text string.
func NewSystemMessage(text string) Message {
	return Message{
		Role:     RoleSystem,
		Contents: Contents{&TextContent{Text: text}},
	}
}

// NewToolMessage creates a tool-role [Message] with a function result.
func NewToolMessage(callID string, result any) Message {
	return Message{
		Role: RoleTool,
		Contents: Contents{&FunctionResultContent{
			CallID: callID,
			Result: result,
		}},
	}
}

// NormalizeMessages converts flexible input forms into a []Message slice.
// Accepted inputs: string (becomes user message), Message, []Message.
// Nil or empty input returns nil.
func NormalizeMessages(inputs ...any) []Message {
	var msgs []Message
	for _, input := range inputs {
		switch v := input.(type) {
		case string:
			msgs = append(msgs, NewUserMessage(v))
		case Message:
			msgs = append(msgs, v)
		case []Message:
			msgs = append(msgs, v...)
		}
	}
	return msgs
}

// PrependInstructions inserts a system message at the beginning of the message
// list if instructions are non-empty and no system message already exists.
func PrependInstructions(messages []Message, instructions string) []Message {
	if instructions == "" {
		return messages
	}
	for _, m := range messages {
		if m.Role == RoleSystem {
			return messages
		}
	}
	return append([]Message{NewSystemMessage(instructions)}, messages...)
}
