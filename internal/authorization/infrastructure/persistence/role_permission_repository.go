package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence/sqlc"
	"github.com/google/uuid"
)

type rolePermissionRepository struct {
	db *sql.DB
}

func NewRolePermissionRepository(db *sql.DB) repository.RolePermissionRepository {
	return &rolePermissionRepository{db: db}
}

func (r *rolePermissionRepository) Assign(ctx context.Context, rp entity.RolePermission) error {
	q := sqlc.New(r.db)
	err := q.AssignRolePermission(ctx, sqlc.AssignRolePermissionParams{
		RoleID:       rp.RoleID,
		PermissionID: rp.PermissionID,
	})
	if err != nil {
		return fmt.Errorf("assign permission: %w", err)
	}
	return nil
}

func (r *rolePermissionRepository) Remove(ctx context.Context, roleID, permissionID uuid.UUID) error {
	q := sqlc.New(r.db)
	err := q.RemoveRolePermission(ctx, sqlc.RemoveRolePermissionParams{
		RoleID:       roleID,
		PermissionID: permissionID,
	})
	if err != nil {
		return fmt.Errorf("remove permission: %w", err)
	}
	return nil
}

func (r *rolePermissionRepository) GetByRoleID(ctx context.Context, roleID uuid.UUID) ([]entity.RolePermission, error) {
	q := sqlc.New(r.db)
	rows, err := q.GetRolePermissionsByRoleID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("get role permissions: %w", err)
	}
	rps := make([]entity.RolePermission, len(rows))
	for i, row := range rows {
		rps[i] = entity.RolePermission{
			RoleID:       row.RoleID,
			PermissionID: row.PermissionID,
		}
	}
	return rps, nil
}

func (r *rolePermissionRepository) GetPermissionsByRoleID(ctx context.Context, roleID uuid.UUID) ([]*entity.Permission, error) {
	q := sqlc.New(r.db)
	rows, err := q.GetPermissionsByRoleID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("get permissions by role: %w", err)
	}
	perms := make([]*entity.Permission, len(rows))
	for i, row := range rows {
		perms[i] = mapSqlcPermissionToEntity(row)
	}
	return perms, nil
}
