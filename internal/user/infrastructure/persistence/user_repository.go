package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/user/infrastructure/persistence/sqlc"
	"github.com/google/uuid"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) List(ctx context.Context, offset, limit int) ([]*entity.User, int, error) {
	q := sqlc.New(r.db)

	total, err := q.CountUsers(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	rows, err := q.ListUsers(ctx, sqlc.ListUsersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	users := make([]*entity.User, len(rows))
	for i, row := range rows {
		users[i] = mapSqlcUserToEntityForAdmin(sqlc.GetUserByIDRow(row))
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
	return mapSqlcUserToEntityForAdmin(row), nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	q := sqlc.New(r.db)
	rows, err := q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		IsActive:  user.IsActive,
		UpdatedAt: user.UpdatedAt,
		DeletedAt: ptrToNullTime(user.DeletedAt),
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

func mapSqlcUserToEntityForAdmin(row sqlc.GetUserByIDRow) *entity.User {
	return &entity.User{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: nullTimeToPtr(row.DeletedAt),
		},
		Email:    row.Email,
		Name:     row.Name,
		IsActive: row.IsActive,
	}
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

func ptrToNullTime(t *time.Time) sql.NullTime {
	if t != nil {
		return sql.NullTime{Time: *t, Valid: true}
	}
	return sql.NullTime{}
}
