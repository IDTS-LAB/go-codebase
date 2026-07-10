package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/infrastructure/persistence/sqlc"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	q := sqlc.New(r.db)
	err := q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:                  user.ID,
		Email:               user.Email,
		Password:            user.Password,
		Name:                user.Name,
		IsActive:            user.IsActive,
		FailedLoginAttempts: int32(user.FailedLoginAttempts),
		LockedUntil:         ptrToNullTime(user.LockedUntil),
		EmailVerified:       sql.NullBool{Bool: user.EmailVerified, Valid: true},
		EmailVerifyToken:    ptrToNullString(user.EmailVerifyToken),
		EmailVerifyExpires:  ptrToNullTime(user.EmailVerifyExpires),
		PasswordResetToken:  ptrToNullString(user.PasswordResetToken),
		PasswordResetExpires: ptrToNullTime(user.PasswordResetExpires),
		CreatedAt:           user.CreatedAt,
		UpdatedAt:           user.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
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
	return mapSqlcUserToEntity(row), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	q := sqlc.New(r.db)
	row, err := q.GetUserByEmail(ctx, email)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return mapSqlcUserToEntity(sqlc.GetUserByIDRow(row)), nil
}

func (r *userRepository) GetByVerifyToken(ctx context.Context, token string) (*entity.User, error) {
	q := sqlc.New(r.db)
	row, err := q.GetUserByVerifyToken(ctx, sql.NullString{String: token, Valid: token != ""})
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by verify token: %w", err)
	}
	return mapSqlcUserToEntity(sqlc.GetUserByIDRow(row)), nil
}

func (r *userRepository) GetByResetToken(ctx context.Context, token string) (*entity.User, error) {
	q := sqlc.New(r.db)
	row, err := q.GetUserByResetToken(ctx, sql.NullString{String: token, Valid: token != ""})
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by reset token: %w", err)
	}
	return mapSqlcUserToEntity(sqlc.GetUserByIDRow(row)), nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	q := sqlc.New(r.db)
	rows, err := q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:                  user.ID,
		Email:               user.Email,
		Password:            user.Password,
		Name:                user.Name,
		IsActive:            user.IsActive,
		UpdatedAt:           user.UpdatedAt,
		FailedLoginAttempts: int32(user.FailedLoginAttempts),
		LockedUntil:         ptrToNullTime(user.LockedUntil),
		EmailVerified:       sql.NullBool{Bool: user.EmailVerified, Valid: true},
		EmailVerifyToken:    ptrToNullString(user.EmailVerifyToken),
		EmailVerifyExpires:  ptrToNullTime(user.EmailVerifyExpires),
		PasswordResetToken:  ptrToNullString(user.PasswordResetToken),
		PasswordResetExpires: ptrToNullTime(user.PasswordResetExpires),
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func mapSqlcUserToEntity(row sqlc.GetUserByIDRow) *entity.User {
	return &entity.User{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: nullTimeToPtr(row.DeletedAt),
		},
		Email:                row.Email,
		Password:             row.Password,
		Name:                 row.Name,
		IsActive:             row.IsActive,
		FailedLoginAttempts:  int(row.FailedLoginAttempts),
		LockedUntil:          nullTimeToPtr(row.LockedUntil),
		EmailVerified:        nullBoolToValue(row.EmailVerified),
		EmailVerifyToken:     nullStringToPtr(row.EmailVerifyToken),
		EmailVerifyExpires:   nullTimeToPtr(row.EmailVerifyExpires),
		PasswordResetToken:   nullStringToPtr(row.PasswordResetToken),
		PasswordResetExpires: nullTimeToPtr(row.PasswordResetExpires),
	}
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func nullBoolToValue(nb sql.NullBool) bool {
	if nb.Valid {
		return nb.Bool
	}
	return false
}

func ptrToNullTime(t *time.Time) sql.NullTime {
	if t != nil {
		return sql.NullTime{Time: *t, Valid: true}
	}
	return sql.NullTime{Valid: false}
}

func ptrToNullString(s *string) sql.NullString {
	if s != nil {
		return sql.NullString{String: *s, Valid: true}
	}
	return sql.NullString{Valid: false}
}
