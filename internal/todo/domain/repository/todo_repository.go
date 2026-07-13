package repository

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/google/uuid"
)

type TodoRepository interface {
	Create(ctx context.Context, todo *entity.Todo) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Todo, error)
	GetAll(ctx context.Context, cursor *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error)
	Update(ctx context.Context, todo *entity.Todo) error
	Delete(ctx context.Context, id uuid.UUID) error
	Search(ctx context.Context, query string, cursor *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error)
}
