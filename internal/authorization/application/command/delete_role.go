package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type DeleteRoleCommand struct {
	ID uuid.UUID
}

type DeleteRoleHandler struct {
	roleRepo repository.RoleRepository
}

func NewDeleteRoleHandler(roleRepo repository.RoleRepository) *DeleteRoleHandler {
	return &DeleteRoleHandler{roleRepo: roleRepo}
}

func (h *DeleteRoleHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(DeleteRoleCommand)
	return nil, h.roleRepo.Delete(ctx, c.ID)
}
