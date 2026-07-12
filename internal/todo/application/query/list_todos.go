package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/application/mapper"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
)

type ListTodosQuery struct {
	Page    int
	PerPage int
}

type ListTodosHandler struct {
	domainSvc *service.TodoDomainService
}

func NewListTodosHandler(domainSvc *service.TodoDomainService) *ListTodosHandler {
	return &ListTodosHandler{domainSvc: domainSvc}
}

func (h *ListTodosHandler) Handle(ctx context.Context, q any) (any, error) {
	query := q.(ListTodosQuery)
	offset := (query.Page - 1) * query.PerPage
	todos, total, err := h.domainSvc.ListTodos(ctx, offset, query.PerPage)
	if err != nil {
		return nil, err
	}
	return mapper.ToTodoListResponse(todos, total, query.Page, query.PerPage), nil
}
