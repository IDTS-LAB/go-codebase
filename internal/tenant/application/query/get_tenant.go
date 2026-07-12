package query

import (
	"context"
	"time"

	coreDomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
	"github.com/google/uuid"
)

type GetTenantQuery struct {
	ID uuid.UUID
}

type GetTenantHandler struct {
	repo repository.TenantRepository
}

func NewGetTenantHandler(repo repository.TenantRepository) *GetTenantHandler {
	return &GetTenantHandler{repo: repo}
}

func (h *GetTenantHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(GetTenantQuery)
	tenant, err := h.repo.GetByID(ctx, q.ID)
	if err != nil {
		return nil, coreDomain.NewDomainError(coreDomain.ErrNotFound, "TENANT_NOT_FOUND", "tenant not found")
	}
	return dto.TenantResponse{
		ID:        tenant.ID.String(),
		Name:      tenant.Name,
		Slug:      tenant.Slug,
		Domain:    tenant.Domain,
		Settings:  tenant.Settings,
		IsActive:  tenant.IsActive,
		CreatedAt: tenant.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: tenant.UpdatedAt.Format(time.RFC3339Nano),
	}, nil
}
