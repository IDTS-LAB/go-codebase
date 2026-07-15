package command

import (
	"context"
	"time"

	authEntity "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/google/uuid"
)

type CreateUserCommand struct {
	Email    string
	Name     string
	IsActive bool
}

type CreateUserHandler struct {
	repo repository.UserRepository
}

func NewCreateUserHandler(repo repository.UserRepository) *CreateUserHandler {
	return &CreateUserHandler{repo: repo}
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(CreateUserCommand)
	now := time.Now().UTC()
	user := &authEntity.User{
		Entity:   domain.Entity{ID: uuid.New(), CreatedAt: now, UpdatedAt: now},
		Email:    c.Email,
		Name:     c.Name,
		IsActive: c.IsActive,
	}
	if err := h.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
