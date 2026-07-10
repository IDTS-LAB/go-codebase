-- name: CreateUser :exec
INSERT INTO users (id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: GetUserByID :one
SELECT id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at, deleted_at
FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at, deleted_at
FROM users WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByVerifyToken :one
SELECT id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at, deleted_at
FROM users WHERE email_verify_token = $1 AND deleted_at IS NULL;

-- name: GetUserByResetToken :one
SELECT id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at, deleted_at
FROM users WHERE password_reset_token = $1 AND deleted_at IS NULL;

-- name: UpdateUser :execrows
UPDATE users SET email = $2, password = $3, name = $4, is_active = $5, updated_at = $6, failed_login_attempts = $7, locked_until = $8, email_verified = $9, email_verify_token = $10, email_verify_expires = $11, password_reset_token = $12, password_reset_expires = $13 WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetRefreshTokenByToken :one
SELECT id, user_id, token, expires_at, revoked_at, created_at, updated_at, deleted_at
FROM refresh_tokens WHERE token = $1 AND deleted_at IS NULL;

-- name: GetRefreshTokensByUserID :many
SELECT id, user_id, token, expires_at, revoked_at, created_at, updated_at, deleted_at
FROM refresh_tokens WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = NOW(), updated_at = NOW() WHERE token = $1 AND deleted_at IS NULL AND revoked_at IS NULL;

-- name: RevokeAllRefreshTokensByUserID :exec
UPDATE refresh_tokens SET revoked_at = NOW(), updated_at = NOW() WHERE user_id = $1 AND deleted_at IS NULL AND revoked_at IS NULL;

-- name: DeleteExpiredRefreshTokens :exec
UPDATE refresh_tokens SET deleted_at = NOW() WHERE expires_at < NOW() AND deleted_at IS NULL;
