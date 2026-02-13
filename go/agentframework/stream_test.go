// Copyright (c) Microsoft. All rights reserved.

package agentframework_test

import (
	"context"
	"testing"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

func TestResponseStream_Collect(t *testing.T) {
	stream := af.NewResponseStream(context.Background(), func(ctx context.Context, ch chan<- int) error {
		for i := 1; i <= 3; i++ {
			ch <- i
		}
		return nil
	})
	defer stream.Close()

	items, err := stream.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len = %d, want 3", len(items))
	}
	for i, v := range items {
		if v != i+1 {
			t.Errorf("[%d] = %d, want %d", i, v, i+1)
		}
	}
}

func TestResponseStream_Next(t *testing.T) {
	stream := af.NewResponseStream(context.Background(), func(ctx context.Context, ch chan<- string) error {
		ch <- "a"
		ch <- "b"
		return nil
	})
	defer stream.Close()

	ctx := context.Background()

	v1, ok, err := stream.Next(ctx)
	if err != nil || !ok || v1 != "a" {
		t.Errorf("next1: val=%q ok=%v err=%v", v1, ok, err)
	}

	v2, ok, err := stream.Next(ctx)
	if err != nil || !ok || v2 != "b" {
		t.Errorf("next2: val=%q ok=%v err=%v", v2, ok, err)
	}

	_, ok, err = stream.Next(ctx)
	if ok {
		t.Error("expected stream to be exhausted")
	}
	if err != nil {
		t.Errorf("unexpected error after exhaustion: %v", err)
	}
}

func TestResponseStream_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	stream := af.NewResponseStream(ctx, func(ctx context.Context, ch chan<- int) error {
		for {
			select {
			case ch <- 42:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})

	// Read one value to confirm it's working
	v, ok, err := stream.Next(ctx)
	if err != nil || !ok || v != 42 {
		t.Fatalf("first next: val=%d ok=%v err=%v", v, ok, err)
	}

	cancel()
	stream.Close()
}

func TestResponseStream_ProducerError(t *testing.T) {
	expectedErr := af.ErrService

	stream := af.NewResponseStream(context.Background(), func(ctx context.Context, ch chan<- int) error {
		ch <- 1
		return expectedErr
	})
	defer stream.Close()

	ctx := context.Background()
	_, _, _ = stream.Next(ctx) // consume the value

	_, ok, err := stream.Next(ctx)
	if ok {
		t.Error("expected stream to be exhausted after error")
	}
	if err == nil {
		t.Fatal("expected error from producer")
	}
}

func TestChatResponseFromUpdates(t *testing.T) {
	updates := []af.ChatResponseUpdate{
		{
			Role:       af.RoleAssistant,
			ResponseID: "resp-1",
			Contents:   af.Contents{&af.TextContent{Text: "Hello, "}},
		},
		{
			Contents: af.Contents{&af.TextContent{Text: "world!"}},
		},
		{
			FinishReason: af.FinishReasonStop,
			Usage:        af.UsageDetails{InputTokens: 5, OutputTokens: 3, TotalTokens: 8},
		},
	}

	resp := af.ChatResponseFromUpdates(updates)

	if resp.ResponseID != "resp-1" {
		t.Errorf("ResponseID = %q", resp.ResponseID)
	}
	if resp.FinishReason != af.FinishReasonStop {
		t.Errorf("FinishReason = %q", resp.FinishReason)
	}
	if resp.Usage.TotalTokens != 8 {
		t.Errorf("TotalTokens = %d", resp.Usage.TotalTokens)
	}
	if len(resp.Messages) != 1 {
		t.Fatalf("messages len = %d", len(resp.Messages))
	}
	// Text deltas should be merged
	if resp.Text() != "Hello, world!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Hello, world!")
	}
}

func TestMapStream(t *testing.T) {
	ctx := context.Background()
	src := af.NewResponseStream(ctx, func(ctx context.Context, ch chan<- int) error {
		ch <- 1
		ch <- 2
		ch <- 3
		return nil
	})

	mapped := af.MapStream(ctx, src, func(i int) string {
		return string(rune('a' + i - 1))
	})
	defer mapped.Close()

	items, err := mapped.Collect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("len = %d", len(items))
	}
	expected := []string{"a", "b", "c"}
	for i, v := range items {
		if v != expected[i] {
			t.Errorf("[%d] = %q, want %q", i, v, expected[i])
		}
	}
}
