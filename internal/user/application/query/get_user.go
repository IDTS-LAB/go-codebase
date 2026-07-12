package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/google/uuid"
)

type GetUserQuery struct {
	ID uuid.UUID
}

type GetUserHandler struct {
	repo repository.UserRepository
}

func NewGetUserHandler(repo repository.UserRepository) *GetUserHandler {
	return &GetUserHandler{repo: repo}
}

func (h *GetUserHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(GetUserQuery)
	return h.repo.GetByID(ctx, q.ID)
}
