package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/mapper"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	"github.com/google/uuid"
)

type CompleteTodoCommand struct {
	ID uuid.UUID
}

type CompleteTodoHandler struct {
	domainSvc *service.TodoDomainService
	eventBus  events.EventBus
}

func NewCompleteTodoHandler(domainSvc *service.TodoDomainService, eventBus events.EventBus) *CompleteTodoHandler {
	return &CompleteTodoHandler{domainSvc: domainSvc, eventBus: eventBus}
}

func (h *CompleteTodoHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(CompleteTodoCommand)
	todo, err := h.domainSvc.CompleteTodo(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	_ = h.eventBus.Publish(ctx, events.Event{
		Type: event.TodoCompletedEvent,
		Payload: event.TodoCompleted{
			ID:        todo.ID,
			Title:     todo.Title,
			UpdatedAt: todo.UpdatedAt,
		},
	})

	return mapper.ToTodoResponse(todo), nil
}
