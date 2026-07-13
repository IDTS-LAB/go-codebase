package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type GetUserRolesQuery struct {
	UserID uuid.UUID
}

type GetUserRolesHandler struct {
	userRoleRepo repository.UserRoleRepository
}

func NewGetUserRolesHandler(userRoleRepo repository.UserRoleRepository) *GetUserRolesHandler {
	return &GetUserRolesHandler{userRoleRepo: userRoleRepo}
}

func (h *GetUserRolesHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(GetUserRolesQuery)
	return h.userRoleRepo.GetRolesByUserID(ctx, q.UserID)
}
