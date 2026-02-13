// Copyright (c) Microsoft. All rights reserved.

package agentframework

import (
	"crypto/rand"
	"fmt"
	"sync"
)

// Session manages conversation state for an agent interaction.
// It operates in one of two mutually exclusive modes:
//   - Service-managed: conversation state lives server-side (identified by ServiceID)
//   - Locally-managed: messages are stored locally via a [MessageStore]
//
// Setting one mode locks out the other.
type Session struct {
	mu              sync.Mutex
	id              string
	serviceID       string
	store           MessageStore
	contextProvider ContextProvider
	modeLocked      bool
}

// SessionOption configures a [Session].
type SessionOption func(*Session)

// WithSessionStore sets the local message store for the session.
func WithSessionStore(store MessageStore) SessionOption {
	return func(s *Session) {
		s.store = store
	}
}

// WithSessionContextProvider attaches a context provider to the session.
func WithSessionContextProvider(cp ContextProvider) SessionOption {
	return func(s *Session) {
		s.contextProvider = cp
	}
}

// NewSession creates a new Session with a generated ID.
func NewSession(opts ...SessionOption) *Session {
	s := &Session{
		id: newUUID(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ID returns the session's unique identifier.
func (s *Session) ID() string { return s.id }

// ServiceID returns the service-managed thread ID, or empty if locally managed.
func (s *Session) ServiceID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.serviceID
}

// SetServiceID locks the session into service-managed mode.
// Returns ErrSessionModeLocked if the session is already in local mode.
func (s *Session) SetServiceID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.modeLocked && s.store != nil {
		return fmt.Errorf("%w: cannot switch to service mode", ErrSessionModeLocked)
	}
	s.serviceID = id
	s.modeLocked = true
	return nil
}

// Store returns the local message store, or nil if service-managed.
func (s *Session) Store() MessageStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store
}

// SetStore locks the session into locally-managed mode.
// Returns ErrSessionModeLocked if the session is already in service mode.
func (s *Session) SetStore(store MessageStore) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.modeLocked && s.serviceID != "" {
		return fmt.Errorf("%w: cannot switch to local mode", ErrSessionModeLocked)
	}
	s.store = store
	s.modeLocked = true
	return nil
}

// ContextProvider returns the session's context provider, if any.
func (s *Session) ContextProvider() ContextProvider { return s.contextProvider }

// Serialize returns the session state as a serializable map.
func (s *Session) Serialize() (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := map[string]any{
		"id": s.id,
	}
	if s.serviceID != "" {
		state["serviceId"] = s.serviceID
	}
	if s.store != nil {
		storeState, err := s.store.Serialize()
		if err != nil {
			return nil, fmt.Errorf("serialize store: %w", err)
		}
		state["store"] = storeState
	}
	return state, nil
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
