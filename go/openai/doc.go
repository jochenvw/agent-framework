// Copyright (c) Microsoft. All rights reserved.

// Package openai provides a [ChatClient] implementation for the OpenAI
// Chat Completions API.
//
// Create a client and pass it to [agentframework.NewAgent]:
//
//	client := openai.New(os.Getenv("OPENAI_API_KEY"),
//	    openai.WithModel("gpt-4o"),
//	)
//
//	agent := agentframework.NewAgent(client)
//
// The client supports both synchronous and streaming responses,
// tool/function calling, and all standard ChatOptions.
//
// # Configuration
//
// Use functional options to configure the client:
//
//   - [WithModel]: set the default model
//   - [WithBaseURL]: override the API endpoint (e.g., Azure OpenAI)
//   - [WithOrganization]: set the OpenAI organization header
//   - [WithHTTPClient]: provide a custom http.Client
//   - [WithHeaders]: add custom headers to every request
//
// # Testing
//
// The client uses an unexported transport interface internally.
// For testing, provide a mock http.Client via [WithHTTPClient]
// with a custom RoundTripper.
package openai
