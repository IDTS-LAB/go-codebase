package query

import (
	"context"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
)

type ListTenantsQuery struct {
	Page    int
	PerPage int
}

type ListTenantsHandler struct {
	repo repository.TenantRepository
}

func NewListTenantsHandler(repo repository.TenantRepository) *ListTenantsHandler {
	return &ListTenantsHandler{repo: repo}
}

func (h *ListTenantsHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(ListTenantsQuery)
	offset := (q.Page - 1) * q.PerPage
	tenants, total, err := h.repo.List(ctx, offset, q.PerPage)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.TenantResponse, len(tenants))
	for i, t := range tenants {
		responses[i] = dto.TenantResponse{
			ID:        t.ID.String(),
			Name:      t.Name,
			Slug:      t.Slug,
			Domain:    t.Domain,
			Settings:  t.Settings,
			IsActive:  t.IsActive,
			CreatedAt: t.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt: t.UpdatedAt.Format(time.RFC3339Nano),
		}
	}

	return dto.TenantListResponse{Tenants: responses, Total: total}, nil
}
