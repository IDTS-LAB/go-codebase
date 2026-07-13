package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/shared/cursor"
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

func (r *tenantRepository) List(ctx context.Context, cursorArg *string, limit int) ([]entity.Tenant, *string, *string, bool, bool, error) {
	args := []interface{}{}
	nextPos := 1
	query := "SELECT id, name, slug, domain, settings, is_active, created_at, updated_at FROM tenants"

	if cursorArg != nil {
		c, err := cursor.Decode(*cursorArg)
		if err != nil {
			return nil, nil, nil, false, false, fmt.Errorf("invalid cursor: %w", err)
		}
		query += fmt.Sprintf(" WHERE (created_at, id) < ($%d, $%d)", nextPos, nextPos+1)
		args = append(args, c.Timestamp, c.ID)
		nextPos += 2
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", nextPos)
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, nil, false, false, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []entity.Tenant
	for rows.Next() {
		var t entity.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Domain, &t.Settings, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, nil, nil, false, false, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, false, false, fmt.Errorf("rows iteration: %w", err)
	}

	hasNext := len(tenants) > limit
	if hasNext {
		tenants = tenants[:limit]
	}

	var nextCursor *string
	var prevCursor *string
	if len(tenants) > 0 {
		last := tenants[len(tenants)-1]
		nc := cursor.Encode(last.CreatedAt, last.ID)
		nextCursor = &nc

		first := tenants[0]
		pc := cursor.Encode(first.CreatedAt, first.ID)
		prevCursor = &pc
	}

	hasPrev := cursorArg != nil
	if hasPrev && len(tenants) == 0 {
		hasPrev = false
	}

	return tenants, nextCursor, prevCursor, hasNext, hasPrev, nil
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
