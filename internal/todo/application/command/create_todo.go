package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/mapper"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
)

type CreateTodoCommand struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type CreateTodoHandler struct {
	domainSvc *service.TodoDomainService
	eventBus  events.EventBus
}

func NewCreateTodoHandler(domainSvc *service.TodoDomainService, eventBus events.EventBus) *CreateTodoHandler {
	return &CreateTodoHandler{domainSvc: domainSvc, eventBus: eventBus}
}

func (h *CreateTodoHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(CreateTodoCommand)
	todo, err := h.domainSvc.CreateTodo(ctx, c.Title, c.Description)
	if err != nil {
		return nil, err
	}

	_ = h.eventBus.Publish(ctx, events.Event{
		Type: event.TodoCreatedEvent,
		Payload: event.TodoCreated{
			ID:        todo.ID,
			Title:     todo.Title,
			CreatedAt: todo.CreatedAt,
		},
	})

	return mapper.ToTodoResponse(todo), nil
}
