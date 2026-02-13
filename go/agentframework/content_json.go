// Copyright (c) Microsoft. All rights reserved.

package agentframework

import (
	"encoding/json"
	"fmt"
)

// contentEnvelope is the JSON wire format using a $type discriminator,
// aligned with schemas/durable-agent-entity-state.json.
type contentEnvelope struct {
	Type string          `json:"$type"`
	Data json.RawMessage `json:"-"`
}

// MarshalContentJSON marshals a single Content value into its JSON envelope.
func MarshalContentJSON(c Content) ([]byte, error) {
	switch v := c.(type) {
	case *TextContent:
		return json.Marshal(struct {
			Type string `json:"$type"`
			Text string `json:"text"`
		}{string(ContentTypeText), v.Text})

	case *TextReasoningContent:
		return json.Marshal(struct {
			Type string `json:"$type"`
			Text string `json:"text,omitempty"`
		}{string(ContentTypeTextReasoning), v.Text})

	case *DataContent:
		return json.Marshal(struct {
			Type      string `json:"$type"`
			URI       string `json:"uri"`
			MediaType string `json:"mediaType,omitempty"`
		}{string(ContentTypeData), v.URI, v.MediaType})

	case *URIContent:
		return json.Marshal(struct {
			Type      string `json:"$type"`
			URI       string `json:"uri"`
			MediaType string `json:"mediaType"`
		}{string(ContentTypeURI), v.URI, v.MediaType})

	case *ErrorContent:
		return json.Marshal(struct {
			Type      string `json:"$type"`
			Message   string `json:"message,omitempty"`
			ErrorCode string `json:"errorCode,omitempty"`
			Details   any    `json:"details,omitempty"`
		}{string(ContentTypeError), v.Message, v.ErrorCode, v.Details})

	case *FunctionCallContent:
		return json.Marshal(struct {
			Type      string          `json:"$type"`
			CallID    string          `json:"callId"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments,omitempty"`
		}{string(ContentTypeFunctionCall), v.CallID, v.Name, json.RawMessage(v.Arguments)})

	case *FunctionResultContent:
		return json.Marshal(struct {
			Type   string `json:"$type"`
			CallID string `json:"callId"`
			Result any    `json:"result,omitempty"`
		}{string(ContentTypeFunctionResult), v.CallID, v.Result})

	case *UsageContent:
		return json.Marshal(struct {
			Type  string       `json:"$type"`
			Usage UsageDetails `json:"usage"`
		}{string(ContentTypeUsage), v.Usage})

	case *HostedFileContent:
		return json.Marshal(struct {
			Type   string `json:"$type"`
			FileID string `json:"fileId"`
		}{string(ContentTypeHostedFile), v.FileID})

	case *HostedVectorStoreContent:
		return json.Marshal(struct {
			Type          string `json:"$type"`
			VectorStoreID string `json:"vectorStoreId"`
		}{string(ContentTypeHostedVectorStore), v.VectorStoreID})

	case *CodeInterpreterCallContent:
		return json.Marshal(struct {
			Type   string `json:"$type"`
			CallID string `json:"callId,omitempty"`
			Code   string `json:"code"`
		}{string(ContentTypeCodeInterpreterCall), v.CallID, v.Code})

	case *CodeInterpreterResultContent:
		return json.Marshal(struct {
			Type   string `json:"$type"`
			CallID string `json:"callId,omitempty"`
			Output string `json:"output"`
		}{string(ContentTypeCodeInterpreterResult), v.CallID, v.Output})

	case *ImageGenCallContent:
		return json.Marshal(struct {
			Type   string `json:"$type"`
			CallID string `json:"callId,omitempty"`
			Prompt string `json:"prompt"`
		}{string(ContentTypeImageGenCall), v.CallID, v.Prompt})

	case *ImageGenResultContent:
		return json.Marshal(struct {
			Type   string `json:"$type"`
			CallID string `json:"callId,omitempty"`
			URI    string `json:"uri"`
		}{string(ContentTypeImageGenResult), v.CallID, v.URI})

	case *MCPServerCallContent:
		return json.Marshal(struct {
			Type      string          `json:"$type"`
			CallID    string          `json:"callId,omitempty"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments,omitempty"`
		}{string(ContentTypeMCPServerCall), v.CallID, v.Name, json.RawMessage(v.Arguments)})

	case *MCPServerResultContent:
		return json.Marshal(struct {
			Type   string `json:"$type"`
			CallID string `json:"callId,omitempty"`
			Result any    `json:"result,omitempty"`
		}{string(ContentTypeMCPServerResult), v.CallID, v.Result})

	case *ApprovalRequestContent:
		return json.Marshal(struct {
			Type      string `json:"$type"`
			CallID    string `json:"callId"`
			Name      string `json:"name"`
			Arguments string `json:"arguments,omitempty"`
		}{string(ContentTypeApprovalRequest), v.CallID, v.Name, v.Arguments})

	case *ApprovalResponseContent:
		return json.Marshal(struct {
			Type     string `json:"$type"`
			CallID   string `json:"callId"`
			Approved bool   `json:"approved"`
			Reason   string `json:"reason,omitempty"`
		}{string(ContentTypeApprovalResponse), v.CallID, v.Approved, v.Reason})

	default:
		return nil, fmt.Errorf("unknown content type: %T", c)
	}
}

// UnmarshalContentJSON unmarshals a single Content value from its JSON envelope.
func UnmarshalContentJSON(data []byte) (Content, error) {
	var env struct {
		Type string `json:"$type"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal content envelope: %w", err)
	}

	switch ContentType(env.Type) {
	case ContentTypeText:
		var v struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &TextContent{Text: v.Text}, nil

	case ContentTypeTextReasoning:
		var v struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &TextReasoningContent{Text: v.Text}, nil

	case ContentTypeData:
		var v struct {
			URI       string `json:"uri"`
			MediaType string `json:"mediaType"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &DataContent{URI: v.URI, MediaType: v.MediaType}, nil

	case ContentTypeURI:
		var v struct {
			URI       string `json:"uri"`
			MediaType string `json:"mediaType"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &URIContent{URI: v.URI, MediaType: v.MediaType}, nil

	case ContentTypeError:
		var v struct {
			Message   string `json:"message"`
			ErrorCode string `json:"errorCode"`
			Details   any    `json:"details"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &ErrorContent{Message: v.Message, ErrorCode: v.ErrorCode, Details: v.Details}, nil

	case ContentTypeFunctionCall:
		var v struct {
			CallID    string          `json:"callId"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &FunctionCallContent{CallID: v.CallID, Name: v.Name, Arguments: string(v.Arguments)}, nil

	case ContentTypeFunctionResult:
		var v struct {
			CallID string `json:"callId"`
			Result any    `json:"result"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &FunctionResultContent{CallID: v.CallID, Result: v.Result}, nil

	case ContentTypeUsage:
		var v struct {
			Usage UsageDetails `json:"usage"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &UsageContent{Usage: v.Usage}, nil

	case ContentTypeHostedFile:
		var v struct {
			FileID string `json:"fileId"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &HostedFileContent{FileID: v.FileID}, nil

	case ContentTypeHostedVectorStore:
		var v struct {
			VectorStoreID string `json:"vectorStoreId"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &HostedVectorStoreContent{VectorStoreID: v.VectorStoreID}, nil

	case ContentTypeCodeInterpreterCall:
		var v struct {
			CallID string `json:"callId"`
			Code   string `json:"code"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &CodeInterpreterCallContent{CallID: v.CallID, Code: v.Code}, nil

	case ContentTypeCodeInterpreterResult:
		var v struct {
			CallID string `json:"callId"`
			Output string `json:"output"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &CodeInterpreterResultContent{CallID: v.CallID, Output: v.Output}, nil

	case ContentTypeImageGenCall:
		var v struct {
			CallID string `json:"callId"`
			Prompt string `json:"prompt"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &ImageGenCallContent{CallID: v.CallID, Prompt: v.Prompt}, nil

	case ContentTypeImageGenResult:
		var v struct {
			CallID string `json:"callId"`
			URI    string `json:"uri"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &ImageGenResultContent{CallID: v.CallID, URI: v.URI}, nil

	case ContentTypeMCPServerCall:
		var v struct {
			CallID    string          `json:"callId"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &MCPServerCallContent{CallID: v.CallID, Name: v.Name, Arguments: string(v.Arguments)}, nil

	case ContentTypeMCPServerResult:
		var v struct {
			CallID string `json:"callId"`
			Result any    `json:"result"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &MCPServerResultContent{CallID: v.CallID, Result: v.Result}, nil

	case ContentTypeApprovalRequest:
		var v struct {
			CallID    string `json:"callId"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &ApprovalRequestContent{CallID: v.CallID, Name: v.Name, Arguments: v.Arguments}, nil

	case ContentTypeApprovalResponse:
		var v struct {
			CallID   string `json:"callId"`
			Approved bool   `json:"approved"`
			Reason   string `json:"reason"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &ApprovalResponseContent{CallID: v.CallID, Approved: v.Approved, Reason: v.Reason}, nil

	default:
		return nil, fmt.Errorf("unknown content $type: %q", env.Type)
	}
}

// Contents is a typed slice enabling JSON marshal/unmarshal of polymorphic Content arrays.
type Contents []Content

// MarshalJSON serializes each Content item using its $type discriminator.
func (cs Contents) MarshalJSON() ([]byte, error) {
	items := make([]json.RawMessage, len(cs))
	for i, c := range cs {
		b, err := MarshalContentJSON(c)
		if err != nil {
			return nil, fmt.Errorf("marshal content[%d]: %w", i, err)
		}
		items[i] = b
	}
	return json.Marshal(items)
}

// UnmarshalJSON deserializes a JSON array of Content items using the $type discriminator.
func (cs *Contents) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	result := make([]Content, len(raw))
	for i, r := range raw {
		c, err := UnmarshalContentJSON(r)
		if err != nil {
			return fmt.Errorf("unmarshal content[%d]: %w", i, err)
		}
		result[i] = c
	}
	*cs = result
	return nil
}
