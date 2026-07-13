-- name: CreateTodo :exec
INSERT INTO todos (id, title, description, completed, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetTodoByID :one
SELECT id, title, description, completed, created_at, updated_at, deleted_at
FROM todos
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateTodo :execrows
UPDATE todos
SET title = $2, description = $3, completed = $4, updated_at = $5
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteTodo :execrows
UPDATE todos
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
