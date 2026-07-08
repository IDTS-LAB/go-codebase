package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
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

func (h *ListTodosHandler) Handle(ctx context.Context, q ListTodosQuery) (dto.TodoListResponse, error) {
	offset := (q.Page - 1) * q.PerPage
	todos, total, err := h.domainSvc.ListTodos(ctx, offset, q.PerPage)
	if err != nil {
		return dto.TodoListResponse{}, err
	}
	return mapper.ToTodoListResponse(todos, total, q.Page, q.PerPage), nil
}
