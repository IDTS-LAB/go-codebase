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
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/google/uuid"
)

type roleRepository struct {
	db           *sql.DB
	tenantConfig *tenantfilter.Config
}

func NewRoleRepository(db *sql.DB, tenantConfig *tenantfilter.Config) repository.RoleRepository {
	return &roleRepository{db: db, tenantConfig: tenantConfig}
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
	return mapRoleRowToEntity(row.ID, row.Name, row.Description, row.CreatedAt, row.UpdatedAt, row.DeletedAt), nil
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
	return mapRoleRowToEntity(row.ID, row.Name, row.Description, row.CreatedAt, row.UpdatedAt, row.DeletedAt), nil
}

func (r *roleRepository) GetAll(ctx context.Context, offset, limit int) ([]*entity.Role, int, error) {
	var args []interface{}
	countQuery := "SELECT COUNT(*) FROM roles WHERE deleted_at IS NULL"
	dataQuery := "SELECT id, name, description, created_at, updated_at, deleted_at FROM roles WHERE deleted_at IS NULL"

	if r.tenantConfig != nil && r.tenantConfig.Enabled {
		tenantID := middleware.GetTenantID(ctx)
		if tenantID != "" {
			countQuery += " AND tenant_id = $1"
			dataQuery += " AND tenant_id = $1"
			args = append(args, tenantID)
		}
	}

	var total int64
	var err error
	if len(args) > 0 {
		err = r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	} else {
		err = r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("count roles: %w", err)
	}

	if len(args) > 0 {
		dataQuery += " ORDER BY created_at DESC LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	} else {
		dataQuery += " ORDER BY created_at DESC LIMIT $1 OFFSET $2"
		args = append(args, limit, offset)
	}

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()

	var roles []*entity.Role
	for rows.Next() {
		var rl entity.Role
		var deletedAt sql.NullTime
		if err := rows.Scan(&rl.ID, &rl.Name, &rl.Description, &rl.CreatedAt, &rl.UpdatedAt, &deletedAt); err != nil {
			return nil, 0, fmt.Errorf("scan role: %w", err)
		}
		if deletedAt.Valid {
			rl.DeletedAt = &deletedAt.Time
		}
		roles = append(roles, &rl)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
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

func mapRoleRowToEntity(id uuid.UUID, name, description string, createdAt, updatedAt time.Time, deletedAt sql.NullTime) *entity.Role {
	return &entity.Role{
		Entity: domain.Entity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			DeletedAt: nullTimeToPtr(deletedAt),
		},
		Name:        name,
		Description: description,
	}
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}
