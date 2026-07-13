package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cursor"
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

func (r *userRepository) List(ctx context.Context, cursorArg *string, limit int) ([]*entity.User, *string, *string, bool, bool, error) {
	args := []interface{}{}
	whereClause := "WHERE u.deleted_at IS NULL"

	if r.tenantConfig != nil && r.tenantConfig.Enabled {
		tenantID := middleware.GetTenantID(ctx)
		if tenantID != "" {
			whereClause += fmt.Sprintf(" AND u.tenant_id = $%d", len(args)+1)
			args = append(args, tenantID)
		}
	}

	nextPos := len(args) + 1
	if cursorArg != nil {
		c, err := cursor.Decode(*cursorArg)
		if err != nil {
			return nil, nil, nil, false, false, fmt.Errorf("invalid cursor: %w", err)
		}
		whereClause += fmt.Sprintf(" AND (u.created_at, u.id) < ($%d, $%d)", nextPos, nextPos+1)
		args = append(args, c.Timestamp, c.ID)
		nextPos += 2
	}

	dataQuery := fmt.Sprintf("SELECT u.id, u.email, u.name, u.is_active, u.created_at, u.updated_at, u.deleted_at FROM users u %s ORDER BY u.created_at DESC, u.id DESC LIMIT $%d", whereClause, nextPos)
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, nil, nil, false, false, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var u entity.User
		var deletedAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &deletedAt); err != nil {
			return nil, nil, nil, false, false, fmt.Errorf("scan user: %w", err)
		}
		if deletedAt.Valid {
			u.DeletedAt = &deletedAt.Time
		}
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, false, false, fmt.Errorf("rows iteration: %w", err)
	}

	hasNext := len(users) > limit
	if hasNext {
		users = users[:limit]
	}

	var nextCursor *string
	var prevCursor *string
	if len(users) > 0 {
		last := users[len(users)-1]
		nc := cursor.Encode(last.CreatedAt, last.ID)
		nextCursor = &nc

		first := users[0]
		pc := cursor.Encode(first.CreatedAt, first.ID)
		prevCursor = &pc
	}

	hasPrev := cursorArg != nil
	if hasPrev && len(users) == 0 {
		hasPrev = false
	}

	return users, nextCursor, prevCursor, hasNext, hasPrev, nil
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
