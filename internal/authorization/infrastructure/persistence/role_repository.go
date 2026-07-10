package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence/sqlc"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type roleRepository struct {
	db *sql.DB
}

func NewRoleRepository(db *sql.DB) repository.RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Create(ctx context.Context, role *entity.Role) error {
	q := sqlc.New(r.db)
	err := q.CreateRole(ctx, sqlc.CreateRoleParams{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert role: %w", err)
	}
	return nil
}

func (r *roleRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Role, error) {
	q := sqlc.New(r.db)
	row, err := q.GetRoleByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}
	return mapSqlcRoleToEntity(row), nil
}

func (r *roleRepository) GetByName(ctx context.Context, name string) (*entity.Role, error) {
	q := sqlc.New(r.db)
	row, err := q.GetRoleByName(ctx, name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get role by name: %w", err)
	}
	return mapSqlcRoleToEntity(row), nil
}

func (r *roleRepository) GetAll(ctx context.Context, offset, limit int) ([]*entity.Role, int, error) {
	q := sqlc.New(r.db)

	total, err := q.CountRoles(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count roles: %w", err)
	}

	rows, err := q.ListRoles(ctx, sqlc.ListRolesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("query roles: %w", err)
	}

	roles := make([]*entity.Role, len(rows))
	for i, row := range rows {
		roles[i] = mapSqlcRoleToEntity(row)
	}
	return roles, int(total), nil
}

func (r *roleRepository) Update(ctx context.Context, role *entity.Role) error {
	q := sqlc.New(r.db)
	rows, err := q.UpdateRole(ctx, sqlc.UpdateRoleParams{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		UpdatedAt:   role.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("role not found")
	}
	return nil
}

func (r *roleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := sqlc.New(r.db)
	rows, err := q.DeleteRole(ctx, id)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("role not found")
	}
	return nil
}

func mapSqlcRoleToEntity(row sqlc.Role) *entity.Role {
	return &entity.Role{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: nullTimeToPtr(row.DeletedAt),
		},
		Name:        row.Name,
		Description: row.Description,
	}
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}
