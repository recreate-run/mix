package events

import (
	"context"
	"sync"
)

// Broker is a generic event broker for publishing and subscribing to events
type Broker[T any] struct {
	subscribers []chan T
	mu          sync.RWMutex
}

// NewBroker creates a new event broker
func NewBroker[T any]() *Broker[T] {
	return &Broker[T]{
		subscribers: make([]chan T, 0),
	}
}

// Subscribe creates a new subscription channel for events
// The channel is buffered with size 100 to prevent blocking publishers
func (b *Broker[T]) Subscribe(ctx context.Context) <-chan T {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan T, 100)
	b.subscribers = append(b.subscribers, ch)

	// Clean up channel when context is cancelled
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		defer b.mu.Unlock()

		// Remove channel from subscribers
		for i, sub := range b.subscribers {
			if sub == ch {
				b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
				close(ch)
				break
			}
		}
	}()

	return ch
}

// Publish sends an event to all subscribers
// Non-blocking - drops events if subscriber channels are full
func (b *Broker[T]) Publish(ctx context.Context, event T) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			// Drop event if channel is full (non-blocking)
		}
	}
}

// Close closes all subscriber channels
func (b *Broker[T]) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.subscribers {
		close(ch)
	}
	b.subscribers = nil
}
