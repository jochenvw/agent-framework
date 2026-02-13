// Copyright (c) Microsoft. All rights reserved.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

// ── Simple /invoke types ─────────────────────────────────────────────

// InvokeRequest is the JSON body for POST /invoke.
type InvokeRequest struct {
	Input          string         `json:"input"`
	ConversationID string         `json:"conversationId,omitempty"`
	Context        map[string]any `json:"context,omitempty"`
}

// InvokeResponse is the JSON body returned from POST /invoke.
type InvokeResponse struct {
	Output    string         `json:"output"`
	ToolCalls any            `json:"toolCalls,omitempty"`
	State     map[string]any `json:"state,omitempty"`
}

// ── A2A JSON-RPC types ───────────────────────────────────────────────

// jsonRPCRequest is the top-level JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// jsonRPCResponse is the top-level JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any         `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// messageSendParams is the params for A2A "message/send".
type messageSendParams struct {
	Message  a2aMessage     `json:"message"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// a2aMessage is an A2A protocol message (request or response).
type a2aMessage struct {
	Kind             string         `json:"kind"`
	Role             string         `json:"role"`
	MessageID        string         `json:"messageId"`
	ContextID        string         `json:"contextId,omitempty"`
	ReferenceTaskIDs []string       `json:"referenceTaskIds,omitempty"`
	Parts            []a2aPart      `json:"parts"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

// a2aPart is a content part in an A2A message.
type a2aPart struct {
	Kind string `json:"kind"`
	Text string `json:"text,omitempty"`
}

// taskGetParams is the params for A2A "tasks/get".
type taskGetParams struct {
	ID string `json:"id"`
}

// ── Server ───────────────────────────────────────────────────────────

// agentServer is the HTTP handler for the agent.
type agentServer struct {
	agent    *af.Agent
	apiKey   string
	port     string
	mu       sync.Mutex
	sessions map[string]*af.Session
	mux      *http.ServeMux
}

// newAgentServer creates a server. If apiKey is empty, /invoke is unauthenticated.
func newAgentServer(agent *af.Agent, apiKey string, port string) *agentServer {
	s := &agentServer{
		agent:    agent,
		apiKey:   apiKey,
		port:     port,
		sessions: make(map[string]*af.Session),
		mux:      http.NewServeMux(),
	}
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /.well-known/agent-card.json", s.handleAgentCard)
	s.mux.HandleFunc("GET /.well-known/agent.json", s.handleAgentCard)
	s.mux.HandleFunc("POST /invoke", s.handleInvoke)
	// A2A JSON-RPC endpoint — handles message/send, tasks/get at the root path.
	s.mux.HandleFunc("POST /", s.handleA2A)
	return s
}

func (s *agentServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[http] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	s.mux.ServeHTTP(w, r)
}

func (s *agentServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *agentServer) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	// Load the static card from agent-card.json.
	data, err := os.ReadFile("agent-card.json")
	if err != nil {
		log.Printf("[agent] failed to read agent-card.json: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "agent card not found"})
		return
	}

	var card map[string]any
	if err := json.Unmarshal(data, &card); err != nil {
		log.Printf("[agent] failed to parse agent-card.json: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid agent card"})
		return
	}

	// Inject dynamic fields: url, authentication.
	baseURL := s.resolveBaseURL(r)
	card["url"] = baseURL + "/"

	authSchemes := []map[string]any{}
	if s.apiKey != "" {
		authSchemes = append(authSchemes, map[string]any{
			"scheme": "bearer",
		})
	}
	card["authentication"] = map[string]any{
		"schemes": authSchemes,
	}

	writeJSON(w, http.StatusOK, card)
}

// ── A2A JSON-RPC handler ─────────────────────────────────────────────

func (s *agentServer) handleA2A(w http.ResponseWriter, r *http.Request) {
	var rpcReq jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&rpcReq); err != nil {
		log.Printf("[a2a] bad JSON-RPC request: %v", err)
		s.writeRPCError(w, nil, -32700, "Parse error")
		return
	}

	log.Printf("[a2a] method=%s id=%s", rpcReq.Method, string(rpcReq.ID))

	switch rpcReq.Method {
	case "message/send":
		s.handleMessageSend(w, r, &rpcReq)
	case "tasks/get":
		s.handleTasksGet(w, &rpcReq)
	default:
		log.Printf("[a2a] unknown method: %s", rpcReq.Method)
		s.writeRPCError(w, rpcReq.ID, -32601, fmt.Sprintf("Method not found: %s", rpcReq.Method))
	}
}

func (s *agentServer) handleMessageSend(w http.ResponseWriter, r *http.Request, rpcReq *jsonRPCRequest) {
	// Auth check.
	if s.apiKey != "" {
		token := extractBearer(r)
		if token != s.apiKey {
			log.Printf("[a2a] unauthorized request from %s", r.RemoteAddr)
			s.writeRPCError(w, rpcReq.ID, -32000, "Unauthorized")
			return
		}
	}

	var params messageSendParams
	if err := json.Unmarshal(rpcReq.Params, &params); err != nil {
		log.Printf("[a2a] bad message/send params: %v", err)
		s.writeRPCError(w, rpcReq.ID, -32602, "Invalid params")
		return
	}

	// Extract text from message parts.
	var inputTexts []string
	for _, part := range params.Message.Parts {
		if part.Kind == "text" && part.Text != "" {
			inputTexts = append(inputTexts, part.Text)
		}
	}
	input := strings.Join(inputTexts, "\n")

	if input == "" {
		log.Printf("[a2a] message/send with no text content")
		s.writeRPCError(w, rpcReq.ID, -32602, "No text content in message")
		return
	}

	contextID := params.Message.ContextID
	log.Printf("[a2a] message/send context=%s input=%q", contextID, input)

	// Get or create session from contextId.
	session := s.getOrCreateSession(contextID)

	// Run the agent.
	resp, err := s.agent.Run(r.Context(),
		[]af.Message{af.NewUserMessage(input)},
		af.WithSession(session),
	)
	if err != nil {
		log.Printf("[a2a] agent error: %v", err)
		s.writeRPCError(w, rpcReq.ID, -32000, fmt.Sprintf("Agent error: %v", err))
		return
	}

	output := resp.Text()
	log.Printf("[a2a] response=%q", output)

	// Return an A2A AgentMessage response.
	result := a2aMessage{
		Kind:      "message",
		Role:      "agent",
		MessageID: fmt.Sprintf("resp-%s", string(rpcReq.ID)),
		ContextID: contextID,
		Parts: []a2aPart{
			{Kind: "text", Text: output},
		},
	}

	s.writeRPCResult(w, rpcReq.ID, result)
}

func (s *agentServer) handleTasksGet(w http.ResponseWriter, rpcReq *jsonRPCRequest) {
	// We don't support long-running tasks; return not-found.
	var params taskGetParams
	if err := json.Unmarshal(rpcReq.Params, &params); err != nil {
		s.writeRPCError(w, rpcReq.ID, -32602, "Invalid params")
		return
	}
	log.Printf("[a2a] tasks/get id=%s (not supported, returning error)", params.ID)
	s.writeRPCError(w, rpcReq.ID, -32001, "Task not found (this agent only supports synchronous message/send)")
}

func (s *agentServer) writeRPCResult(w http.ResponseWriter, id json.RawMessage, result any) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *agentServer) writeRPCError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message},
	}
	writeJSON(w, http.StatusOK, resp)
}

// ── Simple /invoke handler ───────────────────────────────────────────

func (s *agentServer) handleInvoke(w http.ResponseWriter, r *http.Request) {
	log.Printf("[agent] invoke received")

	// Auth check.
	if s.apiKey != "" {
		token := extractBearer(r)
		if token != s.apiKey {
			log.Printf("[agent] unauthorized invoke attempt from %s", r.RemoteAddr)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		log.Printf("[agent] auth=OK")
	}

	var req InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[agent] bad request: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Input == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "input is required"})
		return
	}

	if req.ConversationID != "" {
		log.Printf("[agent] conversation=%s", req.ConversationID)
	}
	log.Printf("[agent] user input=%q", req.Input)

	session := s.getOrCreateSession(req.ConversationID)

	resp, err := s.agent.Run(r.Context(),
		[]af.Message{af.NewUserMessage(req.Input)},
		af.WithSession(session),
	)
	if err != nil {
		log.Printf("[agent] error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "agent execution failed"})
		return
	}

	out := InvokeResponse{
		Output: resp.Text(),
	}
	body, _ := json.Marshal(out)
	log.Printf("[agent] response sent (%d bytes)", len(body))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s *agentServer) getOrCreateSession(id string) *af.Session {
	if id == "" {
		return s.agent.NewSession()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		sess = s.agent.NewSession()
		s.sessions[id] = sess
	}
	return sess
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && h[:len(prefix)] == prefix {
		return h[len(prefix):]
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		log.Printf("[agent] failed to write response: %v", err)
	}
}

func (s *agentServer) resolveBaseURL(r *http.Request) string {
	baseURL := os.Getenv("DEVTUNNEL_URL")
	if baseURL == "" {
		host := r.Header.Get("X-Forwarded-Host")
		scheme := r.Header.Get("X-Forwarded-Proto")
		if host == "" {
			host = r.Host
		}
		if scheme == "" {
			scheme = "http"
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, host)
	}
	return strings.TrimRight(baseURL, "/")
}
