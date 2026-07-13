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
	"github.com/IDTS-LAB/go-codebase/internal/shared/cursor"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/google/uuid"
)

type permissionRepository struct {
	db           *sql.DB
	tenantConfig *tenantfilter.Config
}

func NewPermissionRepository(db *sql.DB, tenantConfig *tenantfilter.Config) repository.PermissionRepository {
	return &permissionRepository{db: db, tenantConfig: tenantConfig}
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
	return mapPermissionRowToEntity(row.ID, row.Name, row.Description, row.Resource, row.Action, row.CreatedAt, row.UpdatedAt, row.DeletedAt), nil
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
	return mapPermissionRowToEntity(row.ID, row.Name, row.Description, row.Resource, row.Action, row.CreatedAt, row.UpdatedAt, row.DeletedAt), nil
}

func (r *permissionRepository) GetAll(ctx context.Context, cursorArg *string, limit int) ([]*entity.Permission, *string, *string, bool, bool, error) {
	args := []interface{}{}
	whereClause := "WHERE deleted_at IS NULL"

	if r.tenantConfig != nil && r.tenantConfig.Enabled {
		tenantID := middleware.GetTenantID(ctx)
		if tenantID != "" {
			whereClause += fmt.Sprintf(" AND tenant_id = $%d", len(args)+1)
			args = append(args, tenantID)
		}
	}

	nextPos := len(args) + 1
	if cursorArg != nil {
		c, err := cursor.Decode(*cursorArg)
		if err != nil {
			return nil, nil, nil, false, false, fmt.Errorf("invalid cursor: %w", err)
		}
		whereClause += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", nextPos, nextPos+1)
		args = append(args, c.Timestamp, c.ID)
		nextPos += 2
	}

	dataQuery := fmt.Sprintf("SELECT id, name, description, resource, action, created_at, updated_at, deleted_at FROM permissions %s ORDER BY created_at DESC, id DESC LIMIT $%d", whereClause, nextPos)
	dataArgs := append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, nil, nil, false, false, fmt.Errorf("query permissions: %w", err)
	}
	defer rows.Close()

	var perms []*entity.Permission
	for rows.Next() {
		var p entity.Permission
		var deletedAt sql.NullTime
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Resource, &p.Action, &p.CreatedAt, &p.UpdatedAt, &deletedAt); err != nil {
			return nil, nil, nil, false, false, fmt.Errorf("scan permission: %w", err)
		}
		if deletedAt.Valid {
			p.DeletedAt = &deletedAt.Time
		}
		perms = append(perms, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, false, false, fmt.Errorf("rows iteration: %w", err)
	}

	hasNext := len(perms) > limit
	if hasNext {
		perms = perms[:limit]
	}

	var nextCursor *string
	var prevCursor *string
	if len(perms) > 0 {
		last := perms[len(perms)-1]
		nc := cursor.Encode(last.CreatedAt, last.ID)
		nextCursor = &nc

		first := perms[0]
		pc := cursor.Encode(first.CreatedAt, first.ID)
		prevCursor = &pc
	}

	hasPrev := cursorArg != nil
	if hasPrev && len(perms) == 0 {
		hasPrev = false
	}

	return perms, nextCursor, prevCursor, hasNext, hasPrev, nil
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
	return mapPermissionRowToEntity(row.ID, row.Name, row.Description, row.Resource, row.Action, row.CreatedAt, row.UpdatedAt, row.DeletedAt)
}

func mapPermissionRowToEntity(id uuid.UUID, name, description, resource, action string, createdAt, updatedAt time.Time, deletedAt sql.NullTime) *entity.Permission {
	return &entity.Permission{
		Entity: domain.Entity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			DeletedAt: nullTimeToPtr(deletedAt),
		},
		Name:        name,
		Description: description,
		Resource:    resource,
		Action:      action,
	}
}
