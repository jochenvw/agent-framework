// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

// clientConfig holds resolved configuration for the OpenAI client.
type clientConfig struct {
	baseURL         string
	organization    string
	httpClient      *http.Client
	headers         map[string]string
	model           string
	azureCredential azcore.TokenCredential
	chatMiddleware  []af.ChatMiddleware
}

// Option configures an OpenAI [Client].
type Option func(*clientConfig)

// WithBaseURL overrides the API base URL (e.g., for Azure OpenAI or proxies).
func WithBaseURL(url string) Option {
	return func(c *clientConfig) { c.baseURL = url }
}

// WithOrganization sets the OpenAI organization header.
func WithOrganization(org string) Option {
	return func(c *clientConfig) { c.organization = org }
}

// WithHTTPClient provides a custom http.Client for requests.
func WithHTTPClient(client *http.Client) Option {
	return func(c *clientConfig) { c.httpClient = client }
}

// WithHeaders adds custom headers to every request.
func WithHeaders(headers map[string]string) Option {
	return func(c *clientConfig) { c.headers = headers }
}

// WithModel sets the default model for requests.
func WithModel(model string) Option {
	return func(c *clientConfig) { c.model = model }
}

// WithAzureCredential enables Azure AD token authentication using the provided credential.
// When set, the client will obtain and refresh tokens automatically instead of using API keys.
func WithAzureCredential(cred azcore.TokenCredential) Option {
	return func(c *clientConfig) { c.azureCredential = cred }
}

// WithChatMiddleware adds middleware to the chat pipeline.
// Middleware is applied in the order provided (first = outermost).
func WithChatMiddleware(mw ...af.ChatMiddleware) Option {
	return func(c *clientConfig) { c.chatMiddleware = append(c.chatMiddleware, mw...) }
}
