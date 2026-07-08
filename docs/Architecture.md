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
