package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/google/uuid"
)

const userSelectColumns = "id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at, deleted_at"

func scanUser(row interface{ Scan(...interface{}) error }) (*entity.User, error) {
	user := &entity.User{}
	err := row.Scan(
		&user.ID, &user.Email, &user.Password, &user.Name, &user.IsActive,
		&user.FailedLoginAttempts, &user.LockedUntil,
		&user.EmailVerified, &user.EmailVerifyToken, &user.EmailVerifyExpires,
		&user.PasswordResetToken, &user.PasswordResetExpires,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	return user, err
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	query := `INSERT INTO users (id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.Password, user.Name, user.IsActive,
		user.FailedLoginAttempts, user.LockedUntil,
		user.EmailVerified, user.EmailVerifyToken, user.EmailVerifyExpires,
		user.PasswordResetToken, user.PasswordResetExpires,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE id = $1 AND deleted_at IS NULL`
	user, err := scanUser(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE email = $1 AND deleted_at IS NULL`
	user, err := scanUser(r.db.QueryRowContext(ctx, query, email))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

func (r *userRepository) GetByVerifyToken(ctx context.Context, token string) (*entity.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE email_verify_token = $1 AND deleted_at IS NULL`
	user, err := scanUser(r.db.QueryRowContext(ctx, query, token))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by verify token: %w", err)
	}
	return user, nil
}

func (r *userRepository) GetByResetToken(ctx context.Context, token string) (*entity.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE password_reset_token = $1 AND deleted_at IS NULL`
	user, err := scanUser(r.db.QueryRowContext(ctx, query, token))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by reset token: %w", err)
	}
	return user, nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	query := `UPDATE users SET email = $2, password = $3, name = $4, is_active = $5, updated_at = $6, failed_login_attempts = $7, locked_until = $8, email_verified = $9, email_verify_token = $10, email_verify_expires = $11, password_reset_token = $12, password_reset_expires = $13 WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, user.ID, user.Email, user.Password, user.Name, user.IsActive, user.UpdatedAt, user.FailedLoginAttempts, user.LockedUntil, user.EmailVerified, user.EmailVerifyToken, user.EmailVerifyExpires, user.PasswordResetToken, user.PasswordResetExpires)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
