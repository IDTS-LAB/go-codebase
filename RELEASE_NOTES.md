# Release Notes

> Commits since master (`2c51273`): 108

## NATS Event Bus (New)

Switchable EventBus between in-memory and NATS JetStream via `events.driver` config:

- `events.driver: memory` (default) — existing synchronous in-memory bus, unchanged
- `events.driver: nats` — JetStream-backed with at-least-once delivery, queue groups, infinite retry on handler error

**Components:**
- Config structs: `EventsConfig`, `StreamConfig`, `ConsumerConfig` in `internal/shared/config/config.go`
- Type registry: `Register`/`CreatePayload` for JSON deserialization (`internal/shared/events/registry.go`)
- `NATSEventBus` with `jetStreamer` abstraction, adapters for `nats.JetStreamContext`/`*nats.Msg`
- JetStream stream `"events"` (file storage, interest retention) created at startup via `ensureStream()`
- Push consumer `"event-bus"` with queue group, `MaxDeliver(-1)`, `AckWait(30s)`, manual ACK
- Auth event structs gained JSON tags for serialization
- Events registered via `fx.Invoke` in `todo/module.go` and `authentication/module.go`
- New `NewLoggingEventBus` accepts `EventBus` interface (wraps both implementations)
- Docker Compose enables `-js` flag on NATS server

## NATS Observability (New)

- Prometheus counters per NATS subject (`nats_published_total`, `nats_received_total`, etc.)
- NATS debug endpoint with in-memory ring buffer (toggled via `nats.debug_endpoint`)
- NATS Grafana dashboard (`dashboards/nats.json`) with publish/receive rates, bytes, and debug snapshots
- Infinity datasource for NATS debug endpoint in Grafana

## Observability Stack (New)

- Prometheus, Grafana, Alertmanager in Docker Compose
- Grafana auto-provisioning: datasources and dashboards
- OpenTelemetry HTTP tracing middleware with span error recording
- Trace IDs propagated to logs

## CQRS Standardization (Refactor)

- `CommandBus`/`QueryBus` interfaces and in-memory implementations in `internal/shared/cqrs`
- Auth, user, authorization, and tenant modules migrated to bus dispatch
- Auth handler tests rewritten for bus dispatch pattern
- Dead code (old application service directories) removed

## Response Formatter & Middleware (Refactor)

- Unified response envelope (`utils.APIResponse`)
- `MapError` for standardized error codes
- `RespondPaginated` for cursor-based paginated responses
- Pure-function handler adapters in `httpadapter` package
- Response formatter middleware

## SQLC Migration (Refactor)

- Multi-domain sqlc configuration with 5 generation targets
- All repositories migrated from `database/sql` to sqlc:
  - Todo, user, authentication (user + refresh_token), authorization (role, permission, role_permission, user_role), auditlog
- Pagination queries removed from sqlc (kept only CRUD)
- JSONB override for `[]byte` in sqlc
- Cursor-based pagination with shared `cursor` package (`CursorMeta`, `Before`, `After`)

## Multi-Tenancy (New)

- Row-level isolation with tenant context
- User normalization per tenant
- Config, context keys, and tenant resolver middleware
- `TenantConfig` with `X-Tenant-ID` header and JWT claim support

## Email Service (New)

- Domain interface (`domain.Mailer`) with multiple providers
- Console, SMTP, and SendGrid providers with tests
- HTML template rendering with templates for verification, reset, welcome, invite
- Email verification and password reset flows in auth module
- Domain events (`UserRegistered`, `EmailVerified`, `PasswordResetRequested`) decouple auth from email
- `LoggingEventBus` decorator for event publish error logging

## Error Hardening (Fix)

- Stack trace hidden on non-production 500 errors
- Error checking improvements
- Security fixes: SMTP STARTTLS, token hashing, template caching
- Remove pqtype dependency
- Sentry-style error handling patterns

## CI/CD & Deployment

- Production Dockerfile and `docker-compose.prod`
- GitHub Actions workflow
- Kubernetes manifests
- Go 1.25 updated
- golangci-lint configuration fixes
- Script install hooks and pre-commit hooks

## Other

- Cursor pagination in all repository implementations
- Shared pagination utilities (`PaginatedPayload`, `PaginatedResult`)
- Swagger/OpenAPI doc annotations updated
- Updated Fx wiring for shared EventBus and email handler
- Architecture, folder structure, and API documentation updates
