package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
)

type ListTodosQuery struct {
	Cursor *string
	Limit  int
}

type ListTodosResult struct {
	Todos      []*entity.Todo
	NextCursor *string
	PrevCursor *string
	HasNext    bool
	HasPrev    bool
	Limit      int
}

type ListTodosHandler struct {
	domainSvc *service.TodoDomainService
}

func NewListTodosHandler(domainSvc *service.TodoDomainService) *ListTodosHandler {
	return &ListTodosHandler{domainSvc: domainSvc}
}

func (h *ListTodosHandler) Handle(ctx context.Context, q any) (any, error) {
	query := q.(ListTodosQuery)
	todos, nextCursor, prevCursor, hasNext, hasPrev, err := h.domainSvc.ListTodos(ctx, query.Cursor, query.Limit)
	if err != nil {
		return nil, err
	}
	return ListTodosResult{
		Todos:      todos,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Limit:      query.Limit,
	}, nil
}
