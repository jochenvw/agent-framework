// Copyright (c) Microsoft. All rights reserved.

package agentframework

import (
	"context"
	"sync"
)

// ResponseStream provides a pull-based iterator over streaming responses.
// It wraps a channel internally but exposes a cleaner API with error
// propagation and cleanup guarantees.
//
// Callers must call Close when done, or use a context with cancellation.
type ResponseStream[T any] struct {
	ch      <-chan T
	errCh   <-chan error
	cancel  context.CancelFunc
	closeOnce sync.Once
	err     error
}

// NewResponseStream creates a ResponseStream by running producer in a goroutine.
// The producer should send values to the channel and return any error.
// The channel is closed automatically when the producer returns.
func NewResponseStream[T any](ctx context.Context, producer func(ctx context.Context, ch chan<- T) error) *ResponseStream[T] {
	ctx, cancel := context.WithCancel(ctx)
	ch := make(chan T, 1) // small buffer to reduce goroutine blocking
	errCh := make(chan error, 1)

	go func() {
		defer close(ch)
		if err := producer(ctx, ch); err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	return &ResponseStream[T]{
		ch:     ch,
		errCh:  errCh,
		cancel: cancel,
	}
}

// Next returns the next value from the stream.
// ok is false when the stream is exhausted. err is non-nil on failure.
func (s *ResponseStream[T]) Next(ctx context.Context) (val T, ok bool, err error) {
	select {
	case <-ctx.Done():
		var zero T
		return zero, false, ctx.Err()
	case v, open := <-s.ch:
		if !open {
			// Channel closed — check for producer error
			select {
			case e := <-s.errCh:
				s.err = e
			default:
			}
			var zero T
			return zero, false, s.err
		}
		return v, true, nil
	}
}

// Collect drains the entire stream and returns all values.
func (s *ResponseStream[T]) Collect(ctx context.Context) ([]T, error) {
	var items []T
	for {
		val, ok, err := s.Next(ctx)
		if err != nil {
			return items, err
		}
		if !ok {
			return items, nil
		}
		items = append(items, val)
	}
}

// Close cancels the producer and releases resources.
// Safe to call multiple times.
func (s *ResponseStream[T]) Close() error {
	s.closeOnce.Do(func() {
		s.cancel()
		// Drain remaining items to unblock producer
		for range s.ch {
		}
		// Drain error channel
		select {
		case e := <-s.errCh:
			if s.err == nil {
				s.err = e
			}
		default:
		}
	})
	return nil
}

// AgentResponseStream wraps a [ResponseStream] of [AgentResponseUpdate] and
// provides a FinalResponse method that collects all updates and merges them.
type AgentResponseStream struct {
	stream  *ResponseStream[AgentResponseUpdate]
	updates []AgentResponseUpdate
}

// NewAgentResponseStream wraps a raw update stream.
func NewAgentResponseStream(stream *ResponseStream[AgentResponseUpdate]) *AgentResponseStream {
	return &AgentResponseStream{stream: stream}
}

// Next returns the next streaming update.
func (s *AgentResponseStream) Next(ctx context.Context) (AgentResponseUpdate, bool, error) {
	val, ok, err := s.stream.Next(ctx)
	if ok {
		s.updates = append(s.updates, val)
	}
	return val, ok, err
}

// FinalResponse collects remaining updates and returns the merged [AgentResponse].
// After calling this, the stream is fully consumed.
func (s *AgentResponseStream) FinalResponse(ctx context.Context) (*AgentResponse, error) {
	for {
		val, ok, err := s.stream.Next(ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		s.updates = append(s.updates, val)
	}
	return AgentResponseFromUpdates(s.updates), nil
}

// Close releases the underlying stream resources.
func (s *AgentResponseStream) Close() error {
	return s.stream.Close()
}

// MapStream transforms a ResponseStream[A] into a ResponseStream[B] using fn.
func MapStream[A, B any](ctx context.Context, src *ResponseStream[A], fn func(A) B) *ResponseStream[B] {
	return NewResponseStream[B](ctx, func(ctx context.Context, ch chan<- B) error {
		defer src.Close()
		for {
			val, ok, err := src.Next(ctx)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			select {
			case ch <- fn(val):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
}
