package service

import (
	"context"
	"errors"

	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/repository"
	"github.com/google/uuid"
)

var (
	ErrTodoNotFound    = errors.New("todo not found")
	ErrTodoAlreadyDone = errors.New("todo is already completed")
	ErrInvalidTitle    = errors.New("title is required")
)

type TodoDomainService struct {
	repo repository.TodoRepository
}

func NewTodoDomainService(repo repository.TodoRepository) *TodoDomainService {
	return &TodoDomainService{repo: repo}
}

func (s *TodoDomainService) CreateTodo(ctx context.Context, title, description string) (*entity.Todo, error) {
	if title == "" {
		return nil, ErrInvalidTitle
	}

	todo := entity.NewTodo(title, description)
	if err := s.repo.Create(ctx, todo); err != nil {
		return nil, err
	}
	return todo, nil
}

func (s *TodoDomainService) GetTodo(ctx context.Context, id uuid.UUID) (*entity.Todo, error) {
	todo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTodoNotFound
	}
	return todo, nil
}

func (s *TodoDomainService) ListTodos(ctx context.Context, cursor *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error) {
	return s.repo.GetAll(ctx, cursor, limit)
}

func (s *TodoDomainService) UpdateTodo(ctx context.Context, id uuid.UUID, title, description string) (*entity.Todo, error) {
	todo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTodoNotFound
	}

	todo.Update(title, description)
	if err := s.repo.Update(ctx, todo); err != nil {
		return nil, err
	}
	return todo, nil
}

func (s *TodoDomainService) DeleteTodo(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrTodoNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *TodoDomainService) CompleteTodo(ctx context.Context, id uuid.UUID) (*entity.Todo, error) {
	todo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTodoNotFound
	}

	if todo.Completed {
		return nil, ErrTodoAlreadyDone
	}

	todo.Complete()
	if err := s.repo.Update(ctx, todo); err != nil {
		return nil, err
	}
	return todo, nil
}

func (s *TodoDomainService) SearchTodos(ctx context.Context, query string, cursor *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error) {
	return s.repo.Search(ctx, query, cursor, limit)
}
