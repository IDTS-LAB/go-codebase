package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence/sqlc"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type permissionRepository struct {
	db *sql.DB
}

func NewPermissionRepository(db *sql.DB) repository.PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) Create(ctx context.Context, perm *entity.Permission) error {
	q := sqlc.New(r.db)
	err := q.CreatePermission(ctx, sqlc.CreatePermissionParams{
		ID:          perm.ID,
		Name:        perm.Name,
		Description: perm.Description,
		Resource:    perm.Resource,
		Action:      perm.Action,
		CreatedAt:   perm.CreatedAt,
		UpdatedAt:   perm.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert permission: %w", err)
	}
	return nil
}

func (r *permissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error) {
	q := sqlc.New(r.db)
	row, err := q.GetPermissionByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("permission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get permission: %w", err)
	}
	return mapSqlcPermissionToEntity(row), nil
}

func (r *permissionRepository) GetByName(ctx context.Context, name string) (*entity.Permission, error) {
	q := sqlc.New(r.db)
	row, err := q.GetPermissionByName(ctx, name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("permission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get permission by name: %w", err)
	}
	return mapSqlcPermissionToEntity(row), nil
}

func (r *permissionRepository) GetAll(ctx context.Context, offset, limit int) ([]*entity.Permission, int, error) {
	q := sqlc.New(r.db)

	total, err := q.CountPermissions(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count permissions: %w", err)
	}

	rows, err := q.ListPermissions(ctx, sqlc.ListPermissionsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("query permissions: %w", err)
	}

	perms := make([]*entity.Permission, len(rows))
	for i, row := range rows {
		perms[i] = mapSqlcPermissionToEntity(row)
	}
	return perms, int(total), nil
}

func (r *permissionRepository) Update(ctx context.Context, perm *entity.Permission) error {
	q := sqlc.New(r.db)
	rows, err := q.UpdatePermission(ctx, sqlc.UpdatePermissionParams{
		ID:          perm.ID,
		Name:        perm.Name,
		Description: perm.Description,
		Resource:    perm.Resource,
		Action:      perm.Action,
		UpdatedAt:   perm.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("update permission: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("permission not found")
	}
	return nil
}

func (r *permissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := sqlc.New(r.db)
	rows, err := q.DeletePermission(ctx, id)
	if err != nil {
		return fmt.Errorf("delete permission: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("permission not found")
	}
	return nil
}

func mapSqlcPermissionToEntity(row sqlc.Permission) *entity.Permission {
	return &entity.Permission{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: nullTimeToPtr(row.DeletedAt),
		},
		Name:        row.Name,
		Description: row.Description,
		Resource:    row.Resource,
		Action:      row.Action,
	}
}
