package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
)

type SearchTodosQuery struct {
	Query  string
	Cursor *string
	Limit  int
}

type SearchTodosResult struct {
	Todos      []*entity.Todo
	NextCursor *string
	PrevCursor *string
	HasNext    bool
	HasPrev    bool
	Limit      int
}

type SearchTodosHandler struct {
	domainSvc *service.TodoDomainService
}

func NewSearchTodosHandler(domainSvc *service.TodoDomainService) *SearchTodosHandler {
	return &SearchTodosHandler{domainSvc: domainSvc}
}

func (h *SearchTodosHandler) Handle(ctx context.Context, q any) (any, error) {
	query := q.(SearchTodosQuery)
	todos, nextCursor, prevCursor, hasNext, hasPrev, err := h.domainSvc.SearchTodos(ctx, query.Query, query.Cursor, query.Limit)
	if err != nil {
		return nil, err
	}
	return SearchTodosResult{
		Todos:      todos,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Limit:      query.Limit,
	}, nil
}
