package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
)

type ListTenantsQuery struct {
	Cursor *string
	Limit  int
}

type ListTenantsHandler struct {
	repo repository.TenantRepository
}

func NewListTenantsHandler(repo repository.TenantRepository) *ListTenantsHandler {
	return &ListTenantsHandler{repo: repo}
}

func (h *ListTenantsHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(ListTenantsQuery)
	tenants, nextCursor, prevCursor, hasNext, hasPrev, err := h.repo.List(ctx, q.Cursor, q.Limit)
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
			CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return dto.TenantListResponse{
		Tenants:    responses,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Limit:      q.Limit,
	}, nil
}
