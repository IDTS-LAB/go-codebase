package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
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

func (h *SearchTodosHandler) Handle(ctx context.Context, q SearchTodosQuery) (dto.TodoListResponse, error) {
	offset := (q.Page - 1) * q.PerPage
	todos, total, err := h.domainSvc.SearchTodos(ctx, q.Query, offset, q.PerPage)
	if err != nil {
		return dto.TodoListResponse{}, err
	}
	return mapper.ToTodoListResponse(todos, total, q.Page, q.PerPage), nil
}
