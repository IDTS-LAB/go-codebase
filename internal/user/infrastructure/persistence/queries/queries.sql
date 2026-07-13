-- name: GetUserByID :one
SELECT id, email, name, is_active, created_at, updated_at, deleted_at
FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUser :execrows
UPDATE users SET email = $2, name = $3, is_active = $4, updated_at = $5 WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteUser :execrows
UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL;
