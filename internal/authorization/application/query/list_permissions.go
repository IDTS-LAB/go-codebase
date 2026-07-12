package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
)

type ListPermissionsQuery struct {
	Page    int
	PerPage int
}

type ListPermissionsResult struct {
	Permissions []*entity.Permission
	Total       int
}

type ListPermissionsHandler struct {
	permRepo repository.PermissionRepository
}

func NewListPermissionsHandler(permRepo repository.PermissionRepository) *ListPermissionsHandler {
	return &ListPermissionsHandler{permRepo: permRepo}
}

func (h *ListPermissionsHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(ListPermissionsQuery)
	offset := (q.Page - 1) * q.PerPage
	permissions, total, err := h.permRepo.GetAll(ctx, offset, q.PerPage)
	if err != nil {
		return nil, err
	}
	return ListPermissionsResult{Permissions: permissions, Total: total}, nil
}
