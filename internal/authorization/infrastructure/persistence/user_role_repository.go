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

type userRoleRepository struct {
	db *sql.DB
}

func NewUserRoleRepository(db *sql.DB) repository.UserRoleRepository {
	return &userRoleRepository{db: db}
}

func (r *userRoleRepository) Assign(ctx context.Context, ur entity.UserRole) error {
	q := sqlc.New(r.db)
	err := q.AssignUserRole(ctx, sqlc.AssignUserRoleParams{
		UserID: ur.UserID,
		RoleID: ur.RoleID,
	})
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	return nil
}

func (r *userRoleRepository) Remove(ctx context.Context, userID, roleID uuid.UUID) error {
	q := sqlc.New(r.db)
	err := q.RemoveUserRole(ctx, sqlc.RemoveUserRoleParams{
		UserID: userID,
		RoleID: roleID,
	})
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}
	return nil
}

func (r *userRoleRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]entity.UserRole, error) {
	q := sqlc.New(r.db)
	rows, err := q.GetUserRolesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	urs := make([]entity.UserRole, len(rows))
	for i, row := range rows {
		urs[i] = entity.UserRole{
			UserID: row.UserID,
			RoleID: row.RoleID,
		}
	}
	return urs, nil
}

func (r *userRoleRepository) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error) {
	q := sqlc.New(r.db)
	rows, err := q.GetRolesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get roles by user: %w", err)
	}
	roles := make([]*entity.Role, len(rows))
	for i, row := range rows {
		roles[i] = mapSqlcRoleToEntity(row)
	}
	return roles, nil
}
