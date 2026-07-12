package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
	"github.com/google/uuid"
)

type tenantRepository struct {
	db *sql.DB
}

func NewTenantRepository(db *sql.DB) repository.TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) Create(ctx context.Context, t *entity.Tenant) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tenants (id, name, slug, domain, settings, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.Name, t.Slug, t.Domain, t.Settings, t.IsActive, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert tenant: %w", err)
	}
	return nil
}

func (r *tenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	var t entity.Tenant
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, domain, settings, is_active, created_at, updated_at
		FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Domain, &t.Settings, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	return &t, nil
}

func (r *tenantRepository) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	var t entity.Tenant
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, domain, settings, is_active, created_at, updated_at
		FROM tenants WHERE slug = $1`, slug,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Domain, &t.Settings, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by slug: %w", err)
	}
	return &t, nil
}

func (r *tenantRepository) List(ctx context.Context, offset, limit int) ([]entity.Tenant, int, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, slug, domain, settings, is_active, created_at, updated_at
		FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []entity.Tenant
	for rows.Next() {
		var t entity.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Domain, &t.Settings, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}
	return tenants, int(total), nil
}

func (r *tenantRepository) Update(ctx context.Context, t *entity.Tenant) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET name = $2, domain = $3, settings = $4, is_active = $5, updated_at = $6 WHERE id = $1`,
		t.ID, t.Name, t.Domain, t.Settings, t.IsActive, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update tenant: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}

func (r *tenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}
