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
