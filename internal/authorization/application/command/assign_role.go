package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type Enforcer interface {
	ReloadPolicies(ctx context.Context) error
	ReloadUserPolicies(ctx context.Context, userID uuid.UUID) error
	Enforce(userID uuid.UUID, resource, action string) (bool, error)
}

type AssignRoleCommand struct {
	UserID uuid.UUID
	RoleID uuid.UUID
}

type AssignRoleHandler struct {
	roleRepo     repository.RoleRepository
	userRoleRepo repository.UserRoleRepository
	enforcer     Enforcer
}

func NewAssignRoleHandler(roleRepo repository.RoleRepository, userRoleRepo repository.UserRoleRepository, enforcer Enforcer) *AssignRoleHandler {
	return &AssignRoleHandler{roleRepo: roleRepo, userRoleRepo: userRoleRepo, enforcer: enforcer}
}

func (h *AssignRoleHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(AssignRoleCommand)
	_, err := h.roleRepo.GetByID(ctx, c.RoleID)
	if err != nil {
		return nil, err
	}
	if err := h.userRoleRepo.Assign(ctx, entity.NewUserRole(c.UserID, c.RoleID)); err != nil {
		return nil, err
	}
	return nil, h.enforcer.ReloadUserPolicies(ctx, c.UserID)
}
