package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/infrastructure/persistence/sqlc"
	"github.com/google/uuid"
)

type tenantRepository struct {
	db *sql.DB
}

func NewTenantRepository(db *sql.DB) repository.TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) Create(ctx context.Context, t *entity.Tenant) error {
	q := sqlc.New(r.db)
	err := q.CreateTenant(ctx, sqlc.CreateTenantParams{
		ID:        t.ID,
		Name:      t.Name,
		Slug:      t.Slug,
		Domain:    ptrToNullString(t.Domain),
		Settings:  t.Settings,
		IsActive:  t.IsActive,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert tenant: %w", err)
	}
	return nil
}

func (r *tenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	q := sqlc.New(r.db)
	row, err := q.GetTenantByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	t := mapTenantToEntity(row)
	return &t, nil
}

func (r *tenantRepository) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	q := sqlc.New(r.db)
	row, err := q.GetTenantBySlug(ctx, slug)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by slug: %w", err)
	}
	t := mapTenantToEntity(row)
	return &t, nil
}

func (r *tenantRepository) List(ctx context.Context, offset, limit int) ([]entity.Tenant, int, error) {
	q := sqlc.New(r.db)

	total, err := q.CountTenants(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	rows, err := q.ListTenants(ctx, sqlc.ListTenantsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}

	tenants := make([]entity.Tenant, len(rows))
	for i, row := range rows {
		tenants[i] = mapTenantToEntity(row)
	}
	return tenants, int(total), nil
}

func (r *tenantRepository) Update(ctx context.Context, t *entity.Tenant) error {
	q := sqlc.New(r.db)
	rows, err := q.UpdateTenant(ctx, sqlc.UpdateTenantParams{
		ID:        t.ID,
		Name:      t.Name,
		Domain:    ptrToNullString(t.Domain),
		Settings:  t.Settings,
		IsActive:  t.IsActive,
		UpdatedAt: t.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("update tenant: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}

func (r *tenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := sqlc.New(r.db)
	rows, err := q.DeleteTenant(ctx, id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}

func mapTenantToEntity(row sqlc.Tenant) entity.Tenant {
	return entity.Tenant{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		Domain:    nullStringToPtr(row.Domain),
		Settings:  row.Settings,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func ptrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}
