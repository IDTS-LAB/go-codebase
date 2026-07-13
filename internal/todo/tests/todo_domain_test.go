package tests

import (
	"context"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestTodo(id uuid.UUID, title string) *entity.Todo {
	return &entity.Todo{
		Entity: domain.Entity{ID: id},
		Title:  title,
	}
}

func newTestTodoWithDesc(id uuid.UUID, title, desc string) *entity.Todo {
	return &entity.Todo{
		Entity:      domain.Entity{ID: id},
		Title:       title,
		Description: desc,
	}
}

func newTestTodoCompleted(id uuid.UUID, title string) *entity.Todo {
	return &entity.Todo{
		Entity:    domain.Entity{ID: id},
		Title:     title,
		Completed: true,
	}
}

func TestTodoDomainService_CreateTodo(t *testing.T) {
	repo := new(MockTodoRepo)
	svc := service.NewTodoDomainService(repo)

	repo.On("Create", mock.Anything, mock.Anything).Return(nil)

	todo, err := svc.CreateTodo(context.Background(), "Test Title", "Test Desc")

	assert.NoError(t, err)
	assert.Equal(t, "Test Title", todo.Title)
	assert.Equal(t, "Test Desc", todo.Description)
	assert.False(t, todo.Completed)
	assert.NotEqual(t, uuid.Nil, todo.ID)
	repo.AssertExpectations(t)
}

func TestTodoDomainService_CreateTodo_EmptyTitle(t *testing.T) {
	repo := new(MockTodoRepo)
	svc := service.NewTodoDomainService(repo)

	_, err := svc.CreateTodo(context.Background(), "", "Desc")

	assert.ErrorIs(t, err, service.ErrInvalidTitle)
}

func TestTodoDomainService_GetTodo(t *testing.T) {
	repo := new(MockTodoRepo)
	svc := service.NewTodoDomainService(repo)

	id := uuid.New()
	expected := newTestTodo(id, "Test")
	repo.On("GetByID", mock.Anything, id).Return(expected, nil)

	todo, err := svc.GetTodo(context.Background(), id)

	assert.NoError(t, err)
	assert.Equal(t, expected, todo)
}

func TestTodoDomainService_GetTodo_NotFound(t *testing.T) {
	repo := new(MockTodoRepo)
	svc := service.NewTodoDomainService(repo)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, assert.AnError)

	_, err := svc.GetTodo(context.Background(), id)

	assert.ErrorIs(t, err, service.ErrTodoNotFound)
}

func TestTodoDomainService_CompleteTodo(t *testing.T) {
	repo := new(MockTodoRepo)
	svc := service.NewTodoDomainService(repo)

	id := uuid.New()
	todo := newTestTodo(id, "Test")
	repo.On("GetByID", mock.Anything, id).Return(todo, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil)

	result, err := svc.CompleteTodo(context.Background(), id)

	assert.NoError(t, err)
	assert.True(t, result.Completed)
}

func TestTodoDomainService_CompleteTodo_AlreadyDone(t *testing.T) {
	repo := new(MockTodoRepo)
	svc := service.NewTodoDomainService(repo)

	id := uuid.New()
	todo := newTestTodoCompleted(id, "Test")
	repo.On("GetByID", mock.Anything, id).Return(todo, nil)

	_, err := svc.CompleteTodo(context.Background(), id)

	assert.ErrorIs(t, err, service.ErrTodoAlreadyDone)
}

func TestTodoDomainService_DeleteTodo(t *testing.T) {
	repo := new(MockTodoRepo)
	svc := service.NewTodoDomainService(repo)

	id := uuid.New()
	todo := newTestTodo(id, "Test")
	repo.On("GetByID", mock.Anything, id).Return(todo, nil)
	repo.On("Delete", mock.Anything, id).Return(nil)

	err := svc.DeleteTodo(context.Background(), id)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestTodoDomainService_DeleteTodo_NotFound(t *testing.T) {
	repo := new(MockTodoRepo)
	svc := service.NewTodoDomainService(repo)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, assert.AnError)

	err := svc.DeleteTodo(context.Background(), id)

	assert.ErrorIs(t, err, service.ErrTodoNotFound)
}

func TestTodoDomainService_UpdateTodo(t *testing.T) {
	repo := new(MockTodoRepo)
	svc := service.NewTodoDomainService(repo)

	id := uuid.New()
	todo := newTestTodoWithDesc(id, "Old Title", "Old Desc")
	repo.On("GetByID", mock.Anything, id).Return(todo, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil)

	result, err := svc.UpdateTodo(context.Background(), id, "New Title", "New Desc")

	assert.NoError(t, err)
	assert.Equal(t, "New Title", result.Title)
	assert.Equal(t, "New Desc", result.Description)
}

func TestTodoDomainService_ListTodos(t *testing.T) {
	repo := new(MockTodoRepo)
	svc := service.NewTodoDomainService(repo)

	todos := []*entity.Todo{
		newTestTodo(uuid.New(), "Todo 1"),
		newTestTodo(uuid.New(), "Todo 2"),
	}
	repo.On("GetAll", mock.Anything, (*string)(nil), 10).Return(todos, nil)

	result, _, _, _, _, err := svc.ListTodos(context.Background(), nil, 10)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestTodoDomainService_SearchTodos(t *testing.T) {
	repo := new(MockTodoRepo)
	svc := service.NewTodoDomainService(repo)

	todos := []*entity.Todo{
		newTestTodo(uuid.New(), "Test Todo"),
	}
	repo.On("Search", mock.Anything, "test", (*string)(nil), 10).Return(todos, nil)

	result, _, _, _, _, err := svc.SearchTodos(context.Background(), "test", nil, 10)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}
