# Architecture

## Principles

### Domain-Driven Design (DDD)

The codebase follows DDD with:
- **Entities** - Objects with identity (Todo, User, Role, Permission)
- **Value Objects** - Immutable objects without identity (TodoFilters)
- **Aggregate Roots** - Consistency boundaries (Todo)
- **Domain Services** - Business logic that doesn't belong to a single entity
- **Repository Interfaces** - Abstractions for data access
- **Specifications** - Reusable business rules
- **Domain Events** - Notifications of state changes

### Clean Architecture

Dependency direction is strictly enforced:

```
Interface Layer → Application Layer → Domain Layer ← Infrastructure Layer
```

- **Domain** has zero external dependencies
- **Application** depends only on Domain
- **Infrastructure** implements Domain interfaces
- **Interface** depends on Application

### CQRS (Command Query Responsibility Segregation)

- **Commands** mutate state (Create, Update, Delete, Complete)
- **Queries** read state (Get, List, Search)
- Handlers are separated into `command/` and `query/` packages

### Vertical Slice Architecture

Each feature is a vertical slice:
```
internal/todo/
├── domain/          # Business rules
├── application/     # Use cases
├── infrastructure/  # External concerns
└── interfaces/      # Delivery mechanisms
```

### Modular Monolith

Each module is independent:
- Owns its own database tables
- Communicates through interfaces or domain events
- Registers itself via `fx.Module`
- Can be extracted to a microservice

### Loosely Coupled Infrastructure

All infrastructure implementations are behind interfaces defined in `internal/core/domain/`:
- `Cache` interface → `internal/infrastructure/cache/` (Redis)
- `Messenger` interface → `internal/infrastructure/messaging/` (NATS)
- `TokenService` interface → `internal/infrastructure/auth/` (JWT)
- `Logger` interface → `internal/infrastructure/logger/` (Zap)

To swap a technology, implement the interface and update the `fx.Module` binding.

## Modules

### Todo Module

Example CRUD module demonstrating the full architecture pattern:
- Domain entity, repository interface, domain service
- CQRS command/query handlers
- SQLC persistence, Redis cache
- HTTP handlers with Swagger annotations

### Authentication Module

JWT-based authentication with access and refresh tokens:
- User entity with bcrypt password hashing
- Register, Login, Refresh, Logout endpoints
- Refresh tokens stored in DB (opaque, single-use)
- Access tokens are short-lived JWTs (15 min)

### Authorization Module

Casbin-based RBAC with database-backed policies:
- Role and Permission entities
- User-Role and Role-Permission assignments
- Casbin enforcer loads policies from DB on startup
- Middleware enforces permissions on protected routes
- Check permission endpoint for runtime validation

## Event-Driven & Response Handling

### Event-Driven Email

Services publish domain events to an `EventBus` interface (in-memory synchronous by default, swappable for RabbitMQ/Kafka). An `EmailHandler` subscribes to auth domain events (`auth.user.registered`, `auth.user.email_verified`, `auth.user.password_reset_requested`) and calls the appropriate mailer method.

Flow: `Service → EventBus.Publish() → EmailHandler → domain.Emailer.Send*()`

### Event Error Handling

A `LoggingEventBus` decorator wraps the concrete bus and logs every publish failure through `domain.Logger`. Event handlers return errors instead of swallowing them, so mailer failures (SMTP down, SendGrid error, etc.) are recorded without breaking the originating HTTP request. Services discard the returned publish error after the decorator has logged it, keeping side effects best-effort.

### OpenTelemetry Tracing

Every HTTP request gets an OpenTelemetry span via a Chi middleware. The middleware:
- Extracts incoming W3C trace context (`traceparent`/`baggage` headers)
- Starts a span named `<METHOD> <path>`
- Records the response status code
- Marks the span as error for 4xx/5xx responses

The `ZapLogger` extracts `trace_id` and `span_id` from the context and attaches them to every log entry, so logs and traces are correlated. Panics and 5xx errors are recorded as exceptions on the current span.

### API Response Format

All HTTP responses use a unified envelope:

```json
// Success (single)
{"success": true, "data": {...}, "meta": null}

// Success (paginated list)
{"success": true, "data": [...], "meta": {"page": 1, "per_page": 20, "total": 100, "total_pages": 5}}

// Error
{"success": false, "data": null, "error": {"code": "VALIDATION_ERROR", "message": "..."}}
```

Errors are mapped to HTTP status codes via `utils.MapError`, which translates `domain.ErrNotFound`, `domain.ErrConflict`, etc.

## Module Registration

Every module follows this pattern:

```go
var Module = fx.Module("modulename",
    fx.Provide(/* ... */),
    fx.Invoke(/* ... */),
)
```

The main application composes modules:

```go
app := fx.New(
    fx.Supply(cfg),
    logger.Module,
    cache.Module,
    auth.Module,
    messaging.Module,
    database.Module,
    telemetry.Module,
    authentication.Module,
    authorization.Module,
    todo.Module,
)
```

## Adding a New Module

1. Create `internal/newmodule/` with domain, application, infrastructure, interfaces
2. Define domain entities and repository interfaces
3. Implement application services and handlers
4. Implement infrastructure (persistence, cache)
5. Implement interface (HTTP, gRPC, etc.)
6. Create `module.go` with `fx.Module`
7. Add to `main.go`
