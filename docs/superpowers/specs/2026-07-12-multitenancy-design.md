# Multi-Tenancy Design

**Date:** 2026-07-12
**Topic:** Multi-tenancy support
**Status:** Approved

## Goal

Add multi-tenant support with row-level isolation via `tenant_id` column, toggleable through config. When disabled, all existing code works unchanged.

## Architecture

**Strategy:** Row-level tenant isolation (shared DB, `tenant_id` column on domain tables).

**Toggle:** Config flag `multitenancy.enabled`. When `false`, tenant_id is always `""` and all queries behave as they do today.

## Tenant Resolution Pipeline

```
Request
  → Extract JWT tenant_id claim (primary source for authenticated users)
  → If JWT tenant_id is empty AND user is super-admin:
      → Check X-Tenant-ID header for override
  → If still empty:
      → Extract from subdomain (tenant.app.com → "tenant")
  → Store resolved TenantID in request context
  → If multitenancy disabled OR no tenant resolved: TenantID = ""
```

### Resolution Sources

1. **JWT claim** (`tenant_id`) — authoritative for authenticated users
2. **HTTP Header** (`X-Tenant-ID`) — only respected when JWT `tenant_id` is empty (super-admin override)
3. **Subdomain** — `{tenant}.{domain}` parsed from `Host` header, fallback when JWT/header absent

### Conflict Resolution

| JWT tenant_id | X-Tenant-ID | Subdomain | Result |
|---|---|---|---|
| `"a"` | any | any | `"a"` (JWT wins) |
| `""` | `"b"` | `"c"` | `"b"` (header wins) |
| `""` | `""` | `"c"` | `"c"` (subdomain fallback) |
| `""` | `""` | `""` | `""` (no tenant) |

## Configuration

```yaml
multitenancy:
  enabled: false
  tenant_header: "X-Tenant-ID"
  tenant_jwt_claim: "tenant_id"
  domain: "app.com"
```

Env var equivalents: `MULTITENANCY_ENABLED`, `MULTITENANCY_TENANT_HEADER`, etc.

## Components

### 1. Tenant Context (`internal/shared/middleware/tenant.go`)
- New context key `TenantIDKey contextKey = "tenant_id"`
- `GetTenantID(ctx) string` / helper

### 2. Tenant Middleware (`internal/shared/middleware/tenant.go`)
- `TenantResolver(cfg *config.Config)` middleware
- Runs AFTER auth middleware
- Pipeline: JWT claim → header → subdomain → context

### 3. Config Changes
- Add `multitenancy` section to `configs/config.yaml`
- Add `TenantConfig` struct to `internal/shared/config/config.go`

### 4. JWT Changes
- `TokenClaims` gets `TenantID string`
- Token generation includes `tenant_id` claim

### 5. Tenants Table

A `tenants` table stores tenant metadata. This table is NOT tenant-scoped (it defines tenants).

```sql
CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(100) NOT NULL UNIQUE,   -- used in subdomain resolution
    domain     VARCHAR(255),                    -- custom domain (optional)
    settings   JSONB NOT NULL DEFAULT '{}',     -- flexible feature flags/limits
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Settings JSONB shape (example):
```json
{
    "max_users": 100,
    "max_storage_mb": 1024,
    "features": ["audit_log", "email_notifications"]
}
```

### 6. Tenant Domain

A `internal/tenant/` module with its own domain/application/infrastructure layers:
- **Entity:** `Tenant` (id, name, slug, domain, settings, is_active, timestamps)
- **Repository:** `TenantRepository` (CRUD + FindBySlug)
- **Service:** `TenantService` (CRUD, validation)
- **Handlers:** Admin-only CRUD endpoints under `/api/v1/admin/tenants`

**Tenant handlers are excluded from tenant filtering** (they manage tenants themselves).

### 7. Database Migrations

**Migration `007_create_tenants.sql`** — creates the tenants table.

**Migration `008_normalize_users.sql`** — splits the monolithic `users` table into purpose-specific tables:

```sql
-- Core identity (auth module)
CREATE TABLE users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email            VARCHAR(255) NOT NULL UNIQUE,
    name             VARCHAR(255) NOT NULL DEFAULT '',
    is_active        BOOLEAN NOT NULL DEFAULT true,
    email_verified_at TIMESTAMPTZ,
    tenant_id        VARCHAR(36) NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

-- Authentication secrets (auth module)
CREATE TABLE user_credentials (
    user_id       UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    last_login_at TIMESTAMPTZ
);

-- Security/lockout (auth module)
CREATE TABLE user_security (
    user_id        UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    login_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until   TIMESTAMPTZ,
    mfa_enabled    BOOLEAN NOT NULL DEFAULT false,
    mfa_secret     VARCHAR(255)
);

-- Verification and reset tokens (auth module)
CREATE TABLE user_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_type  VARCHAR(50) NOT NULL,  -- 'email_verification', 'password_reset'
    token_hash  VARCHAR(255) NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_user_tokens_user_id ON user_tokens(user_id);
CREATE INDEX idx_user_tokens_hash ON user_tokens(token_hash);

-- Profile / personal info (user module)
CREATE TABLE user_profiles (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    first_name  VARCHAR(255) NOT NULL DEFAULT '',
    last_name   VARCHAR(255) NOT NULL DEFAULT '',
    phone       VARCHAR(50),
    avatar_url  VARCHAR(500),
    timezone    VARCHAR(100) DEFAULT 'UTC',
    locale      VARCHAR(10) DEFAULT 'en',
    bio         TEXT
);

-- Addresses (user module, one-to-many)
CREATE TABLE user_addresses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label       VARCHAR(100),                    -- 'Home', 'Work', 'Billing'
    is_default  BOOLEAN NOT NULL DEFAULT false,
    street      VARCHAR(255),
    city        VARCHAR(100),
    state       VARCHAR(100),
    postal_code VARCHAR(20),
    country     VARCHAR(100),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_user_addresses_user_id ON user_addresses(user_id);

-- Active sessions (auth module)
CREATE TABLE user_sessions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    device_info       VARCHAR(500),
    ip_address        VARCHAR(45),
    user_agent        TEXT,
    expires_at        TIMESTAMPTZ NOT NULL,
    last_used_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);

-- Social auth links (auth module)
CREATE TABLE user_social_links (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        VARCHAR(50) NOT NULL,  -- 'google', 'github', 'apple'
    provider_id     VARCHAR(255) NOT NULL,
    provider_email  VARCHAR(255),
    avatar_url      VARCHAR(500),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, provider_id)
);
CREATE INDEX idx_user_social_links_user_id ON user_social_links(user_id);

-- User preferences (user module)
CREATE TABLE user_preferences (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    preferences JSONB NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**Data migration:** The `008_normalize_users.sql` migration will use INSERT INTO ... SELECT to copy data from the old `users` table into the new normalized tables, then drop the old columns (or keep `deleted_at` on `users`).

**Migration `009_add_tenant_id.sql`** — adds `tenant_id VARCHAR(36) NOT NULL DEFAULT ''` to all domain tables with composite indexes `(tenant_id, ...)`.

### 9. Repository Changes
- All queries filter by `tenant_id` when present
- Helper `tenantFromCtx(ctx)` extracts tenant from context
- Creates include `TenantID` from context
- Unauthenticated endpoints skip tenant filtering

### 10. Wire
- TenantResolver added to router middleware chain after auth

## Toggle Behavior

| State | tenant_id value | Query behavior |
|---|---|---|
| `enabled: false` | `""` | Everything works as today |
| `enabled: true`, no tenant resolved | `""` | Non-tenant rows (admin/legacy) |
| `enabled: true`, tenant resolved | UUID string | All queries scoped to tenant |

## Key Files Changed

- `internal/shared/config/config.go` — add TenantConfig
- `configs/config.yaml` — add multitenancy section
- `internal/shared/middleware/tenant.go` — new file
- `internal/infrastructure/auth/jwt.go` — add TenantID to claims
- `internal/core/domain/token.go` — add TenantID to TokenClaims
- `internal/tenant/` — new module (entity, repository, service, handlers, module.go)
- `migrations/007_create_tenants.sql` — tenants table
- `migrations/008_normalize_users.sql` — user tables split (users, credentials, security, tokens, profiles, addresses, sessions, social_links, preferences)
- `migrations/009_add_tenant_id.sql` — tenant_id columns to domain tables
- `sqlc.yaml` — add tenant_id field mapping, tenant queries
- All existing repository/sqlc files — tenant filtering (create params + WHERE clauses)
- `internal/shared/router/router.go` — wire TenantResolver middleware + admin routes
- `cmd/api/main.go` — wire tenant module
- Tests — pass tenant context where needed
