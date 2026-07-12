-- +goose Up

-- Add tenant_id to existing tables
ALTER TABLE todos ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE roles ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE role_permissions ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE error_logs ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) NOT NULL DEFAULT '';

-- Composite indexes (tenant_id, created_at) for tables with created_at
CREATE INDEX IF NOT EXISTS idx_todos_tenant_created ON todos(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_roles_tenant_created ON roles(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_permissions_tenant_created ON permissions(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_user_roles_tenant_created ON user_roles(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_role_permissions_tenant_created ON role_permissions(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created ON audit_logs(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_error_logs_tenant_created ON error_logs(tenant_id, created_at);

-- Tenant-scoped indexes on users and tenants
CREATE INDEX IF NOT EXISTS idx_users_tenant_email ON users(tenant_id, email);
CREATE INDEX IF NOT EXISTS idx_tenants_tenant_slug ON tenants(tenant_id, slug);

-- +goose Down

ALTER TABLE todos DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE roles DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE permissions DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE user_roles DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE role_permissions DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE error_logs DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_todos_tenant_created;
DROP INDEX IF EXISTS idx_roles_tenant_created;
DROP INDEX IF EXISTS idx_permissions_tenant_created;
DROP INDEX IF EXISTS idx_user_roles_tenant_created;
DROP INDEX IF EXISTS idx_role_permissions_tenant_created;
DROP INDEX IF EXISTS idx_audit_logs_tenant_created;
DROP INDEX IF EXISTS idx_error_logs_tenant_created;
DROP INDEX IF EXISTS idx_users_tenant_email;
DROP INDEX IF EXISTS idx_tenants_tenant_slug;
