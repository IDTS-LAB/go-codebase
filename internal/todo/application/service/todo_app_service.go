package service

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/query"
	"github.com/google/uuid"
)

type TodoAppService struct {
	createHandler   *command.CreateTodoHandler
	updateHandler   *command.UpdateTodoHandler
	deleteHandler   *command.DeleteTodoHandler
	completeHandler *command.CompleteTodoHandler
	getHandler      *query.GetTodoHandler
	listHandler     *query.ListTodosHandler
	searchHandler   *query.SearchTodosHandler
}

func NewTodoAppService(
	createHandler *command.CreateTodoHandler,
	updateHandler *command.UpdateTodoHandler,
	deleteHandler *command.DeleteTodoHandler,
	completeHandler *command.CompleteTodoHandler,
	getHandler *query.GetTodoHandler,
	listHandler *query.ListTodosHandler,
	searchHandler *query.SearchTodosHandler,
) *TodoAppService {
	return &TodoAppService{
		createHandler:   createHandler,
		updateHandler:   updateHandler,
		deleteHandler:   deleteHandler,
		completeHandler: completeHandler,
		getHandler:      getHandler,
		listHandler:     listHandler,
		searchHandler:   searchHandler,
	}
}

func (s *TodoAppService) CreateTodo(ctx context.Context, req dto.CreateTodoRequest) (dto.TodoResponse, error) {
	return s.createHandler.Handle(ctx, command.CreateTodoCommand{
		Title:       req.Title,
		Description: req.Description,
	})
}

func (s *TodoAppService) GetTodo(ctx context.Context, id uuid.UUID) (dto.TodoResponse, error) {
	return s.getHandler.Handle(ctx, query.GetTodoQuery{ID: id})
}

func (s *TodoAppService) ListTodos(ctx context.Context, page, perPage int) (dto.TodoListResponse, error) {
	return s.listHandler.Handle(ctx, query.ListTodosQuery{Page: page, PerPage: perPage})
}

func (s *TodoAppService) UpdateTodo(ctx context.Context, id uuid.UUID, req dto.UpdateTodoRequest) (dto.TodoResponse, error) {
	return s.updateHandler.Handle(ctx, command.UpdateTodoCommand{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
	})
}

func (s *TodoAppService) DeleteTodo(ctx context.Context, id uuid.UUID) error {
	return s.deleteHandler.Handle(ctx, command.DeleteTodoCommand{ID: id})
}

func (s *TodoAppService) CompleteTodo(ctx context.Context, id uuid.UUID) (dto.TodoResponse, error) {
	return s.completeHandler.Handle(ctx, command.CompleteTodoCommand{ID: id})
}

func (s *TodoAppService) SearchTodos(ctx context.Context, queryStr string, page, perPage int) (dto.TodoListResponse, error) {
	return s.searchHandler.Handle(ctx, query.SearchTodosQuery{Query: queryStr, Page: page, PerPage: perPage})
}
