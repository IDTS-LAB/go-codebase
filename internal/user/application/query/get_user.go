package query

import (
	"context"

	roleProvider "github.com/IDTS-LAB/go-codebase/internal/authorization/public"
	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/google/uuid"
)

type GetUserQuery struct {
	ID uuid.UUID
}

type GetUserHandler struct {
	repo         repository.UserRepository
	roleProvider roleProvider.AuthorizationProvider
}

func NewGetUserHandler(repo repository.UserRepository, roleProvider roleProvider.AuthorizationProvider) *GetUserHandler {
	return &GetUserHandler{repo: repo, roleProvider: roleProvider}
}

func (h *GetUserHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(GetUserQuery)
	return h.repo.GetByID(ctx, q.ID)
}
