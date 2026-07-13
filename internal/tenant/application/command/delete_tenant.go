package command

import (
	"context"

	coreDomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
	"github.com/google/uuid"
)

type DeleteTenantCommand struct {
	ID uuid.UUID
}

type DeleteTenantHandler struct {
	repo repository.TenantRepository
}

func NewDeleteTenantHandler(repo repository.TenantRepository) *DeleteTenantHandler {
	return &DeleteTenantHandler{repo: repo}
}

func (h *DeleteTenantHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(DeleteTenantCommand)
	_, err := h.repo.GetByID(ctx, c.ID)
	if err != nil {
		return nil, coreDomain.NewDomainError(coreDomain.ErrNotFound, "TENANT_NOT_FOUND", "tenant not found")
	}
	return nil, h.repo.Delete(ctx, c.ID)
}
