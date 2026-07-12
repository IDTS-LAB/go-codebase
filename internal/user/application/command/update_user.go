package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/google/uuid"
)

type UpdateUserCommand struct {
	ID       uuid.UUID
	Name     string
	Email    string
	IsActive bool
}

type UpdateUserHandler struct {
	repo repository.UserRepository
}

func NewUpdateUserHandler(repo repository.UserRepository) *UpdateUserHandler {
	return &UpdateUserHandler{repo: repo}
}

func (h *UpdateUserHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(UpdateUserCommand)
	user, err := h.repo.GetByID(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	if c.Name != "" {
		user.Name = c.Name
	}
	if c.Email != "" {
		user.Email = c.Email
	}
	user.IsActive = c.IsActive
	user.Touch()
	if err := h.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
