package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type UpdateRoleCommand struct {
	ID          uuid.UUID
	Name        string
	Description string
}

type UpdateRoleHandler struct {
	roleRepo repository.RoleRepository
}

func NewUpdateRoleHandler(roleRepo repository.RoleRepository) *UpdateRoleHandler {
	return &UpdateRoleHandler{roleRepo: roleRepo}
}

func (h *UpdateRoleHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(UpdateRoleCommand)
	role, err := h.roleRepo.GetByID(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	if c.Name != "" {
		role.Name = c.Name
	}
	if c.Description != "" {
		role.Description = c.Description
	}
	role.Touch()
	if err := h.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}
