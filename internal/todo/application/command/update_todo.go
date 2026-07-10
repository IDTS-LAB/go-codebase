package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/mapper"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	"github.com/google/uuid"
)

type UpdateTodoCommand struct {
	ID          uuid.UUID
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateTodoHandler struct {
	domainSvc *service.TodoDomainService
	eventBus  events.EventBus
}

func NewUpdateTodoHandler(domainSvc *service.TodoDomainService, eventBus events.EventBus) *UpdateTodoHandler {
	return &UpdateTodoHandler{domainSvc: domainSvc, eventBus: eventBus}
}

func (h *UpdateTodoHandler) Handle(ctx context.Context, cmd UpdateTodoCommand) (dto.TodoResponse, error) {
	todo, err := h.domainSvc.UpdateTodo(ctx, cmd.ID, cmd.Title, cmd.Description)
	if err != nil {
		return dto.TodoResponse{}, err
	}

	_ = h.eventBus.Publish(ctx, events.Event{
		Type: event.TodoUpdatedEvent,
		Payload: event.TodoUpdated{
			ID:        todo.ID,
			Title:     todo.Title,
			UpdatedAt: todo.UpdatedAt,
		},
	})

	return mapper.ToTodoResponse(todo), nil
}
