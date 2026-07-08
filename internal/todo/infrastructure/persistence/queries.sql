-- name: CreateTodo :exec
INSERT INTO todos (id, title, description, completed, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetTodoByID :one
SELECT id, title, description, completed, created_at, updated_at, deleted_at
FROM todos
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListTodos :many
SELECT id, title, description, completed, created_at, updated_at, deleted_at
FROM todos
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountTodos :one
SELECT COUNT(*) FROM todos WHERE deleted_at IS NULL;

-- name: UpdateTodo :exec
UPDATE todos
SET title = $2, description = $3, completed = $4, updated_at = $5
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteTodo :exec
UPDATE todos
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: SearchTodos :many
SELECT id, title, description, completed, created_at, updated_at, deleted_at
FROM todos
WHERE deleted_at IS NULL AND (title ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSearchTodos :one
SELECT COUNT(*) FROM todos
WHERE deleted_at IS NULL AND (title ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%');
