package command

import (
	"context"
	"encoding/json"
	"time"

	coreDomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
	"github.com/google/uuid"
)

type CreateTenantCommand struct {
	Name     string
	Slug     string
	Domain   *string
	Settings json.RawMessage
}

type CreateTenantHandler struct {
	repo repository.TenantRepository
}

func NewCreateTenantHandler(repo repository.TenantRepository) *CreateTenantHandler {
	return &CreateTenantHandler{repo: repo}
}

func (h *CreateTenantHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(CreateTenantCommand)
	existing, _ := h.repo.GetBySlug(ctx, c.Slug)
	if existing != nil {
		return nil, coreDomain.NewDomainError(coreDomain.ErrAlreadyExists, "TENANT_EXISTS", "tenant with this slug already exists")
	}

	now := time.Now()
	settings := c.Settings
	if settings == nil {
		settings = json.RawMessage("{}")
	}

	tenant := &entity.Tenant{
		ID:        uuid.New(),
		Name:      c.Name,
		Slug:      c.Slug,
		Domain:    c.Domain,
		Settings:  settings,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.repo.Create(ctx, tenant); err != nil {
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
