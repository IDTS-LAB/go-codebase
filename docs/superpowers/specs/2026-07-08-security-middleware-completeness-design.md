# Security & Middleware Completeness Design

Date: 2026-07-08

## Context

Audit against a production readiness checklist revealed 6 missing features and 7 partial implementations. This spec covers implementing all 13 items to bring the codebase to full compliance.

## Items

### 1. Rate Limiting (Redis-backed) — NEW

**Middleware:** `RateLimit(rdb *redis.Client, limit int, window time.Duration)` in `internal/shared/middleware/ratelimit.go`

- Redis `INCR` + `EXPIRE` sliding window per client IP
- Returns `429 Too Many Requests` with `Retry-After` header
- Config: `RateLimit.Requests` (int, default 100), `RateLimit.Window` (int seconds, default 60)
- Env: `RATE_LIMIT_REQUESTS`, `RATE_LIMIT_WINDOW`

### 2. Security Headers Middleware — NEW

**Middleware:** `SecurityHeaders(next http.Handler)` in `internal/shared/middleware/headers.go`

Headers set:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Content-Security-Policy: frame-ancestors 'none'`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: camera=(), microphone=(), geolocation=()`

No config, always applied.

### 3. Audit Logging — NEW

**Middleware:** `AuditLogger(log domain.Logger)` in `internal/shared/middleware/audit.go`

- Logs: timestamp, request_id, method, path, status, user_id (if authed), IP, user-agent
- Uses structured fields via `domain.Logger`
- Status >= 500 logged at ERROR level, else INFO
- Separate from the existing request Logger middleware

### 4. Account Lockout — NEW

**User entity:** Add `FailedLoginAttempts int` and `LockedUntil *time.Time` fields

**Migration:** `004_add_account_lockout.sql` — adds columns to `users` table

**Auth service changes:**
- `Login()`: on failed password check, increment `FailedLoginAttempts`
- After `MaxLoginAttempts` (default 5) failures, set `LockedUntil = now + LockoutDuration`
- `Login()`: check `LockedUntil` before password validation; if locked, return `ErrAccountLocked`
- Successful login resets `FailedLoginAttempts = 0` and `LockedUntil = nil`

**Config:** `Auth.MaxLoginAttempts` (int, default 5), `Auth.LockoutDuration` (int seconds, default 900)

### 5. Token Revocation (access token denylist) — FIX PARTIAL

**Redis denylist:** On logout and refresh, store JWT `jti` in Redis with TTL = remaining token expiry
- Key: `token:blacklist:{jti}`
- `Authentication` middleware parses `jti` from claims, checks Redis before validation
- Requires adding `jti` to `domain.TokenClaims` and returning it from `ValidateToken`

**Config:** `Auth.TokenDenylist` (bool, default true)

### 6. Request Size Limiting — NEW

**Middleware:** `MaxBodySize(maxBytes int64) func(http.Handler) http.Handler` in `internal/shared/middleware/bodysize.go`

- Wraps `r.Body` with `http.MaxBytesReader`
- Returns `413 Request Entity Too Large` on overflow
- Applied to POST/PUT/PATCH routes only
- Config: `Server.MaxRequestBodySize` (int bytes, default 10485760 = 10MB)

### 7. CORS Configuration — FIX PARTIAL

**Change:** `CORS()` middleware reads from config struct

Config struct `CORSConfig`:
- `AllowedOrigins` ([]string, default `["*"]`)
- `AllowedMethods` ([]string)
- `AllowedHeaders` ([]string)
- `AllowCredentials` (bool)
- `MaxAge` (int)

Env: `CORS_ORIGINS` (comma-separated)

### 8. Structured Logging — FIX PARTIAL

**Change:** `Logger` middleware includes user context from request context

- Read `UserIDKey` and `UserEmailKey` from context (already stored by `Authentication` middleware)
- Add `user_id` and `user_email` fields to log entry (empty string if not authed)

### 9. Global Exception Handling — FIX PARTIAL

**Change:** Replace bare `Recovery` with `ErrorHandler` that:
- Catches panics, logs stack trace, returns 500
- Registers domain errors → HTTP status mappings
- Unregistered errors return 500 with generic message

### 10. OpenAPI Production Guard — FIX PARTIAL

**Change:** Conditionally mount `/swagger/*` in `main.go`

- Config: `Server.AppEnv` (string, default "development"), env: `APP_ENV`
- If `AppEnv == "production"`, skip swagger route mounting

### 11. Liveness Endpoint — FIX PARTIAL

**Change:** Add `GET /live` returning `{"status":"alive"}` in `main.go`

Pure liveness, no dependency checks. `/ready` remains as-is.

### 12. Configuration via Env Vars — FIX PARTIAL

**Changes:**
- Add `github.com/joho/godotenv` to load `.env` file automatically on startup
- Reject startup if `JWT_SECRET == "your-secret-key-change-in-production"` and `APP_ENV == "production"`
- Add `APP_ENV` to config struct and env overrides

### 13. Idempotency Support — NEW

**Middleware:** `Idempotency(rdb *redis.Client)` in `internal/shared/middleware/idempotency.go`

- Checks `Idempotency-Key` header on POST requests only
- Key: `idempotency:{key}` in Redis
- If hit: return cached response body + status code
- If miss: wrap `responseWriter`, cache response on success (2xx), store with TTL
- Config: `Idempotency.Enabled` (bool, default false), `Idempotency.TTL` (int seconds, default 86400)

## Config Additions

```yaml
rate_limit:
  requests: 100
  window: 60

cors:
  allowed_origins: ["*"]
  allowed_methods: ["GET","POST","PUT","PATCH","DELETE","OPTIONS"]
  allowed_headers: ["Accept","Authorization","Content-Type","X-Request-ID"]
  allow_credentials: true
  max_age: 300

auth:
  max_login_attempts: 5
  lockout_duration: 900
  token_denylist: true

server:
  max_request_body_size: 10485760
  app_env: development

idempotency:
  enabled: false
  ttl: 86400
```

## Files Modified

- `internal/shared/middleware/middleware.go` — Logger fix, Recovery→ErrorHandler, CORS from config
- `internal/shared/middleware/ratelimit.go` — NEW
- `internal/shared/middleware/headers.go` — NEW
- `internal/shared/middleware/audit.go` — NEW
- `internal/shared/middleware/bodysize.go` — NEW
- `internal/shared/middleware/idempotency.go` — NEW
- `internal/shared/config/config.go` — new config fields, godotenv, secret rejection
- `internal/authentication/domain/entity/user.go` — lockout fields
- `internal/authentication/application/service/authentication_service.go` — lockout logic, token denylist
- `internal/core/domain/auth.go` — add JTI to TokenClaims
- `internal/infrastructure/auth/jwt.go` — return JTI in claims
- `cmd/api/main.go` — mount new middleware, /live endpoint, conditional swagger
- `migrations/004_add_account_lockout.sql` — NEW
- `configs/config.yaml` — new sections
- `.env.example` — new vars

## Verification

- `go build ./...` passes
- `go vet ./...` passes
- `go test ./...` passes
- All middleware applied in correct order in `main.go`
