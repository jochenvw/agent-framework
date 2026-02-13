// Copyright (c) Microsoft. All rights reserved.

package agentframework

import "context"

// MessageStore persists conversation messages for a [Session].
type MessageStore interface {
	// ListMessages returns all stored messages in order.
	ListMessages(ctx context.Context) ([]Message, error)

	// AddMessages appends messages to the store.
	AddMessages(ctx context.Context, msgs []Message) error

	// Serialize returns the store's state as a serializable map.
	Serialize() (map[string]any, error)
}

// InMemoryStore is a simple in-memory [MessageStore].
type InMemoryStore struct {
	messages []Message
}

// NewInMemoryStore creates an empty [InMemoryStore].
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{}
}

func (s *InMemoryStore) ListMessages(_ context.Context) ([]Message, error) {
	cp := make([]Message, len(s.messages))
	copy(cp, s.messages)
	return cp, nil
}

func (s *InMemoryStore) AddMessages(_ context.Context, msgs []Message) error {
	s.messages = append(s.messages, msgs...)
	return nil
}

func (s *InMemoryStore) Serialize() (map[string]any, error) {
	return map[string]any{
		"messages": s.messages,
	}, nil
}
