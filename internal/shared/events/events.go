package events

import (
	"context"
	"sync"
)

type Event struct {
	Type    string
	Payload interface{}
}

type Handler func(ctx context.Context, event Event) error

// EventBus defines the contract for publishing and subscribing to events.
type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventType string, handler Handler)
}

// InMemoryEventBus is a synchronous in-memory implementation of EventBus.
type InMemoryEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewInMemoryEventBus creates a new InMemoryEventBus.
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]Handler),
	}
}

func (eb *InMemoryEventBus) Subscribe(eventType string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *InMemoryEventBus) Publish(ctx context.Context, event Event) error {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
