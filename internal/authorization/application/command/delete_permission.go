package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type DeletePermissionCommand struct {
	ID uuid.UUID
}

type DeletePermissionHandler struct {
	permRepo repository.PermissionRepository
}

func NewDeletePermissionHandler(permRepo repository.PermissionRepository) *DeletePermissionHandler {
	return &DeletePermissionHandler{permRepo: permRepo}
}

func (h *DeletePermissionHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(DeletePermissionCommand)
	return nil, h.permRepo.Delete(ctx, c.ID)
}
