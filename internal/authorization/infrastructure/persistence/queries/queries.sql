-- name: CreateRole :exec
INSERT INTO roles (id, name, description, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetRoleByID :one
SELECT id, name, description, created_at, updated_at, deleted_at
FROM roles WHERE id = $1 AND deleted_at IS NULL;

-- name: GetRoleByName :one
SELECT id, name, description, created_at, updated_at, deleted_at
FROM roles WHERE name = $1 AND deleted_at IS NULL;

-- name: UpdateRole :execrows
UPDATE roles SET name = $2, description = $3, updated_at = $4 WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteRole :execrows
UPDATE roles SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreatePermission :exec
INSERT INTO permissions (id, name, description, resource, action, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetPermissionByID :one
SELECT id, name, description, resource, action, created_at, updated_at, deleted_at
FROM permissions WHERE id = $1 AND deleted_at IS NULL;

-- name: GetPermissionByName :one
SELECT id, name, description, resource, action, created_at, updated_at, deleted_at
FROM permissions WHERE name = $1 AND deleted_at IS NULL;

-- name: UpdatePermission :execrows
UPDATE permissions SET name = $2, description = $3, resource = $4, action = $5, updated_at = $6 WHERE id = $1 AND deleted_at IS NULL;

-- name: DeletePermission :execrows
UPDATE permissions SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL;

-- name: AssignRolePermission :exec
INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT (role_id, permission_id) DO NOTHING;

-- name: RemoveRolePermission :exec
DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2;

-- name: GetRolePermissionsByRoleID :many
SELECT role_id, permission_id FROM role_permissions WHERE role_id = $1;

-- name: GetPermissionsByRoleID :many
SELECT p.id, p.name, p.description, p.resource, p.action, p.created_at, p.updated_at, p.deleted_at
FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = $1 AND p.deleted_at IS NULL
ORDER BY p.created_at DESC;

-- name: AssignUserRole :exec
INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT (user_id, role_id) DO NOTHING;

-- name: RemoveUserRole :exec
DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2;

-- name: GetUserRolesByUserID :many
SELECT user_id, role_id FROM user_roles WHERE user_id = $1;

-- name: GetRolesByUserID :many
SELECT r.id, r.name, r.description, r.created_at, r.updated_at, r.deleted_at
FROM roles r
JOIN user_roles ur ON r.id = ur.role_id
WHERE ur.user_id = $1 AND r.deleted_at IS NULL
ORDER BY r.created_at DESC;
