package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type UpdatePermissionCommand struct {
	ID          uuid.UUID
	Name        string
	Description string
	Resource    string
	Action      string
}

type UpdatePermissionHandler struct {
	permRepo repository.PermissionRepository
}

func NewUpdatePermissionHandler(permRepo repository.PermissionRepository) *UpdatePermissionHandler {
	return &UpdatePermissionHandler{permRepo: permRepo}
}

func (h *UpdatePermissionHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(UpdatePermissionCommand)
	perm, err := h.permRepo.GetByID(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	if c.Name != "" {
		perm.Name = c.Name
	}
	if c.Description != "" {
		perm.Description = c.Description
	}
	if c.Resource != "" {
		perm.Resource = c.Resource
	}
	if c.Action != "" {
		perm.Action = c.Action
	}
	perm.Touch()
	if err := h.permRepo.Update(ctx, perm); err != nil {
		return nil, err
	}
	return perm, nil
}
