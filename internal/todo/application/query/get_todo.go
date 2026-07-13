package query

import (
	"context"

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

func (h *GetTodoHandler) Handle(ctx context.Context, q any) (any, error) {
	query := q.(GetTodoQuery)
	todo, err := h.domainSvc.GetTodo(ctx, query.ID)
	if err != nil {
		return nil, err
	}
	return mapper.ToTodoResponse(todo), nil
}
