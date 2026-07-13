package repository

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/google/uuid"
)

type UserRepository interface {
	List(ctx context.Context, cursor *string, limit int) ([]*entity.User, *string, *string, bool, bool, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}
