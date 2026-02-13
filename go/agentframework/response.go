// Copyright (c) Microsoft. All rights reserved.

package agentframework

import "strings"

// ChatResponse is the complete (non-streaming) response from a [ChatClient].
type ChatResponse struct {
	Messages       []Message
	ResponseID     string
	ConversationID string
	ModelID        string
	CreatedAt      string
	FinishReason   FinishReason
	Usage          UsageDetails
	Extra          map[string]any
	Raw            any
}

// Text returns the concatenated text of all messages in this response.
func (r *ChatResponse) Text() string {
	var b strings.Builder
	for i := range r.Messages {
		b.WriteString(r.Messages[i].Text())
	}
	return b.String()
}

// ChatResponseUpdate is a single chunk received during streaming from a [ChatClient].
type ChatResponseUpdate struct {
	Contents       Contents
	Role           Role
	ResponseID     string
	ConversationID string
	ModelID        string
	FinishReason   FinishReason
	Usage          UsageDetails
	Raw            any
}

// Text returns the concatenated text of all [TextContent] items in this update.
func (u *ChatResponseUpdate) Text() string {
	var b strings.Builder
	for _, c := range u.Contents {
		if tc, ok := c.(*TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// AgentResponse is the complete response from an [Agent] run.
type AgentResponse struct {
	Messages   []Message
	ResponseID string
	AgentID    string
	Usage      UsageDetails
	Extra      map[string]any
	Raw        any
}

// Text returns the concatenated text of all messages in this agent response.
func (r *AgentResponse) Text() string {
	var b strings.Builder
	for i := range r.Messages {
		b.WriteString(r.Messages[i].Text())
	}
	return b.String()
}

// UserInputRequests returns all [ApprovalRequestContent] items across messages.
func (r *AgentResponse) UserInputRequests() []Content {
	var reqs []Content
	for _, m := range r.Messages {
		for _, c := range m.Contents {
			if c.Type() == ContentTypeApprovalRequest {
				reqs = append(reqs, c)
			}
		}
	}
	return reqs
}

// AgentResponseUpdate is a single streaming chunk from an [Agent] run.
type AgentResponseUpdate struct {
	Contents   Contents
	Role       Role
	AgentID    string
	ResponseID string
	Usage      UsageDetails
	Raw        any
}

// Text returns the concatenated text of all [TextContent] items in this update.
func (u *AgentResponseUpdate) Text() string {
	var b strings.Builder
	for _, c := range u.Contents {
		if tc, ok := c.(*TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// ChatResponseFromUpdates builds a complete [ChatResponse] by merging
// a sequence of streaming updates.
func ChatResponseFromUpdates(updates []ChatResponseUpdate) *ChatResponse {
	resp := &ChatResponse{}
	var allContents Contents
	for _, u := range updates {
		allContents = append(allContents, u.Contents...)
		if u.ResponseID != "" {
			resp.ResponseID = u.ResponseID
		}
		if u.ConversationID != "" {
			resp.ConversationID = u.ConversationID
		}
		if u.ModelID != "" {
			resp.ModelID = u.ModelID
		}
		if u.FinishReason != "" {
			resp.FinishReason = u.FinishReason
		}
		if u.Usage.TotalTokens > 0 {
			resp.Usage = u.Usage
		}
		if u.Role == "" && resp.Messages == nil {
			// default
		}
	}

	// Merge text content deltas into a single TextContent.
	merged := mergeContentDeltas(allContents)
	if len(merged) > 0 {
		role := RoleAssistant
		if len(updates) > 0 && updates[0].Role != "" {
			role = updates[0].Role
		}
		resp.Messages = []Message{{Role: role, Contents: merged}}
	}
	return resp
}

// mergeContentDeltas consolidates sequential TextContent runs into single
// items, and passes non-text content through as-is.
func mergeContentDeltas(cs Contents) Contents {
	if len(cs) == 0 {
		return nil
	}
	var merged Contents
	var textBuf strings.Builder
	flush := func() {
		if textBuf.Len() > 0 {
			merged = append(merged, &TextContent{Text: textBuf.String()})
			textBuf.Reset()
		}
	}
	for _, c := range cs {
		if tc, ok := c.(*TextContent); ok {
			textBuf.WriteString(tc.Text)
		} else {
			flush()
			merged = append(merged, c)
		}
	}
	flush()
	return merged
}

// AgentResponseFromUpdates builds a complete [AgentResponse] by merging
// a sequence of streaming updates.
func AgentResponseFromUpdates(updates []AgentResponseUpdate) *AgentResponse {
	resp := &AgentResponse{}
	var allContents Contents
	for _, u := range updates {
		allContents = append(allContents, u.Contents...)
		if u.AgentID != "" {
			resp.AgentID = u.AgentID
		}
		if u.ResponseID != "" {
			resp.ResponseID = u.ResponseID
		}
		if u.Usage.TotalTokens > 0 {
			resp.Usage = u.Usage
		}
	}

	merged := mergeContentDeltas(allContents)
	if len(merged) > 0 {
		role := RoleAssistant
		if len(updates) > 0 && updates[0].Role != "" {
			role = updates[0].Role
		}
		resp.Messages = []Message{{Role: role, Contents: merged}}
	}
	return resp
}
