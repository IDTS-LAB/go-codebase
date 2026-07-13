# Multi-Tenancy + User Normalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add toggleable row-level multi-tenancy (tenant_id), normalized user tables (10 purpose-specific tables), and a tenant management module.

**Architecture:** Row-level isolation via `tenant_id` column on all domain tables. Tenant resolved from JWT → header → subdomain pipeline. User tables split from one monolithic `users` table into purpose-specific tables by module boundary.

**Tech Stack:** Go 1.25, Chi router, Uber Fx, sqlc, PostgreSQL 16, testify.

## Global Constraints

- Multi-tenancy must be toggleable via `multitenancy.enabled` config; when `false`, existing behavior is unchanged.
- All domain entities get `TenantID string` field.
- New user tables must include a data migration from the old `users` table.
- Each migration must be reversible (goose Up/Down).
- Run `go build ./... && go vet ./... && go test ./...` before each commit.

---

## File Structure

```
migrations/
  007_create_tenants.sql               — new
  008_normalize_users.sql              — new
  009_add_tenant_id.sql                — new
internal/
  shared/
    config/config.go                    — modify: add TenantConfig
    middleware/tenant.go                — new: tenant context + middleware
    middleware/middleware.go            — modify: add TenantIDKey
    router/router.go                   — modify: wire TenantResolver
  core/domain/
    token.go                            — modify: add TenantID field
  infrastructure/auth/jwt.go            — modify: add TenantID to claims
  authentication/                       — modify: split to new tables
    domain/entity/user.go               — modify: remove old fields
    domain/event/events.go              — modify
    application/service/...             — modify: use new tables
    infrastructure/persistence/...      — modify: sqlc + repos
    interfaces/http/handlers.go         — modify
  user/                                 — modify: use new profile/address tables
    application/service/...             — modify
    infrastructure/persistence/...      — modify
    interfaces/http/handler.go          — modify
  tenant/                               — new module
    domain/entity/tenant.go             — new
    domain/repository/tenant.go         — new
    application/service/tenant.go       — new
    application/dto/tenant.go           — new
    infrastructure/persistence/...      — new: sqlc + repos
    interfaces/http/handlers.go         — new
    interfaces/http/routes.go           — new
    module.go                           — new
  shared/events/                        — modify if UserEvents change
  todo/                                 — modify: add tenant_id to queries
  auditlog/                             — modify: add tenant_id
  authorization/                        — modify: add tenant_id to RBAC tables
```

---

### Task 1: Config, Context Keys, and Middleware

**Files:**
- Modify: `internal/shared/config/config.go`
- Modify: `configs/config.yaml`
- Create: `internal/shared/middleware/tenant.go`
- Modify: `internal/shared/middleware/middleware.go`

**Interfaces:**
- Produces:
  ```go
  // config.go
  type TenantConfig struct {
      Enabled       bool   `yaml:"enabled"`
      TenantHeader  string `yaml:"tenant_header"`
      TenantJWTClaim string `yaml:"tenant_jwt_claim"`
      Domain        string `yaml:"domain"`
  }

  // middleware/middleware.go
  const TenantIDKey contextKey = "tenant_id"
  func GetTenantID(ctx context.Context) string

  // middleware/tenant.go
  func TenantResolver(cfg *config.TenantConfig) func(http.Handler) http.Handler
  func getDomainFromHost(host, domainSuffix string) string
  ```

- [ ] **Step 1: Add TenantConfig to config**

  In `internal/shared/config/config.go`, add:
  ```go
  type TenantConfig struct {
      Enabled       bool   `yaml:"enabled"`
      TenantHeader  string `yaml:"tenant_header"`
      TenantJWTClaim string `yaml:"tenant_jwt_claim"`
      Domain        string `yaml:"domain"`
  }
  ```

  Add to `Config` struct:
  ```go
  Tenant TenantConfig `yaml:"multitenancy"`
  ```

  Add env overrides in `applyEnvOverrides()`:
  ```go
  c.Tenant.Enabled = getEnvBool("MULTITENANCY_ENABLED", c.Tenant.Enabled)
  c.Tenant.TenantHeader = getEnv("MULTITENANCY_TENANT_HEADER", c.Tenant.TenantHeader)
  c.Tenant.Domain = getEnv("MULTITENANCY_DOMAIN", c.Tenant.Domain)
  ```

  Also add `getEnvBool` helper if not present.

- [ ] **Step 2: Update configs/config.yaml**

  Add to `configs/config.yaml`:
  ```yaml
  multitenancy:
    enabled: false
    tenant_header: "X-Tenant-ID"
    tenant_jwt_claim: "tenant_id"
    domain: "app.com"
  ```

- [ ] **Step 3: Add TenantID to context keys**

  In `internal/shared/middleware/middleware.go`, add:
  ```go
  const TenantIDKey contextKey = "tenant_id"
  ```

  Add `GetTenantID`:
  ```go
  func GetTenantID(ctx context.Context) string {
      if v, ok := ctx.Value(TenantIDKey).(string); ok {
          return v
      }
      return ""
  }
  ```

- [ ] **Step 4: Create tenant middleware**

  Create `internal/shared/middleware/tenant.go`:
  ```go
  package middleware

  import (
      "context"
      "net/http"
      "strings"

      "github.com/IDTS-LAB/go-codebase/internal/shared/config"
  )

  func TenantResolver(cfg *config.TenantConfig) func(http.Handler) http.Handler {
      return func(next http.Handler) http.Handler {
          return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              tenantID := ""
              if cfg.Enabled {
                  tenantID = resolveTenant(r, cfg)
              }
              ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
              next.ServeHTTP(w, r.WithContext(ctx))
          })
      }
  }

  func resolveTenant(r *http.Request, cfg *config.TenantConfig) string {
      // 1. JWT claim (already in context after auth middleware)
      if tid := GetTenantIDFromClaims(r.Context()); tid != "" {
          return tid
      }
      // 2. Header override
      if h := r.Header.Get(cfg.TenantHeader); h != "" {
          return h
      }
      // 3. Subdomain
      if sub := domainFromHost(r.Host, cfg.Domain); sub != "" && sub != "www" {
          return sub
      }
      return ""
  }

  func domainFromHost(host, domainSuffix string) string {
      host = strings.Split(host, ":")[0] // strip port
      if !strings.HasSuffix(host, "."+domainSuffix) {
          return ""
      }
      return strings.TrimSuffix(host, "."+domainSuffix)
  }

  func GetTenantIDFromClaims(ctx context.Context) string {
      // Extract from JWT claims stored in context by auth middleware
      // The auth middleware stores claims; we need a helper to get tenant_id
      return ""
  }
  ```

  Note: `GetTenantIDFromClaims` is a stub — it will be filled in Task 3 when JWT claims include TenantID. For now it returns empty string.

- [ ] **Step 5: Build and test**

  Run: `go build ./...`
  Expected: success.

- [ ] **Step 6: Commit**

  ```bash
  git add internal/shared/config/ internal/shared/middleware/ configs/config.yaml
  git commit -m "feat(multitenancy): add config, context keys, and tenant resolver middleware"
  ```

---

### Task 2: Wire TenantResolver into Router

**Files:**
- Modify: `internal/shared/router/router.go`

- [ ] **Step 1: Wire middleware**

  In `internal/shared/router/router.go`, add `TenantResolver` after `Auth` middleware:
  ```go
  r.Group(func(r chi.Router) {
      r.Use(mw.Auth)
      r.Use(middleware.TenantResolver(&cfg.Tenant)) // add here
      r.Use(mw.MaxBodySize)
      r.Mount("/todos", h.Todo)
      r.Mount("/users", h.User)
      r.Mount("/auth/sessions", h.Authz)
  })
  ```

  Add `middleware` import if needed.

- [ ] **Step 2: Build**

  Run: `go build ./...`
  Expected: success.

- [ ] **Step 3: Commit**

  ```bash
  git add internal/shared/router/router.go
  git commit -m "feat(router): wire TenantResolver middleware"
  ```

---

### Task 3: Add TenantID to JWT Claims and Token Generation

**Files:**
- Modify: `internal/core/domain/token.go`
- Modify: `internal/infrastructure/auth/jwt.go`
- Modify: `internal/authentication/application/service/authentication_service.go`
- Modify: `internal/shared/middleware/tenant.go` (fill GetTenantIDFromClaims)

**Interfaces:**
- Consumes: `middleware.GetTenantID(ctx)` from Task 1.
- Produces: `TokenClaims.TenantID string` stored in JWT and context.

- [ ] **Step 1: Add TenantID to TokenClaims**

  In `internal/core/domain/token.go`:
  ```go
  type TokenClaims struct {
      UserID   string
      Email    string
      Role     string
      JTI      string
      TenantID string
  }
  ```

- [ ] **Step 2: Update JWT generation**

  In `internal/infrastructure/auth/jwt.go`:
  ```go
  // In GenerateToken, add after Role:
  claims["tenant_id"] = tc.TenantID
  ```

  In `ValidateToken`, add after parsing:
  ```go
  tenantID, _ := parsedToken.Claims.Get("tenant_id")
  // Set TenantID on the returned TokenClaims
  ```

- [ ] **Step 3: Pass TenantID when generating tokens**

  In `authentication_service.go`, when creating `domain.TokenClaims`, set `TenantID` from context:
  ```go
  tenantID := middleware.GetTenantID(ctx)
  claims := domain.TokenClaims{
      UserID:   user.ID.String(),
      Email:    user.Email,
      Role:     user.Role,
      JTI:      uuid.New().String(),
      TenantID: tenantID,
  }
  ```

  Add `"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"` import.

- [ ] **Step 4: Fill GetTenantIDFromClaims**

  In `internal/shared/middleware/tenant.go`, implement the stub:
  ```go
  func GetTenantIDFromClaims(ctx context.Context) string {
      return GetUserClaim(ctx, "tenant_id")
  }
  ```
  Or, since auth middleware stores claims in context, add a generic claim getter or use the specific claim from stored UserID/Email/Role pattern.

  The cleanest approach: add `GetUserClaim` helper or read from JWT claims store. Since the auth middleware stores claims directly, add:
  ```go
  const TenantClaimKey contextKey = "tenant_claim"
  func SetTenantClaim(ctx context.Context, tenantID string) context.Context {
      return context.WithValue(ctx, TenantClaimKey, tenantID)
  }
  func GetTenantIDFromClaims(ctx context.Context) string {
      if v, ok := ctx.Value(TenantClaimKey).(string); ok {
          return v
      }
      return ""
  }
  ```

  Set it in `authentication_service.go` when creating claims:
  ```go
  ctx = middleware.SetTenantClaim(ctx, tenantID)
  ```

- [ ] **Step 5: Build and test**

  Run: `go build ./... && go test ./internal/authentication/...`
  Expected: success.

- [ ] **Step 6: Commit**

  ```bash
  git add internal/core/domain/token.go internal/infrastructure/auth/jwt.go \
         internal/authentication/application/service/authentication_service.go \
         internal/shared/middleware/tenant.go
  git commit -m "feat(jwt): add TenantID to token claims and generation"
  ```

---

### Task 4: Create Tenant Module (Entity, Service, Repository, Handlers)

**Files:**
- Create: `internal/tenant/domain/entity/tenant.go`
- Create: `internal/tenant/domain/repository/tenant.go`
- Create: `internal/tenant/application/dto/tenant.go`
- Create: `internal/tenant/application/service/tenant.go`
- Create: `internal/tenant/infrastructure/persistence/sqlc/queries.sql`
- Create: `internal/tenant/infrastructure/persistence/tenant_repository.go`
- Create: `internal/tenant/interfaces/http/handlers.go`
- Create: `internal/tenant/interfaces/http/routes.go`
- Create: `internal/tenant/module.go`

**Interfaces:**
- Consumes: config, database, logger from Fx.
- Produces: Tenant CRUD handlers under `/api/v1/admin/tenants`.

- [ ] **Step 1: Create domain entity**

  `internal/tenant/domain/entity/tenant.go`:
  ```go
  package entity

  import (
      "time"
      "encoding/json"
      "github.com/google/uuid"
  )

  type Tenant struct {
      ID        uuid.UUID       `json:"id"`
      Name      string          `json:"name"`
      Slug      string          `json:"slug"`
      Domain    *string         `json:"domain,omitempty"`
      Settings  json.RawMessage `json:"settings"`
      IsActive  bool            `json:"is_active"`
      CreatedAt time.Time       `json:"created_at"`
      UpdatedAt time.Time       `json:"updated_at"`
  }
  ```

- [ ] **Step 2: Create repository interface**

  `internal/tenant/domain/repository/tenant.go`:
  ```go
  package repository

  import (
      "context"
      "github.com/IDTS-LAB/go-codebase/internal/tenant/domain/entity"
      "github.com/google/uuid"
  )

  type TenantRepository interface {
      Create(ctx context.Context, t *entity.Tenant) error
      GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error)
      GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error)
      List(ctx context.Context, offset, limit int) ([]entity.Tenant, int, error)
      Update(ctx context.Context, t *entity.Tenant) error
      Delete(ctx context.Context, id uuid.UUID) error
  }
  ```

- [ ] **Step 3: Create DTOs**

  `internal/tenant/application/dto/tenant.go`:
  ```go
  package dto

  import "encoding/json"

  type CreateTenantRequest struct {
      Name     string          `json:"name" validate:"required"`
      Slug     string          `json:"slug" validate:"required"`
      Domain   *string         `json:"domain"`
      Settings json.RawMessage `json:"settings"`
  }

  type UpdateTenantRequest struct {
      Name     *string         `json:"name"`
      Domain   *string         `json:"domain"`
      Settings json.RawMessage `json:"settings"`
      IsActive *bool           `json:"is_active"`
  }

  type TenantResponse struct {
      ID        string          `json:"id"`
      Name      string          `json:"name"`
      Slug      string          `json:"slug"`
      Domain    *string         `json:"domain,omitempty"`
      Settings  json.RawMessage `json:"settings"`
      IsActive  bool            `json:"is_active"`
      CreatedAt string          `json:"created_at"`
      UpdatedAt string          `json:"updated_at"`
  }
  ```

- [ ] **Step 4: Create service**

  `internal/tenant/application/service/tenant.go`:
  ```go
  package service

  import (
      "context"
      "errors"
      "time"

      "github.com/IDTS-LAB/go-codebase/internal/core/domain"
      "github.com/IDTS-LAB/go-codebase/internal/tenant/domain/entity"
      "github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
      "github.com/google/uuid"
  )

  var (
      ErrTenantNotFound = errors.New("tenant not found")
      ErrTenantExists   = errors.New("tenant slug already exists")
  )

  type TenantService struct {
      repo repository.TenantRepository
  }

  func NewTenantService(repo repository.TenantRepository) *TenantService {
      return &TenantService{repo: repo}
  }

  func (s *TenantService) Create(ctx context.Context, name, slug string, domain *string, settings []byte) (*entity.Tenant, error) {
      tenant := &entity.Tenant{
          ID:        uuid.New(),
          Name:      name,
          Slug:      slug,
          Domain:    domain,
          Settings:  settings,
          IsActive:  true,
          CreatedAt: time.Now(),
          UpdatedAt: time.Now(),
      }
      if err := s.repo.Create(ctx, tenant); err != nil {
          return nil, ErrTenantExists
      }
      return tenant, nil
  }

  func (s *TenantService) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
      tenant, err := s.repo.GetByID(ctx, id)
      if err != nil {
          return nil, ErrTenantNotFound
      }
      return tenant, nil
  }

  func (s *TenantService) List(ctx context.Context, page, perPage int) ([]entity.Tenant, int, error) {
      offset := (page - 1) * perPage
      return s.repo.List(ctx, offset, perPage)
  }

  func (s *TenantService) Update(ctx context.Context, id uuid.UUID, req interface{}) (*entity.Tenant, error) { ... }
  func (s *TenantService) Delete(ctx context.Context, id uuid.UUID) error { ... }
  ```

- [ ] **Step 5: Create sqlc queries**

  `internal/tenant/infrastructure/persistence/sqlc/queries.sql`:
  ```sql
  -- name: CreateTenant :exec
  INSERT INTO tenants (id, name, slug, domain, settings, is_active, created_at, updated_at)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

  -- name: GetTenantByID :one
  SELECT * FROM tenants WHERE id = $1;

  -- name: GetTenantBySlug :one
  SELECT * FROM tenants WHERE slug = $1;

  -- name: ListTenants :many
  SELECT * FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2;

  -- name: UpdateTenant :exec
  UPDATE tenants SET name = $2, domain = $3, settings = $4, is_active = $5, updated_at = now() WHERE id = $1;

  -- name: DeleteTenant :exec
  DELETE FROM tenants WHERE id = $1;
  ```

- [ ] **Step 6: Create infrastructure repository**

  Implement `TenantRepository` using sqlc-generated code.

- [ ] **Step 7: Create HTTP handlers + routes + Fx module**

  Wire everything with Fx. Mount at `/api/v1/admin/tenants`.

- [ ] **Step 8: Build and test**

  Run: `go build ./...`
  Expected: success.

- [ ] **Step 9: Commit**

  ```bash
  git add internal/tenant/
  git commit -m "feat(tenant): add tenant management module"
  ```

---

### Task 5: Migration 007 — Create Tenants Table

**Files:**
- Create: `migrations/007_create_tenants.sql`

- [ ] **Step 1: Write migration**

  `migrations/007_create_tenants.sql`:
  ```sql
  -- +goose Up
  CREATE TABLE tenants (
      id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      name       VARCHAR(255) NOT NULL,
      slug       VARCHAR(100) NOT NULL UNIQUE,
      domain     VARCHAR(255),
      settings   JSONB NOT NULL DEFAULT '{}',
      is_active  BOOLEAN NOT NULL DEFAULT true,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );

  -- +goose Down
  DROP TABLE IF EXISTS tenants;
  ```

- [ ] **Step 2: Commit**

  ```bash
  git add migrations/007_create_tenants.sql
  git commit -m "feat(migrations): create tenants table"
  ```

---

### Task 6: Migration 008 — Normalize User Tables

**Files:**
- Create: `migrations/008_normalize_users.sql`

- [ ] **Step 1: Write the migration**

  Write the complete Up migration that creates all 9 new tables, copies data from old `users` table, and drops the old columns.

  The old `users` table has columns: id, email, password_hash, name, is_active, email_verified_at, verification_token, verification_token_expires_at, reset_token, reset_token_expires_at, login_attempts, locked_until, last_login_at, created_at, updated_at, deleted_at.

  Write Down migration that reverses the split (re-adds columns, copies data back).

  See `docs/superpowers/specs/2026-07-12-multitenancy-design.md` for the exact table schemas.

- [ ] **Step 2: Commit**

  ```bash
  git add migrations/008_normalize_users.sql
  git commit -m "feat(migrations): normalize users into purpose-specific tables"
  ```

---

### Task 7: Migration 009 — Add tenant_id to Domain Tables

**Files:**
- Create: `migrations/009_add_tenant_id.sql`

- [ ] **Step 1: Write the migration**

  Add `tenant_id VARCHAR(36) NOT NULL DEFAULT ''` to: todos, users, roles, permissions, user_roles, role_permissions, audit_logs, error_logs.

  Add composite indexes: `(tenant_id, created_at)`, `(tenant_id, id)`.

- [ ] **Step 2: Commit**

  ```bash
  git add migrations/009_add_tenant_id.sql
  git commit -m "feat(migrations): add tenant_id column to domain tables"
  ```

---

### Task 8: Update Auth Module for New User Tables

**Files:**
- Modify: `internal/authentication/domain/entity/user.go`
- Modify: `internal/authentication/domain/entity/refresh_token.go`
- Modify: `internal/authentication/domain/repository/user_repository.go`
- Modify: `internal/authentication/domain/repository/refresh_token_repository.go`
- Modify: `internal/authentication/application/service/authentication_service.go`
- Modify: `internal/authentication/infrastructure/persistence/` (all files)
- Modify: `internal/authentication/infrastructure/persistence/sqlc/queries.sql`
- Modify: `internal/shared/auditlog/` (add tenant_id)

**Scope:** Refactor auth module to use new user_credentials, user_security, user_tokens, user_sessions tables instead of the old monolithic users table.

- [ ] **Step 1: Update domain entities**

  `User` entity no longer has password_hash, login_attempts, locked_until, etc. Keep only: ID, Email, Name, IsActive, EmailVerifiedAt, TenantID, timestamps.

  Create new entities for `UserCredential`, `UserSecurity`, `UserToken`, `UserSession` in the auth module's domain.

- [ ] **Step 2: Update repository interfaces**

  `UserRepository`:
  - Remove `UpdatePassword`, `GetByRefreshToken`, `IncrementLoginAttempts`, `LockAccount`, `ResetLoginAttempts`
  - Add credential/security lookup methods or a separate `CredentialRepository`

- [ ] **Step 3: Update sqlc queries**

  Add queries for the new tables. Generate new sqlc models.

- [ ] **Step 4: Update service layer**

  `AuthenticationService` uses the new repositories for register, login, verify, reset, refresh, logout flows.

- [ ] **Step 5: Build and test**

  Run: `go build ./... && go test ./internal/authentication/...`
  Expected: success. Fix any test that references old entity fields.

- [ ] **Step 6: Commit**

  ```bash
  git add internal/authentication/
  git commit -m "refactor(auth): use normalized user tables"
  ```

---

### Task 9: Update User Module for New Profile/Address/Preference Tables

**Files:**
- Modify: `internal/user/domain/entity/` (if any)
- Modify: `internal/user/domain/repository/user_repository.go`
- Modify: `internal/user/application/service/` 
- Modify: `internal/user/infrastructure/persistence/`
- Modify: `internal/user/interfaces/http/handler.go`

**Scope:** Update user module to read/write from user_profiles and user_addresses tables.

- [ ] **Step 1: Update repository and service**

  `UserRepository` gains `GetProfile(ctx, userID)`, `UpdateProfile(ctx, userID, ...)`, `ListAddresses(ctx, userID)`, `CreateAddress`, etc.

- [ ] **Step 2: Build and test**

  Run: `go build ./... && go test ./internal/user/...`
  Expected: success.

- [ ] **Step 3: Commit**

  ```bash
  git add internal/user/
  git commit -m "refactor(user): use profile, address, and preference tables"
  ```

---

### Task 10: Add tenant_id Filtering to All Repositories

**Files:**
- Modify: `internal/todo/infrastructure/persistence/` (sqlc + repository)
- Modify: `internal/todo/infrastructure/persistence/sqlc/queries.sql`
- Modify: `internal/authorization/infrastructure/persistence/` (sqlc + repository)
- Modify: `internal/authorization/infrastructure/persistence/sqlc/queries.sql`
- Modify: `internal/auditlog/` (sqlc + repository)

**Scope:** Add `tenant_id` field to all CreateParams and `WHERE tenant_id = $N` to all queries.

- [ ] **Step 1: Add tenantFromCtx helper**

  In `internal/shared/middleware/tenant.go` or a shared utils file, add:
  ```go
  func TenantFromCtx(ctx context.Context) string {
      return GetTenantID(ctx)
  }
  ```

- [ ] **Step 2-5: Update each module's sqlc and repository**

  For each module:
  1. Add `tenant_id` to CREATE params
  2. Filter SELECT/UPDATE/DELETE by tenant_id
  3. Regenerate sqlc code
  4. Build and test

- [ ] **Step 6: Commit**

  ```bash
  git add internal/todo/ internal/authorization/ internal/shared/auditlog/
  git commit -m "feat(multitenancy): add tenant_id filtering to all repositories"
  ```

---

### Task 11: Update Todo Command Handlers for Tenant Context

**Files:**
- Modify: `internal/todo/application/command/` (create, update, delete, complete, list)

**Scope:** Pass tenant context from middleware through to repository.

- [ ] **Step 1: Pass tenant context**

  In command handlers, ensure context propagation passes tenant_id.

- [ ] **Step 2: Build and test**

  Run: `go build ./... && go test ./internal/todo/...`
  Expected: success.

- [ ] **Step 3: Commit**

  ```bash
  git add internal/todo/application/command/
  git commit -m "feat(todo): pass tenant context through command handlers"
  ```

---

### Task 12: Update Authorization Module for Tenant Scope

**Files:**
- Modify: `internal/authorization/infrastructure/casbin/enforcer.go`
- Modify: `internal/authorization/application/service/authorization_service.go`

**Scope:** Casbin policies can optionally be scoped to tenant. When multitenancy is enabled, role names may include tenant prefix or Casbin policies get a tenant_id field.

- [ ] **Step 1: Update Casbin enforcer**

  Add tenant-aware policy checking.

- [ ] **Step 2: Build and test**

  Run: `go build ./... && go test ./internal/authorization/...`
  Expected: success.

- [ ] **Step 3: Commit**

  ```bash
  git add internal/authorization/
  git commit -m "feat(authz): add tenant-aware Casbin policy scope"
  ```

---

### Task 13: Audit Log Tenant Context

**Files:**
- Modify: `internal/shared/auditlog/` (entity, repository, middleware)

**Scope:** Include tenant_id in audit log entries.

- [ ] **Step 1: Add tenant_id to AuditLog entity and Insert params**

- [ ] **Step 2: Build and test**

- [ ] **Step 3: Commit**

---

### Task 14: Full Integration and Test Pass

- [ ] **Step 1: Run full suite**

  Run: `go build ./... && go vet ./... && go test ./...`
  Expected: all pass.

- [ ] **Step 2: Fix any failures**

- [ ] **Step 3: Commit**

  ```bash
  git add -A
  git commit -m "chore: fix tests and build after multitenancy refactor"
  ```
