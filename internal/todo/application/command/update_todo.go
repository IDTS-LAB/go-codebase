package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/mapper"
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
}

func NewUpdateTodoHandler(domainSvc *service.TodoDomainService) *UpdateTodoHandler {
	return &UpdateTodoHandler{domainSvc: domainSvc}
}

func (h *UpdateTodoHandler) Handle(ctx context.Context, cmd UpdateTodoCommand) (dto.TodoResponse, error) {
	todo, err := h.domainSvc.UpdateTodo(ctx, cmd.ID, cmd.Title, cmd.Description)
	if err != nil {
		return dto.TodoResponse{}, err
	}
	return mapper.ToTodoResponse(todo), nil
}
