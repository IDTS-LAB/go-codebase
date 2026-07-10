package command

import (
	"context"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	"github.com/google/uuid"
)

type DeleteTodoCommand struct {
	ID uuid.UUID
}

type DeleteTodoHandler struct {
	domainSvc *service.TodoDomainService
	eventBus  events.EventBus
}

func NewDeleteTodoHandler(domainSvc *service.TodoDomainService, eventBus events.EventBus) *DeleteTodoHandler {
	return &DeleteTodoHandler{domainSvc: domainSvc, eventBus: eventBus}
}

func (h *DeleteTodoHandler) Handle(ctx context.Context, cmd DeleteTodoCommand) error {
	if err := h.domainSvc.DeleteTodo(ctx, cmd.ID); err != nil {
		return err
	}

	_ = h.eventBus.Publish(ctx, events.Event{
		Type: event.TodoDeletedEvent,
		Payload: event.TodoDeleted{
			ID:        cmd.ID,
			DeletedAt: time.Now().UTC(),
		},
	})

	return nil
}
