// Copyright (c) Microsoft. All rights reserved.

package agentframework

// UsageDetails holds token consumption statistics for a model response.
type UsageDetails struct {
	InputTokens  int `json:"inputTokenCount,omitempty"`
	OutputTokens int `json:"outputTokenCount,omitempty"`
	TotalTokens  int `json:"totalTokenCount,omitempty"`
}
