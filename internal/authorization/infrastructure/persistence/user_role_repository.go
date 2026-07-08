package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type userRoleRepository struct {
	db *sql.DB
}

func NewUserRoleRepository(db *sql.DB) repository.UserRoleRepository {
	return &userRoleRepository{db: db}
}

func (r *userRoleRepository) Assign(ctx context.Context, ur entity.UserRole) error {
	query := `INSERT INTO user_roles (user_id, role_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT (user_id, role_id) DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, ur.UserID, ur.RoleID)
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	return nil
}

func (r *userRoleRepository) Remove(ctx context.Context, userID, roleID uuid.UUID) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}
	return nil
}

func (r *userRoleRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]entity.UserRole, error) {
	query := `SELECT user_id, role_id FROM user_roles WHERE user_id = $1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	defer rows.Close()

	var urs []entity.UserRole
	for rows.Next() {
		var ur entity.UserRole
		if err := rows.Scan(&ur.UserID, &ur.RoleID); err != nil {
			return nil, fmt.Errorf("scan user role: %w", err)
		}
		urs = append(urs, ur)
	}
	return urs, nil
}

func (r *userRoleRepository) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at, r.deleted_at
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.deleted_at IS NULL
		ORDER BY r.created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get roles by user: %w", err)
	}
	defer rows.Close()

	var roles []*entity.Role
	for rows.Next() {
		role := &entity.Role{}
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt, &role.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}
