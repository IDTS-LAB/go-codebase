package command

import (
	"context"
	"encoding/json"
	"time"

	coreDomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
	"github.com/google/uuid"
)

type UpdateTenantCommand struct {
	ID       uuid.UUID
	Name     *string
	Domain   *string
	Settings json.RawMessage
	IsActive *bool
}

type UpdateTenantHandler struct {
	repo repository.TenantRepository
}

func NewUpdateTenantHandler(repo repository.TenantRepository) *UpdateTenantHandler {
	return &UpdateTenantHandler{repo: repo}
}

func (h *UpdateTenantHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(UpdateTenantCommand)
	tenant, err := h.repo.GetByID(ctx, c.ID)
	if err != nil {
		return nil, coreDomain.NewDomainError(coreDomain.ErrNotFound, "TENANT_NOT_FOUND", "tenant not found")
	}

	if c.Name != nil {
		tenant.Name = *c.Name
	}
	if c.Domain != nil {
		tenant.Domain = c.Domain
	}
	if c.Settings != nil {
		tenant.Settings = c.Settings
	}
	if c.IsActive != nil {
		tenant.IsActive = *c.IsActive
	}
	tenant.UpdatedAt = time.Now()

	if err := h.repo.Update(ctx, tenant); err != nil {
		return nil, err
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
