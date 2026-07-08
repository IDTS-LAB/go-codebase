package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type rolePermissionRepository struct {
	db *sql.DB
}

func NewRolePermissionRepository(db *sql.DB) repository.RolePermissionRepository {
	return &rolePermissionRepository{db: db}
}

func (r *rolePermissionRepository) Assign(ctx context.Context, rp entity.RolePermission) error {
	query := `INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT (role_id, permission_id) DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, rp.RoleID, rp.PermissionID)
	if err != nil {
		return fmt.Errorf("assign permission: %w", err)
	}
	return nil
}

func (r *rolePermissionRepository) Remove(ctx context.Context, roleID, permissionID uuid.UUID) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err := r.db.ExecContext(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("remove permission: %w", err)
	}
	return nil
}

func (r *rolePermissionRepository) GetByRoleID(ctx context.Context, roleID uuid.UUID) ([]entity.RolePermission, error) {
	query := `SELECT role_id, permission_id FROM role_permissions WHERE role_id = $1`
	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("get role permissions: %w", err)
	}
	defer rows.Close()

	var rps []entity.RolePermission
	for rows.Next() {
		var rp entity.RolePermission
		if err := rows.Scan(&rp.RoleID, &rp.PermissionID); err != nil {
			return nil, fmt.Errorf("scan role permission: %w", err)
		}
		rps = append(rps, rp)
	}
	return rps, nil
}

func (r *rolePermissionRepository) GetPermissionsByRoleID(ctx context.Context, roleID uuid.UUID) ([]*entity.Permission, error) {
	query := `
		SELECT p.id, p.name, p.description, p.resource, p.action, p.created_at, p.updated_at, p.deleted_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1 AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("get permissions by role: %w", err)
	}
	defer rows.Close()

	var perms []*entity.Permission
	for rows.Next() {
		perm := &entity.Permission{}
		if err := rows.Scan(&perm.ID, &perm.Name, &perm.Description, &perm.Resource, &perm.Action, &perm.CreatedAt, &perm.UpdatedAt, &perm.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, perm)
	}
	return perms, nil
}
