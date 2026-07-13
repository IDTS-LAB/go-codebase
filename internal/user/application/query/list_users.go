package query

import (
	"context"

	authEntity "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
)

type ListUsersQuery struct {
	Cursor *string
	Limit  int
}

type ListUsersResult struct {
	Users      []*authEntity.User
	NextCursor *string
	PrevCursor *string
	HasNext    bool
	HasPrev    bool
	Limit      int
}

type ListUsersHandler struct {
	repo repository.UserRepository
}

func NewListUsersHandler(repo repository.UserRepository) *ListUsersHandler {
	return &ListUsersHandler{repo: repo}
}

func (h *ListUsersHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(ListUsersQuery)
	users, nextCursor, prevCursor, hasNext, hasPrev, err := h.repo.List(ctx, q.Cursor, q.Limit)
	if err != nil {
		return nil, err
	}
	return ListUsersResult{
		Users:      users,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Limit:      q.Limit,
	}, nil
}
