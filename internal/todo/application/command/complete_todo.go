package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/mapper"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	"github.com/google/uuid"
)

type CompleteTodoCommand struct {
	ID uuid.UUID
}

type CompleteTodoHandler struct {
	domainSvc *service.TodoDomainService
}

func NewCompleteTodoHandler(domainSvc *service.TodoDomainService) *CompleteTodoHandler {
	return &CompleteTodoHandler{domainSvc: domainSvc}
}

func (h *CompleteTodoHandler) Handle(ctx context.Context, cmd CompleteTodoCommand) (dto.TodoResponse, error) {
	todo, err := h.domainSvc.CompleteTodo(ctx, cmd.ID)
	if err != nil {
		return dto.TodoResponse{}, err
	}
	return mapper.ToTodoResponse(todo), nil
}
