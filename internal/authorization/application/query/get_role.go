package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type GetRoleQuery struct {
	ID uuid.UUID
}

type GetRoleHandler struct {
	roleRepo repository.RoleRepository
}

func NewGetRoleHandler(roleRepo repository.RoleRepository) *GetRoleHandler {
	return &GetRoleHandler{roleRepo: roleRepo}
}

func (h *GetRoleHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(GetRoleQuery)
	return h.roleRepo.GetByID(ctx, q.ID)
}
