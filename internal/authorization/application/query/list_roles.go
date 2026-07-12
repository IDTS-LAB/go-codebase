package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
)

type ListRolesQuery struct {
	Page    int
	PerPage int
}

type ListRolesResult struct {
	Roles []*entity.Role
	Total int
}

type ListRolesHandler struct {
	roleRepo repository.RoleRepository
}

func NewListRolesHandler(roleRepo repository.RoleRepository) *ListRolesHandler {
	return &ListRolesHandler{roleRepo: roleRepo}
}

func (h *ListRolesHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(ListRolesQuery)
	offset := (q.Page - 1) * q.PerPage
	roles, total, err := h.roleRepo.GetAll(ctx, offset, q.PerPage)
	if err != nil {
		return nil, err
	}
	return ListRolesResult{Roles: roles, Total: total}, nil
}
