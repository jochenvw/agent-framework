// Copyright (c) Microsoft. All rights reserved.

package agentframework

// ToolChoice controls how the model selects tools.
type ToolChoice string

const (
	ToolChoiceAuto     ToolChoice = "auto"
	ToolChoiceRequired ToolChoice = "required"
	ToolChoiceNone     ToolChoice = "none"
)

// ToolChoiceFunction returns a ToolChoice that forces the model to call
// the named function.
func ToolChoiceFunction(name string) ToolChoice {
	return ToolChoice("function:" + name)
}

// ChatOptions configures a single chat completion request.
// Pointer fields use nil to represent "unset" (use provider default).
type ChatOptions struct {
	ModelID          string
	Temperature      *float64
	TopP             *float64
	MaxTokens        *int
	Stop             []string
	Seed             *int
	FrequencyPenalty *float64
	PresencePenalty  *float64
	Tools            []Tool
	ToolChoice       ToolChoice
	ResponseFormat   any // JSON Schema object or struct type descriptor
	Metadata         map[string]string
	User             string
	Instructions     string
	ConversationID   string
	Store            *bool

	// Extra holds provider-specific options not covered by standard fields.
	Extra map[string]any
}

// MergeChatOptions produces a new ChatOptions by overlaying override values
// onto base. Nil or zero-value fields in override do not overwrite base.
// Tools are merged by name (override replaces same-named tools).
// Metadata is merged (override keys win). Instructions are concatenated.
func MergeChatOptions(base, override *ChatOptions) *ChatOptions {
	if base == nil {
		if override == nil {
			return &ChatOptions{}
		}
		cp := *override
		return &cp
	}
	if override == nil {
		cp := *base
		return &cp
	}

	merged := *base

	if override.ModelID != "" {
		merged.ModelID = override.ModelID
	}
	if override.Temperature != nil {
		merged.Temperature = override.Temperature
	}
	if override.TopP != nil {
		merged.TopP = override.TopP
	}
	if override.MaxTokens != nil {
		merged.MaxTokens = override.MaxTokens
	}
	if len(override.Stop) > 0 {
		merged.Stop = override.Stop
	}
	if override.Seed != nil {
		merged.Seed = override.Seed
	}
	if override.FrequencyPenalty != nil {
		merged.FrequencyPenalty = override.FrequencyPenalty
	}
	if override.PresencePenalty != nil {
		merged.PresencePenalty = override.PresencePenalty
	}
	if override.ToolChoice != "" {
		merged.ToolChoice = override.ToolChoice
	}
	if override.ResponseFormat != nil {
		merged.ResponseFormat = override.ResponseFormat
	}
	if override.User != "" {
		merged.User = override.User
	}
	if override.ConversationID != "" {
		merged.ConversationID = override.ConversationID
	}
	if override.Store != nil {
		merged.Store = override.Store
	}

	// Instructions: concatenate
	if override.Instructions != "" {
		if merged.Instructions != "" {
			merged.Instructions += "\n" + override.Instructions
		} else {
			merged.Instructions = override.Instructions
		}
	}

	// Tools: merge by name
	if len(override.Tools) > 0 {
		byName := make(map[string]Tool, len(merged.Tools)+len(override.Tools))
		for _, t := range merged.Tools {
			byName[t.Name()] = t
		}
		for _, t := range override.Tools {
			byName[t.Name()] = t
		}
		tools := make([]Tool, 0, len(byName))
		// Preserve order: base first, then new from override
		seen := make(map[string]bool, len(byName))
		for _, t := range merged.Tools {
			if _, ok := byName[t.Name()]; ok {
				tools = append(tools, byName[t.Name()])
				seen[t.Name()] = true
			}
		}
		for _, t := range override.Tools {
			if !seen[t.Name()] {
				tools = append(tools, t)
			}
		}
		merged.Tools = tools
	}

	// Metadata: merge maps
	if len(override.Metadata) > 0 {
		if merged.Metadata == nil {
			merged.Metadata = make(map[string]string, len(override.Metadata))
		}
		for k, v := range override.Metadata {
			merged.Metadata[k] = v
		}
	}

	// Extra: merge maps
	if len(override.Extra) > 0 {
		if merged.Extra == nil {
			merged.Extra = make(map[string]any, len(override.Extra))
		}
		for k, v := range override.Extra {
			merged.Extra[k] = v
		}
	}

	return &merged
}
