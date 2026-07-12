package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/google/uuid"
)

type DeleteUserCommand struct {
	ID uuid.UUID
}

type DeleteUserHandler struct {
	repo repository.UserRepository
}

func NewDeleteUserHandler(repo repository.UserRepository) *DeleteUserHandler {
	return &DeleteUserHandler{repo: repo}
}

func (h *DeleteUserHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(DeleteUserCommand)
	user, err := h.repo.GetByID(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	user.SoftDelete()
	return nil, h.repo.Update(ctx, user)
}
