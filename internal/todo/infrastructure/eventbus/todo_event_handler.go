package eventbus

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/event"
)

type TodoEventHandler struct {
	log domain.Logger
}

func NewTodoEventHandler(log domain.Logger) *TodoEventHandler {
	return &TodoEventHandler{log: log}
}

func (h *TodoEventHandler) Register(bus *events.EventBus) {
	bus.Subscribe(event.TodoCreatedEvent, h.onCreated)
	bus.Subscribe(event.TodoUpdatedEvent, h.onUpdated)
	bus.Subscribe(event.TodoCompletedEvent, h.onCompleted)
	bus.Subscribe(event.TodoDeletedEvent, h.onDeleted)
}

func (h *TodoEventHandler) onCreated(ctx context.Context, e events.Event) error {
	if todoEvent, ok := e.Payload.(event.TodoCreated); ok {
		h.log.Info(ctx, "todo created",
			domain.String("id", todoEvent.ID.String()),
			domain.String("title", todoEvent.Title),
		)
	}
	return nil
}

func (h *TodoEventHandler) onUpdated(ctx context.Context, e events.Event) error {
	if todoEvent, ok := e.Payload.(event.TodoUpdated); ok {
		h.log.Info(ctx, "todo updated",
			domain.String("id", todoEvent.ID.String()),
			domain.String("title", todoEvent.Title),
		)
	}
	return nil
}

func (h *TodoEventHandler) onCompleted(ctx context.Context, e events.Event) error {
	if todoEvent, ok := e.Payload.(event.TodoCompleted); ok {
		h.log.Info(ctx, "todo completed",
			domain.String("id", todoEvent.ID.String()),
			domain.String("title", todoEvent.Title),
		)
	}
	return nil
}

func (h *TodoEventHandler) onDeleted(ctx context.Context, e events.Event) error {
	if todoEvent, ok := e.Payload.(event.TodoDeleted); ok {
		h.log.Info(ctx, "todo deleted",
			domain.String("id", todoEvent.ID.String()),
		)
	}
	return nil
}
