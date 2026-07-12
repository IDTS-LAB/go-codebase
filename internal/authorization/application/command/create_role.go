package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	coredomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
)

type CreateRoleCommand struct {
	Name        string
	Description string
}

type CreateRoleHandler struct {
	roleRepo repository.RoleRepository
}

func NewCreateRoleHandler(roleRepo repository.RoleRepository) *CreateRoleHandler {
	return &CreateRoleHandler{roleRepo: roleRepo}
}

func (h *CreateRoleHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(CreateRoleCommand)
	existing, _ := h.roleRepo.GetByName(ctx, c.Name)
	if existing != nil {
		return nil, coredomain.ErrConflict
	}
	role := entity.NewRole(c.Name, c.Description)
	if err := h.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}
