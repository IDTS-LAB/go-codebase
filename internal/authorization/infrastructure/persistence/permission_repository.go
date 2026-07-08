package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type permissionRepository struct {
	db *sql.DB
}

func NewPermissionRepository(db *sql.DB) repository.PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) Create(ctx context.Context, perm *entity.Permission) error {
	query := `INSERT INTO permissions (id, name, description, resource, action, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, perm.ID, perm.Name, perm.Description, perm.Resource, perm.Action, perm.CreatedAt, perm.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert permission: %w", err)
	}
	return nil
}

func (r *permissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error) {
	query := `SELECT id, name, description, resource, action, created_at, updated_at, deleted_at FROM permissions WHERE id = $1 AND deleted_at IS NULL`
	perm := &entity.Permission{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(&perm.ID, &perm.Name, &perm.Description, &perm.Resource, &perm.Action, &perm.CreatedAt, &perm.UpdatedAt, &perm.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("permission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get permission: %w", err)
	}
	return perm, nil
}

func (r *permissionRepository) GetByName(ctx context.Context, name string) (*entity.Permission, error) {
	query := `SELECT id, name, description, resource, action, created_at, updated_at, deleted_at FROM permissions WHERE name = $1 AND deleted_at IS NULL`
	perm := &entity.Permission{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(&perm.ID, &perm.Name, &perm.Description, &perm.Resource, &perm.Action, &perm.CreatedAt, &perm.UpdatedAt, &perm.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("permission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get permission by name: %w", err)
	}
	return perm, nil
}

func (r *permissionRepository) GetAll(ctx context.Context, offset, limit int) ([]*entity.Permission, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM permissions WHERE deleted_at IS NULL`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count permissions: %w", err)
	}

	query := `SELECT id, name, description, resource, action, created_at, updated_at, deleted_at FROM permissions WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query permissions: %w", err)
	}
	defer rows.Close()

	var perms []*entity.Permission
	for rows.Next() {
		perm := &entity.Permission{}
		if err := rows.Scan(&perm.ID, &perm.Name, &perm.Description, &perm.Resource, &perm.Action, &perm.CreatedAt, &perm.UpdatedAt, &perm.DeletedAt); err != nil {
			return nil, 0, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, perm)
	}
	return perms, total, nil
}

func (r *permissionRepository) Update(ctx context.Context, perm *entity.Permission) error {
	query := `UPDATE permissions SET name = $2, description = $3, resource = $4, action = $5, updated_at = $6 WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, perm.ID, perm.Name, perm.Description, perm.Resource, perm.Action, perm.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update permission: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("permission not found")
	}
	return nil
}

func (r *permissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE permissions SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete permission: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("permission not found")
	}
	return nil
}
