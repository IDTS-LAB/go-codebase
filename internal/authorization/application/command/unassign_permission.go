package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type UnassignPermissionCommand struct {
	RoleID       uuid.UUID
	PermissionID uuid.UUID
}

type UnassignPermissionHandler struct {
	rolePermRepo repository.RolePermissionRepository
	enforcer     Enforcer
}

func NewUnassignPermissionHandler(rolePermRepo repository.RolePermissionRepository, enforcer Enforcer) *UnassignPermissionHandler {
	return &UnassignPermissionHandler{rolePermRepo: rolePermRepo, enforcer: enforcer}
}

func (h *UnassignPermissionHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(UnassignPermissionCommand)
	if err := h.rolePermRepo.Remove(ctx, c.RoleID, c.PermissionID); err != nil {
		return nil, err
	}
	return nil, h.enforcer.ReloadPolicies(ctx)
}
