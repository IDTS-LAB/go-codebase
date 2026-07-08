package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	"github.com/google/uuid"
)

type DeleteTodoCommand struct {
	ID uuid.UUID
}

type DeleteTodoHandler struct {
	domainSvc *service.TodoDomainService
}

func NewDeleteTodoHandler(domainSvc *service.TodoDomainService) *DeleteTodoHandler {
	return &DeleteTodoHandler{domainSvc: domainSvc}
}

func (h *DeleteTodoHandler) Handle(ctx context.Context, cmd DeleteTodoCommand) error {
	return h.domainSvc.DeleteTodo(ctx, cmd.ID)
}
