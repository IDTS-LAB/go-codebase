package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/application/mapper"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
)

type SearchTodosQuery struct {
	Query   string
	Page    int
	PerPage int
}

type SearchTodosHandler struct {
	domainSvc *service.TodoDomainService
}

func NewSearchTodosHandler(domainSvc *service.TodoDomainService) *SearchTodosHandler {
	return &SearchTodosHandler{domainSvc: domainSvc}
}

func (h *SearchTodosHandler) Handle(ctx context.Context, q any) (any, error) {
	query := q.(SearchTodosQuery)
	offset := (query.Page - 1) * query.PerPage
	todos, total, err := h.domainSvc.SearchTodos(ctx, query.Query, offset, query.PerPage)
	if err != nil {
		return nil, err
	}
	return mapper.ToTodoListResponse(todos, total, query.Page, query.PerPage), nil
}
