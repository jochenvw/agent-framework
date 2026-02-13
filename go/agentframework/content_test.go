// Copyright (c) Microsoft. All rights reserved.

package agentframework_test

import (
	"encoding/json"
	"testing"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

// --- Content JSON round-trip tests ---

func TestContentJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		content af.Content
		check   func(t *testing.T, got af.Content)
	}{
		{
			name:    "TextContent",
			content: &af.TextContent{Text: "hello"},
			check: func(t *testing.T, got af.Content) {
				tc, ok := got.(*af.TextContent)
				if !ok {
					t.Fatalf("expected *TextContent, got %T", got)
				}
				if tc.Text != "hello" {
					t.Errorf("text = %q, want %q", tc.Text, "hello")
				}
			},
		},
		{
			name:    "TextReasoningContent",
			content: &af.TextReasoningContent{Text: "thinking..."},
			check: func(t *testing.T, got af.Content) {
				tc, ok := got.(*af.TextReasoningContent)
				if !ok {
					t.Fatalf("expected *TextReasoningContent, got %T", got)
				}
				if tc.Text != "thinking..." {
					t.Errorf("text = %q, want %q", tc.Text, "thinking...")
				}
			},
		},
		{
			name:    "DataContent",
			content: &af.DataContent{URI: "data:image/png;base64,abc", MediaType: "image/png"},
			check: func(t *testing.T, got af.Content) {
				dc, ok := got.(*af.DataContent)
				if !ok {
					t.Fatalf("expected *DataContent, got %T", got)
				}
				if dc.URI != "data:image/png;base64,abc" {
					t.Errorf("URI = %q, want data:image/png;base64,abc", dc.URI)
				}
				if dc.MediaType != "image/png" {
					t.Errorf("MediaType = %q, want image/png", dc.MediaType)
				}
			},
		},
		{
			name:    "URIContent",
			content: &af.URIContent{URI: "https://example.com/img.png", MediaType: "image/png"},
			check: func(t *testing.T, got af.Content) {
				uc, ok := got.(*af.URIContent)
				if !ok {
					t.Fatalf("expected *URIContent, got %T", got)
				}
				if uc.URI != "https://example.com/img.png" {
					t.Errorf("URI = %q", uc.URI)
				}
			},
		},
		{
			name:    "ErrorContent",
			content: &af.ErrorContent{Message: "bad request", ErrorCode: "400"},
			check: func(t *testing.T, got af.Content) {
				ec, ok := got.(*af.ErrorContent)
				if !ok {
					t.Fatalf("expected *ErrorContent, got %T", got)
				}
				if ec.Message != "bad request" {
					t.Errorf("Message = %q", ec.Message)
				}
				if ec.ErrorCode != "400" {
					t.Errorf("ErrorCode = %q", ec.ErrorCode)
				}
			},
		},
		{
			name:    "FunctionCallContent",
			content: &af.FunctionCallContent{CallID: "c1", Name: "get_weather", Arguments: `{"city":"Seattle"}`},
			check: func(t *testing.T, got af.Content) {
				fc, ok := got.(*af.FunctionCallContent)
				if !ok {
					t.Fatalf("expected *FunctionCallContent, got %T", got)
				}
				if fc.CallID != "c1" || fc.Name != "get_weather" {
					t.Errorf("CallID=%q Name=%q", fc.CallID, fc.Name)
				}
			},
		},
		{
			name:    "FunctionResultContent",
			content: &af.FunctionResultContent{CallID: "c1", Result: "72°F"},
			check: func(t *testing.T, got af.Content) {
				fr, ok := got.(*af.FunctionResultContent)
				if !ok {
					t.Fatalf("expected *FunctionResultContent, got %T", got)
				}
				if fr.CallID != "c1" {
					t.Errorf("CallID = %q", fr.CallID)
				}
			},
		},
		{
			name:    "ApprovalRequestContent",
			content: &af.ApprovalRequestContent{CallID: "c2", Name: "send_email", Arguments: `{"to":"bob"}`},
			check: func(t *testing.T, got af.Content) {
				ar, ok := got.(*af.ApprovalRequestContent)
				if !ok {
					t.Fatalf("expected *ApprovalRequestContent, got %T", got)
				}
				if ar.Name != "send_email" {
					t.Errorf("Name = %q", ar.Name)
				}
			},
		},
		{
			name:    "ApprovalResponseContent",
			content: &af.ApprovalResponseContent{CallID: "c2", Approved: true, Reason: "ok"},
			check: func(t *testing.T, got af.Content) {
				ar, ok := got.(*af.ApprovalResponseContent)
				if !ok {
					t.Fatalf("expected *ApprovalResponseContent, got %T", got)
				}
				if !ar.Approved || ar.Reason != "ok" {
					t.Errorf("Approved=%v Reason=%q", ar.Approved, ar.Reason)
				}
			},
		},
		{
			name:    "HostedFileContent",
			content: &af.HostedFileContent{FileID: "file-123"},
			check: func(t *testing.T, got af.Content) {
				hf, ok := got.(*af.HostedFileContent)
				if !ok {
					t.Fatalf("expected *HostedFileContent, got %T", got)
				}
				if hf.FileID != "file-123" {
					t.Errorf("FileID = %q", hf.FileID)
				}
			},
		},
		{
			name:    "UsageContent",
			content: &af.UsageContent{Usage: af.UsageDetails{InputTokens: 10, OutputTokens: 20, TotalTokens: 30}},
			check: func(t *testing.T, got af.Content) {
				uc, ok := got.(*af.UsageContent)
				if !ok {
					t.Fatalf("expected *UsageContent, got %T", got)
				}
				if uc.Usage.TotalTokens != 30 {
					t.Errorf("TotalTokens = %d", uc.Usage.TotalTokens)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := af.MarshalContentJSON(tc.content)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			got, err := af.UnmarshalContentJSON(data)
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if got.Type() != tc.content.Type() {
				t.Errorf("type = %q, want %q", got.Type(), tc.content.Type())
			}

			tc.check(t, got)
		})
	}
}

func TestContentJSONHasTypeDiscriminator(t *testing.T) {
	data, err := af.MarshalContentJSON(&af.TextContent{Text: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	typ, ok := envelope["$type"]
	if !ok {
		t.Fatal("missing $type field in JSON")
	}
	if typ != "text" {
		t.Errorf("$type = %q, want %q", typ, "text")
	}
}

func TestContentsSliceMarshalUnmarshal(t *testing.T) {
	original := af.Contents{
		&af.TextContent{Text: "hello"},
		&af.FunctionCallContent{CallID: "c1", Name: "fn", Arguments: "{}"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var restored af.Contents
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(restored) != 2 {
		t.Fatalf("len = %d, want 2", len(restored))
	}
	if restored[0].Type() != af.ContentTypeText {
		t.Errorf("[0] type = %q", restored[0].Type())
	}
	if restored[1].Type() != af.ContentTypeFunctionCall {
		t.Errorf("[1] type = %q", restored[1].Type())
	}
}

func TestUnmarshalContentJSON_UnknownType(t *testing.T) {
	data := []byte(`{"$type":"unknown_type","foo":"bar"}`)
	_, err := af.UnmarshalContentJSON(data)
	if err == nil {
		t.Fatal("expected error for unknown $type")
	}
}
