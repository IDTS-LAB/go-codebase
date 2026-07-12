-- name: CreateTenant :exec
INSERT INTO tenants (id, name, slug, domain, settings, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetTenantByID :one
SELECT id, name, slug, domain, settings, is_active, created_at, updated_at
FROM tenants WHERE id = $1;

-- name: GetTenantBySlug :one
SELECT id, name, slug, domain, settings, is_active, created_at, updated_at
FROM tenants WHERE slug = $1;

-- name: ListTenants :many
SELECT id, name, slug, domain, settings, is_active, created_at, updated_at
FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountTenants :one
SELECT COUNT(*) FROM tenants;

-- name: UpdateTenant :execrows
UPDATE tenants SET name = $2, domain = $3, settings = $4, is_active = $5, updated_at = $6
WHERE id = $1;

-- name: DeleteTenant :execrows
DELETE FROM tenants WHERE id = $1;
