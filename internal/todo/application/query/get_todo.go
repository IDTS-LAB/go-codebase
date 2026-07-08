package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/mapper"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	"github.com/google/uuid"
)

type GetTodoQuery struct {
	ID uuid.UUID
}

type GetTodoHandler struct {
	domainSvc *service.TodoDomainService
}

func NewGetTodoHandler(domainSvc *service.TodoDomainService) *GetTodoHandler {
	return &GetTodoHandler{domainSvc: domainSvc}
}

func (h *GetTodoHandler) Handle(ctx context.Context, q GetTodoQuery) (dto.TodoResponse, error) {
	todo, err := h.domainSvc.GetTodo(ctx, q.ID)
	if err != nil {
		return dto.TodoResponse{}, err
	}
	return mapper.ToTodoResponse(todo), nil
}
