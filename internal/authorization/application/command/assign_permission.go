package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type AssignPermissionCommand struct {
	RoleID       uuid.UUID
	PermissionID uuid.UUID
}

type AssignPermissionHandler struct {
	roleRepo     repository.RoleRepository
	permRepo     repository.PermissionRepository
	rolePermRepo repository.RolePermissionRepository
	enforcer     Enforcer
}

func NewAssignPermissionHandler(roleRepo repository.RoleRepository, permRepo repository.PermissionRepository, rolePermRepo repository.RolePermissionRepository, enforcer Enforcer) *AssignPermissionHandler {
	return &AssignPermissionHandler{roleRepo: roleRepo, permRepo: permRepo, rolePermRepo: rolePermRepo, enforcer: enforcer}
}

func (h *AssignPermissionHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(AssignPermissionCommand)
	_, err := h.roleRepo.GetByID(ctx, c.RoleID)
	if err != nil {
		return nil, err
	}
	_, err = h.permRepo.GetByID(ctx, c.PermissionID)
	if err != nil {
		return nil, err
	}
	if err := h.rolePermRepo.Assign(ctx, entity.NewRolePermission(c.RoleID, c.PermissionID)); err != nil {
		return nil, err
	}
	return nil, h.enforcer.ReloadPolicies(ctx)
}
