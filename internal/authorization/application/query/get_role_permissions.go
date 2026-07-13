package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type GetRolePermissionsQuery struct {
	RoleID uuid.UUID
}

type GetRolePermissionsHandler struct {
	rolePermRepo repository.RolePermissionRepository
}

func NewGetRolePermissionsHandler(rolePermRepo repository.RolePermissionRepository) *GetRolePermissionsHandler {
	return &GetRolePermissionsHandler{rolePermRepo: rolePermRepo}
}

func (h *GetRolePermissionsHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(GetRolePermissionsQuery)
	return h.rolePermRepo.GetPermissionsByRoleID(ctx, q.RoleID)
}
