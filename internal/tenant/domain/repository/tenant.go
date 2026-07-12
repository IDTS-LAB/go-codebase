package repository

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/entity"
	"github.com/google/uuid"
)

type TenantRepository interface {
	Create(ctx context.Context, t *entity.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error)
	List(ctx context.Context, offset, limit int) ([]entity.Tenant, int, error)
	Update(ctx context.Context, t *entity.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
}
