package query

import (
	"context"

	authEntity "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
)

type ListUsersQuery struct {
	Offset int
	Limit  int
}

type ListUsersResult struct {
	Users []*authEntity.User
	Total int
}

type ListUsersHandler struct {
	repo repository.UserRepository
}

func NewListUsersHandler(repo repository.UserRepository) *ListUsersHandler {
	return &ListUsersHandler{repo: repo}
}

func (h *ListUsersHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(ListUsersQuery)
	users, total, err := h.repo.List(ctx, q.Offset, q.Limit)
	if err != nil {
		return nil, err
	}
	return ListUsersResult{Users: users, Total: total}, nil
}
