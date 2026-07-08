package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/mapper"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
)

type CreateTodoCommand struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type CreateTodoHandler struct {
	domainSvc *service.TodoDomainService
}

func NewCreateTodoHandler(domainSvc *service.TodoDomainService) *CreateTodoHandler {
	return &CreateTodoHandler{domainSvc: domainSvc}
}

func (h *CreateTodoHandler) Handle(ctx context.Context, cmd CreateTodoCommand) (dto.TodoResponse, error) {
	todo, err := h.domainSvc.CreateTodo(ctx, cmd.Title, cmd.Description)
	if err != nil {
		return dto.TodoResponse{}, err
	}
	return mapper.ToTodoResponse(todo), nil
}
