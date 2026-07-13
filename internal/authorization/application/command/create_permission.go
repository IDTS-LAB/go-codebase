package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	coredomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
)

type CreatePermissionCommand struct {
	Name        string
	Description string
	Resource    string
	Action      string
}

type CreatePermissionHandler struct {
	permRepo repository.PermissionRepository
}

func NewCreatePermissionHandler(permRepo repository.PermissionRepository) *CreatePermissionHandler {
	return &CreatePermissionHandler{permRepo: permRepo}
}

func (h *CreatePermissionHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(CreatePermissionCommand)
	existing, _ := h.permRepo.GetByName(ctx, c.Name)
	if existing != nil {
		return nil, coredomain.ErrConflict
	}
	perm := entity.NewPermission(c.Name, c.Description, c.Resource, c.Action)
	if err := h.permRepo.Create(ctx, perm); err != nil {
		return nil, err
	}
	return perm, nil
}
