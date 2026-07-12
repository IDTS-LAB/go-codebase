package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/google/uuid"
)

type userRepository struct {
	db           *sql.DB
	tenantConfig *tenantfilter.Config
}

func NewUserRepository(db *sql.DB, tenantConfig *tenantfilter.Config) repository.UserRepository {
	return &userRepository{db: db, tenantConfig: tenantConfig}
}

func (r *userRepository) List(ctx context.Context, offset, limit int) ([]*entity.User, int, error) {
	var args []interface{}
	countQuery := "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL"
	dataQuery := "SELECT u.id, u.email, u.name, u.is_active, u.created_at, u.updated_at, u.deleted_at FROM users u WHERE u.deleted_at IS NULL"

	if r.tenantConfig != nil && r.tenantConfig.Enabled {
		tenantID := middleware.GetTenantID(ctx)
		if tenantID != "" {
			countQuery += " AND u.tenant_id = $1"
			dataQuery += " AND u.tenant_id = $1"
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
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	if len(args) > 0 {
		dataQuery += " ORDER BY u.created_at DESC LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	} else {
		dataQuery += " ORDER BY u.created_at DESC LIMIT $1 OFFSET $2"
		args = append(args, limit, offset)
	}

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var u entity.User
		var deletedAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &deletedAt); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		if deletedAt.Valid {
			u.DeletedAt = &deletedAt.Time
		}
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	return users, int(total), nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var u entity.User
	var deletedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.name, u.is_active, u.created_at, u.updated_at, u.deleted_at
		FROM users u WHERE u.id = $1 AND u.deleted_at IS NULL
	`, id).Scan(&u.ID, &u.Email, &u.Name, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &deletedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if deletedAt.Valid {
		u.DeletedAt = &deletedAt.Time
	}
	return &u, nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE users SET email = $2, name = $3, is_active = $4, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL
	`, user.ID, user.Email, user.Name, user.IsActive)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update user rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete user rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
