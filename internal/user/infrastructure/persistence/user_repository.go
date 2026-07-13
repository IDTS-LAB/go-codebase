package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/user/infrastructure/persistence/sqlc"
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
	q := sqlc.New(r.db)
	row, err := q.GetUserByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	u := &entity.User{
		Entity:   domain.Entity{ID: row.ID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt},
		Email:    row.Email,
		Name:     row.Name,
		IsActive: row.IsActive,
	}
	if row.DeletedAt.Valid {
		u.DeletedAt = &row.DeletedAt.Time
	}
	return u, nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	q := sqlc.New(r.db)
	rows, err := q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		IsActive:  user.IsActive,
		UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := sqlc.New(r.db)
	rows, err := q.DeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
