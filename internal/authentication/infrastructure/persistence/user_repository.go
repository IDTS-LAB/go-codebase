package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &userRepository{db: db}
}

type userRow struct {
	ID                   uuid.UUID
	Email                string
	Name                 string
	IsActive             bool
	EmailVerifiedAt      sql.NullTime
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            sql.NullTime
	PasswordHash         sql.NullString
	LastLoginAt          sql.NullTime
	LoginAttempts        sql.NullInt32
	LockedUntil          sql.NullTime
	EmailVerifyToken     sql.NullString
	EmailVerifyExpires   sql.NullTime
	PasswordResetToken   sql.NullString
	PasswordResetExpires sql.NullTime
}

func scanUser(row *sql.Row) (userRow, error) {
	var r userRow
	err := row.Scan(
		&r.ID,
		&r.Email,
		&r.Name,
		&r.IsActive,
		&r.EmailVerifiedAt,
		&r.CreatedAt,
		&r.UpdatedAt,
		&r.DeletedAt,
		&r.PasswordHash,
		&r.LastLoginAt,
		&r.LoginAttempts,
		&r.LockedUntil,
		&r.EmailVerifyToken,
		&r.EmailVerifyExpires,
		&r.PasswordResetToken,
		&r.PasswordResetExpires,
	)
	return r, err
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var emailVerifiedAt *time.Time
	if user.EmailVerified {
		now := time.Now().UTC()
		emailVerifiedAt = &now
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (id, email, name, is_active, email_verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, user.Email, user.Name, user.IsActive, ptrToNullTime(emailVerifiedAt), user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_credentials (user_id, password_hash, last_login_at)
		VALUES ($1, $2, $3)
	`, user.ID, user.Password, nil)
	if err != nil {
		return fmt.Errorf("insert user credentials: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_security (user_id, login_attempts, locked_until)
		VALUES ($1, $2, $3)
	`, user.ID, user.FailedLoginAttempts, ptrToNullTime(user.LockedUntil))
	if err != nil {
		return fmt.Errorf("insert user security: %w", err)
	}

	if user.EmailVerifyToken != nil {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_tokens (user_id, token_type, token_hash, expires_at)
			VALUES ($1, 'email_verification', $2, $3)
		`, user.ID, *user.EmailVerifyToken, ptrToNullTime(user.EmailVerifyExpires))
		if err != nil {
			return fmt.Errorf("insert email verify token: %w", err)
		}
	}

	if user.PasswordResetToken != nil {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_tokens (user_id, token_type, token_hash, expires_at)
			VALUES ($1, 'password_reset', $2, $3)
		`, user.ID, *user.PasswordResetToken, ptrToNullTime(user.PasswordResetExpires))
		if err != nil {
			return fmt.Errorf("insert password reset token: %w", err)
		}
	}

	return tx.Commit()
}

const userSelectColumns = `SELECT u.id, u.email, u.name, u.is_active, u.email_verified_at, u.created_at, u.updated_at, u.deleted_at,
       uc.password_hash, uc.last_login_at,
       us.login_attempts, us.locked_until`

const userTokenNulls = `NULL::varchar, NULL::timestamptz, NULL::varchar, NULL::timestamptz`

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query := userSelectColumns + `, ` + userTokenNulls + `
FROM users u
LEFT JOIN user_credentials uc ON u.id = uc.user_id
LEFT JOIN user_security us ON u.id = us.user_id
WHERE u.id = $1 AND u.deleted_at IS NULL`
	row := r.db.QueryRowContext(ctx, query, id)
	dbRow, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return mapRowToEntity(dbRow), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := userSelectColumns + `, ` + userTokenNulls + `
FROM users u
LEFT JOIN user_credentials uc ON u.id = uc.user_id
LEFT JOIN user_security us ON u.id = us.user_id
WHERE u.email = $1 AND u.deleted_at IS NULL`
	row := r.db.QueryRowContext(ctx, query, email)
	dbRow, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return mapRowToEntity(dbRow), nil
}

func (r *userRepository) GetByVerifyToken(ctx context.Context, token string) (*entity.User, error) {
	query := userSelectColumns + `,
       ut.token_hash, ut.expires_at,
       NULL::varchar, NULL::timestamptz
FROM users u
LEFT JOIN user_credentials uc ON u.id = uc.user_id
LEFT JOIN user_security us ON u.id = us.user_id
INNER JOIN user_tokens ut ON u.id = ut.user_id AND ut.token_hash = $1 AND ut.token_type = 'email_verification' AND ut.consumed_at IS NULL AND (ut.expires_at IS NULL OR ut.expires_at > NOW())
WHERE u.deleted_at IS NULL`
	row := r.db.QueryRowContext(ctx, query, token)
	dbRow, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by verify token: %w", err)
	}
	return mapRowToEntity(dbRow), nil
}

func (r *userRepository) GetByResetToken(ctx context.Context, token string) (*entity.User, error) {
	query := userSelectColumns + `,
       NULL::varchar, NULL::timestamptz,
       ut.token_hash, ut.expires_at
FROM users u
LEFT JOIN user_credentials uc ON u.id = uc.user_id
LEFT JOIN user_security us ON u.id = us.user_id
INNER JOIN user_tokens ut ON u.id = ut.user_id AND ut.token_hash = $1 AND ut.token_type = 'password_reset' AND ut.consumed_at IS NULL AND (ut.expires_at IS NULL OR ut.expires_at > NOW())
WHERE u.deleted_at IS NULL`
	row := r.db.QueryRowContext(ctx, query, token)
	dbRow, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by reset token: %w", err)
	}
	return mapRowToEntity(dbRow), nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `
		UPDATE users SET
		    email = $1,
		    name = $2,
		    is_active = $3,
		    email_verified_at = CASE WHEN $4 THEN COALESCE(email_verified_at, NOW()) ELSE NULL END,
		    updated_at = NOW()
		WHERE id = $5 AND deleted_at IS NULL
	`, user.Email, user.Name, user.IsActive, user.EmailVerified, user.ID)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_credentials (user_id, password_hash, last_login_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET password_hash = EXCLUDED.password_hash, last_login_at = EXCLUDED.last_login_at
	`, user.ID, user.Password, nil)
	if err != nil {
		return fmt.Errorf("upsert user credentials: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_security (user_id, login_attempts, locked_until)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET login_attempts = EXCLUDED.login_attempts, locked_until = EXCLUDED.locked_until
	`, user.ID, user.FailedLoginAttempts, ptrToNullTime(user.LockedUntil))
	if err != nil {
		return fmt.Errorf("upsert user security: %w", err)
	}

	if user.EmailVerifyToken != nil {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_tokens (user_id, token_type, token_hash, expires_at)
			VALUES ($1, 'email_verification', $2, $3)
		`, user.ID, *user.EmailVerifyToken, ptrToNullTime(user.EmailVerifyExpires))
		if err != nil {
			return fmt.Errorf("insert email verify token: %w", err)
		}
	}

	if user.PasswordResetToken != nil {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_tokens (user_id, token_type, token_hash, expires_at)
			VALUES ($1, 'password_reset', $2, $3)
		`, user.ID, *user.PasswordResetToken, ptrToNullTime(user.PasswordResetExpires))
		if err != nil {
			return fmt.Errorf("insert password reset token: %w", err)
		}
	}

	return tx.Commit()
}

func mapRowToEntity(row userRow) *entity.User {
	return &entity.User{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: nullTimeToPtr(row.DeletedAt),
		},
		Email:                row.Email,
		Password:             row.PasswordHash.String,
		Name:                 row.Name,
		IsActive:             row.IsActive,
		FailedLoginAttempts:  int(row.LoginAttempts.Int32),
		LockedUntil:          nullTimeToPtr(row.LockedUntil),
		EmailVerified:        row.EmailVerifiedAt.Valid,
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

func ptrToNullTime(t *time.Time) sql.NullTime {
	if t != nil {
		return sql.NullTime{Time: *t, Valid: true}
	}
	return sql.NullTime{Valid: false}
}
