package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
)

type ListPermissionsQuery struct {
	Cursor *string
	Limit  int
}

type ListPermissionsResult struct {
	Permissions []*entity.Permission
	NextCursor  *string
	PrevCursor  *string
	HasNext     bool
	HasPrev     bool
}

type ListPermissionsHandler struct {
	permRepo repository.PermissionRepository
}

func NewListPermissionsHandler(permRepo repository.PermissionRepository) *ListPermissionsHandler {
	return &ListPermissionsHandler{permRepo: permRepo}
}

func (h *ListPermissionsHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(ListPermissionsQuery)
	permissions, nextCursor, prevCursor, hasNext, hasPrev, err := h.permRepo.GetAll(ctx, q.Cursor, q.Limit)
	if err != nil {
		return nil, err
	}
	return ListPermissionsResult{
		Permissions: permissions,
		NextCursor:  nextCursor,
		PrevCursor:  prevCursor,
		HasNext:     hasNext,
		HasPrev:     hasPrev,
	}, nil
}
