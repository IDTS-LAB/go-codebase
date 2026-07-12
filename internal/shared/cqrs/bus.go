package cqrs

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type CommandHandler interface {
	Handle(ctx context.Context, cmd any) (any, error)
}

type CommandBus interface {
	Dispatch(ctx context.Context, cmd any) (any, error)
	Register(cmd any, handler CommandHandler)
}

type QueryHandler interface {
	Handle(ctx context.Context, query any) (any, error)
}

type QueryBus interface {
	Ask(ctx context.Context, query any) (any, error)
	Register(query any, handler QueryHandler)
}

type inMemoryCommandBus struct {
	mu       sync.RWMutex
	handlers map[string]CommandHandler
}

func NewInMemoryCommandBus() *inMemoryCommandBus {
	return &inMemoryCommandBus{handlers: make(map[string]CommandHandler)}
}

func (b *inMemoryCommandBus) Register(cmd any, handler CommandHandler) {
	key := reflect.TypeOf(cmd).String()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[key] = handler
}

func (b *inMemoryCommandBus) Dispatch(ctx context.Context, cmd any) (any, error) {
	key := reflect.TypeOf(cmd).String()
	b.mu.RLock()
	handler, ok := b.handlers[key]
	b.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no handler registered for command: %s", key)
	}
	return handler.Handle(ctx, cmd)
}

type inMemoryQueryBus struct {
	mu       sync.RWMutex
	handlers map[string]QueryHandler
}

func NewInMemoryQueryBus() *inMemoryQueryBus {
	return &inMemoryQueryBus{handlers: make(map[string]QueryHandler)}
}

func (b *inMemoryQueryBus) Register(query any, handler QueryHandler) {
	key := reflect.TypeOf(query).String()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[key] = handler
}

func (b *inMemoryQueryBus) Ask(ctx context.Context, query any) (any, error) {
	key := reflect.TypeOf(query).String()
	b.mu.RLock()
	handler, ok := b.handlers[key]
	b.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no handler registered for query: %s", key)
	}
	return handler.Handle(ctx, query)
}
