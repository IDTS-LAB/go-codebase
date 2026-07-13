package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type GetPermissionQuery struct {
	ID uuid.UUID
}

type GetPermissionHandler struct {
	permRepo repository.PermissionRepository
}

func NewGetPermissionHandler(permRepo repository.PermissionRepository) *GetPermissionHandler {
	return &GetPermissionHandler{permRepo: permRepo}
}

func (h *GetPermissionHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(GetPermissionQuery)
	return h.permRepo.GetByID(ctx, q.ID)
}
