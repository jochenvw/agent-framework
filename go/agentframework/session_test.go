// Copyright (c) Microsoft. All rights reserved.

package agentframework_test

import (
	"context"
	"errors"
	"testing"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

func TestSessionDualMode_ServiceManaged(t *testing.T) {
	s := af.NewSession()

	if s.ID() == "" {
		t.Fatal("session ID should not be empty")
	}

	// Set service ID
	if err := s.SetServiceID("thread-123"); err != nil {
		t.Fatalf("SetServiceID: %v", err)
	}
	if s.ServiceID() != "thread-123" {
		t.Errorf("ServiceID = %q", s.ServiceID())
	}

	// Now trying to set a store should fail
	err := s.SetStore(af.NewInMemoryStore())
	if err == nil {
		t.Fatal("expected error switching to local mode")
	}
	if !errors.Is(err, af.ErrSessionModeLocked) {
		t.Errorf("error = %v, want ErrSessionModeLocked", err)
	}
}

func TestSessionDualMode_LocallyManaged(t *testing.T) {
	store := af.NewInMemoryStore()
	s := af.NewSession(af.WithSessionStore(store))

	// Lock into local mode
	if err := s.SetStore(store); err != nil {
		t.Fatalf("SetStore: %v", err)
	}

	// Now trying to set service ID should fail
	err := s.SetServiceID("thread-456")
	if err == nil {
		t.Fatal("expected error switching to service mode")
	}
	if !errors.Is(err, af.ErrSessionModeLocked) {
		t.Errorf("error = %v, want ErrSessionModeLocked", err)
	}
}

func TestInMemoryStore(t *testing.T) {
	store := af.NewInMemoryStore()
	ctx := context.Background()

	// Initially empty
	msgs, err := store.ListMessages(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("initial len = %d", len(msgs))
	}

	// Add messages
	err = store.AddMessages(ctx, []af.Message{
		af.NewUserMessage("hello"),
		af.NewAssistantMessage("hi there"),
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	msgs, err = store.ListMessages(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("len = %d, want 2", len(msgs))
	}
	if msgs[0].Text() != "hello" {
		t.Errorf("[0].Text() = %q", msgs[0].Text())
	}

	// ListMessages returns a copy
	msgs[0] = af.NewAssistantMessage("modified")
	original, _ := store.ListMessages(ctx)
	if original[0].Text() != "hello" {
		t.Error("ListMessages should return a copy")
	}
}

func TestSessionSerialize(t *testing.T) {
	s := af.NewSession()
	if err := s.SetServiceID("thread-abc"); err != nil {
		t.Fatal(err)
	}

	state, err := s.Serialize()
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	if state["id"] != s.ID() {
		t.Errorf("id = %v", state["id"])
	}
	if state["serviceId"] != "thread-abc" {
		t.Errorf("serviceId = %v", state["serviceId"])
	}
}
