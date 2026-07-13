package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type UnassignRoleCommand struct {
	UserID uuid.UUID
	RoleID uuid.UUID
}

type UnassignRoleHandler struct {
	userRoleRepo repository.UserRoleRepository
	enforcer     Enforcer
}

func NewUnassignRoleHandler(userRoleRepo repository.UserRoleRepository, enforcer Enforcer) *UnassignRoleHandler {
	return &UnassignRoleHandler{userRoleRepo: userRoleRepo, enforcer: enforcer}
}

func (h *UnassignRoleHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(UnassignRoleCommand)
	if err := h.userRoleRepo.Remove(ctx, c.UserID, c.RoleID); err != nil {
		return nil, err
	}
	return nil, h.enforcer.ReloadUserPolicies(ctx, c.UserID)
}
