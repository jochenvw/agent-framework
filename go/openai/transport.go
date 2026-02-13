// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	af "github.com/microsoft/agent-framework/go/agentframework"
)

const defaultBaseURL = "https://api.openai.com/v1"

// transport is an unexported interface for HTTP communication.
// The default implementation uses net/http; tests inject a mock.
type transport interface {
	do(ctx context.Context, method, path string, body any) (*http.Response, error)
}

// httpTransport is the default transport using net/http.
type httpTransport struct {
	client          *http.Client
	baseURL         string
	apiKey          string
	org             string
	headers         map[string]string
	azureCredential azcore.TokenCredential
}

func newHTTPTransport(apiKey string, opts *clientConfig) *httpTransport {
	t := &httpTransport{
		client:          opts.httpClient,
		baseURL:         opts.baseURL,
		apiKey:          apiKey,
		org:             opts.organization,
		headers:         opts.headers,
		azureCredential: opts.azureCredential,
	}
	if t.client == nil {
		t.client = http.DefaultClient
	}
	if t.baseURL == "" {
		t.baseURL = defaultBaseURL
	}
	return t
}

func (t *httpTransport) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	url := t.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	
	// Handle authentication
	if t.azureCredential != nil {
		// Azure AD token authentication
		slog.DebugContext(ctx, "acquiring Azure AD token for Cognitive Services")
		token, err := t.azureCredential.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{"https://cognitiveservices.azure.com/.default"},
		})
		if err != nil {
			return nil, fmt.Errorf("get azure token: %w", err)
		}
		slog.DebugContext(ctx, "using Azure AD token authentication", "token_expires_on", token.ExpiresOn)
		req.Header.Set("Authorization", "Bearer "+token.Token)
	} else if _, ok := t.headers["api-key"]; !ok {
		// API key authentication (skip if Azure "api-key" header is set)
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}
	
	if t.org != "" {
		req.Header.Set("OpenAI-Organization", t.org)
	}
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, parseErrorResponse(resp)
	}

	return resp, nil
}

// parseErrorResponse reads an error response body and returns a typed error.
func parseErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var apiErr struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &apiErr)

	msg := apiErr.Error.Message
	if msg == "" {
		msg = string(body)
	}

	svcErr := &af.ServiceError{
		StatusCode: resp.StatusCode,
		Message:    msg,
		Code:       apiErr.Error.Code,
	}

	switch {
	case apiErr.Error.Code == "content_filter":
		svcErr.Err = af.ErrContentFilter
	case resp.StatusCode == 401 || resp.StatusCode == 403:
		svcErr.Err = af.ErrAuth
	case resp.StatusCode == 400:
		svcErr.Err = af.ErrInvalidRequest
	default:
		svcErr.Err = af.ErrService
	}

	return svcErr
}
