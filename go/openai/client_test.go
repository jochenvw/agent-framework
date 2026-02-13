// Copyright (c) Microsoft. All rights reserved.

package openai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	af "github.com/microsoft/agent-framework/go/agentframework"
	"github.com/microsoft/agent-framework/go/openai"
)

// mockTransportFunc is a RoundTripper that delegates to a function.
type mockTransportFunc func(*http.Request) (*http.Response, error)

func (f mockTransportFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockHTTPClient(fn func(*http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{Transport: mockTransportFunc(fn)}
}

func jsonResponse(status int, body any) *http.Response {
	b, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(b)),
	}
}

func TestClient_Response_Basic(t *testing.T) {
	content := "Hello, I'm an AI assistant!"
	apiResp := map[string]any{
		"id":      "chatcmpl-123",
		"object":  "chat.completion",
		"model":   "gpt-4o",
		"choices": []map[string]any{{
			"index":         0,
			"finish_reason": "stop",
			"message": map[string]any{
				"role":    "assistant",
				"content": content,
			},
		}},
		"usage": map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 8,
			"total_tokens":      18,
		},
	}

	httpClient := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		// Verify request
		if req.Method != "POST" {
			t.Errorf("method = %q", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/chat/completions") {
			t.Errorf("path = %q", req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("auth = %q", req.Header.Get("Authorization"))
		}

		// Verify request body has correct structure
		body, _ := io.ReadAll(req.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["model"] != "gpt-4o" {
			t.Errorf("request model = %v", reqBody["model"])
		}

		return jsonResponse(200, apiResp), nil
	})

	client := openai.New("test-key",
		openai.WithModel("gpt-4o"),
		openai.WithHTTPClient(httpClient),
	)

	resp, err := client.Response(context.Background(),
		[]af.Message{af.NewUserMessage("hi")},
		nil,
	)
	if err != nil {
		t.Fatalf("Response: %v", err)
	}

	if resp.ResponseID != "chatcmpl-123" {
		t.Errorf("ResponseID = %q", resp.ResponseID)
	}
	if resp.ModelID != "gpt-4o" {
		t.Errorf("ModelID = %q", resp.ModelID)
	}
	if resp.FinishReason != af.FinishReasonStop {
		t.Errorf("FinishReason = %q", resp.FinishReason)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 8 {
		t.Errorf("OutputTokens = %d", resp.Usage.OutputTokens)
	}
	if resp.Text() != content {
		t.Errorf("Text = %q", resp.Text())
	}
}

func TestClient_Response_ToolCalls(t *testing.T) {
	apiResp := map[string]any{
		"id":    "chatcmpl-456",
		"model": "gpt-4o",
		"choices": []map[string]any{{
			"index":         0,
			"finish_reason": "tool_calls",
			"message": map[string]any{
				"role": "assistant",
				"tool_calls": []map[string]any{{
					"id":   "call_abc",
					"type": "function",
					"function": map[string]any{
						"name":      "get_weather",
						"arguments": `{"city":"Seattle"}`,
					},
				}},
			},
		}},
	}

	httpClient := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, apiResp), nil
	})

	client := openai.New("test-key",
		openai.WithModel("gpt-4o"),
		openai.WithHTTPClient(httpClient),
	)

	resp, err := client.Response(context.Background(),
		[]af.Message{af.NewUserMessage("weather?")},
		nil,
	)
	if err != nil {
		t.Fatalf("Response: %v", err)
	}

	if resp.FinishReason != af.FinishReasonToolCalls {
		t.Errorf("FinishReason = %q", resp.FinishReason)
	}
	if len(resp.Messages) != 1 {
		t.Fatalf("messages = %d", len(resp.Messages))
	}

	msg := resp.Messages[0]
	if len(msg.Contents) != 1 {
		t.Fatalf("contents = %d", len(msg.Contents))
	}

	fc, ok := msg.Contents[0].(*af.FunctionCallContent)
	if !ok {
		t.Fatalf("content type = %T", msg.Contents[0])
	}
	if fc.CallID != "call_abc" {
		t.Errorf("CallID = %q", fc.CallID)
	}
	if fc.Name != "get_weather" {
		t.Errorf("Name = %q", fc.Name)
	}
}

func TestClient_Response_ErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     map[string]any
		checkErr func(t *testing.T, err error)
	}{
		{
			name:   "401 Unauthorized",
			status: 401,
			body: map[string]any{
				"error": map[string]any{
					"message": "Invalid API key",
					"type":    "authentication_error",
				},
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected error")
				}
				var svcErr *af.ServiceError
				if !errors.As(err, &svcErr) {
					t.Fatal("expected ServiceError")
				}
				if svcErr.StatusCode != 401 {
					t.Errorf("StatusCode = %d", svcErr.StatusCode)
				}
			},
		},
		{
			name:   "Content Filter",
			status: 400,
			body: map[string]any{
				"error": map[string]any{
					"message": "content filtered",
					"code":    "content_filter",
				},
			},
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected error")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpClient := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
				return jsonResponse(tc.status, tc.body), nil
			})

			client := openai.New("bad-key",
				openai.WithModel("gpt-4o"),
				openai.WithHTTPClient(httpClient),
			)

			_, err := client.Response(context.Background(),
				[]af.Message{af.NewUserMessage("hi")},
				nil,
			)
			tc.checkErr(t, err)
		})
	}
}

func TestClient_StreamResponse(t *testing.T) {
	sseData := strings.Join([]string{
		`data: {"id":"chatcmpl-1","model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
		``,
		`data: {"id":"chatcmpl-1","model":"gpt-4o","choices":[{"index":0,"delta":{"content":", world!"},"finish_reason":null}]}`,
		``,
		`data: {"id":"chatcmpl-1","model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	httpClient := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		// Verify stream flag
		body, _ := io.ReadAll(req.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["stream"] != true {
			t.Errorf("stream = %v", reqBody["stream"])
		}

		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(sseData)),
		}, nil
	})

	client := openai.New("test-key",
		openai.WithModel("gpt-4o"),
		openai.WithHTTPClient(httpClient),
	)

	stream, err := client.StreamResponse(context.Background(),
		[]af.Message{af.NewUserMessage("hi")},
		nil,
	)
	if err != nil {
		t.Fatalf("StreamResponse: %v", err)
	}
	defer stream.Close()

	updates, err := stream.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if len(updates) < 2 {
		t.Fatalf("updates = %d, want >= 2", len(updates))
	}

	// First update should have role + content
	if updates[0].Role != af.RoleAssistant {
		t.Errorf("[0].Role = %q", updates[0].Role)
	}
	if updates[0].Text() != "Hello" {
		t.Errorf("[0].Text = %q", updates[0].Text())
	}

	// Second update should have content continuation
	if updates[1].Text() != ", world!" {
		t.Errorf("[1].Text = %q", updates[1].Text())
	}

	// Merge updates into a complete response
	resp := af.ChatResponseFromUpdates(updates)
	if resp.Text() != "Hello, world!" {
		t.Errorf("merged text = %q", resp.Text())
	}
}

func TestClient_WithOptions(t *testing.T) {
	var sentOrg string
	httpClient := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		sentOrg = req.Header.Get("OpenAI-Organization")
		return jsonResponse(200, map[string]any{
			"id": "chatcmpl-1", "model": "gpt-4o",
			"choices": []map[string]any{{
				"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": "ok"},
			}},
		}), nil
	})

	client := openai.New("test-key",
		openai.WithModel("gpt-4o"),
		openai.WithOrganization("org-abc"),
		openai.WithHTTPClient(httpClient),
	)

	_, err := client.Response(context.Background(),
		[]af.Message{af.NewUserMessage("hi")},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	if sentOrg != "org-abc" {
		t.Errorf("org header = %q", sentOrg)
	}
}

func TestClient_ChatOptions_PassedThrough(t *testing.T) {
	var sentBody map[string]any
	httpClient := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		json.Unmarshal(body, &sentBody)
		return jsonResponse(200, map[string]any{
			"id": "chatcmpl-1", "model": "gpt-4o",
			"choices": []map[string]any{{
				"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": "ok"},
			}},
		}), nil
	})

	temp := 0.3
	maxTok := 100
	client := openai.New("test-key",
		openai.WithModel("gpt-4o"),
		openai.WithHTTPClient(httpClient),
	)

	_, err := client.Response(context.Background(),
		[]af.Message{af.NewUserMessage("hi")},
		&af.ChatOptions{
			Temperature: &temp,
			MaxTokens:   &maxTok,
			ToolChoice:  af.ToolChoiceNone,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if sentBody["temperature"] != 0.3 {
		t.Errorf("temperature = %v", sentBody["temperature"])
	}
	// max_completion_tokens in OpenAI API
	if sentBody["max_completion_tokens"] != float64(100) {
		t.Errorf("max_completion_tokens = %v", sentBody["max_completion_tokens"])
	}
	if sentBody["tool_choice"] != "none" {
		t.Errorf("tool_choice = %v", sentBody["tool_choice"])
	}
}
