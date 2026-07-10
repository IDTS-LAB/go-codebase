# Go Codebase

A production-ready Golang backend template built with **Domain-Driven Design (DDD)**, **Vertical Slice Architecture**, **CQRS**, **Clean Architecture**, and **Modular Monolith** principles.

## Quick Start

```bash
# Start all services (PostgreSQL, Redis, NATS, Jaeger, Prometheus, Grafana)
make docker-up

# Run database migrations
make migrate-up

# Generate SQLC code
make sqlc

# Seed default roles and permissions
make seed

# Start the API server
make run
```

The server starts at `http://localhost:8080`.

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Health check**: `http://localhost:8080/health`

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.22+ |
| HTTP | Chi Router |
| DI | Uber Fx |
| Database | PostgreSQL |
| Query | SQLC |
| Migration | Goose |
| Config | Koanf |
| Logging | Zap |
| Validation | go-playground/validator |
| Auth | JWT (access + refresh tokens) |
| RBAC | Casbin |
| Cache | Redis |
| Messaging | NATS |
| Events | In-memory EventBus (interface; RabbitMQ/Kafka-ready) |
| Tracing | OpenTelemetry |
| Docs | Swagger (swaggo/swag) |
| Testing | Testify + GoMock |
| Container | Docker + Docker Compose |

## Architecture

- **DDD** - Entities, Value Objects, Domain Services, Specifications, Domain Events
- **CQRS** - Separate Command and Query handlers
- **Clean Architecture** - Strict dependency direction
- **Modular Monolith** - Each module is independent
- **Vertical Slice** - Each feature is self-contained
- **Loosely Coupled** - Infrastructure implementations behind interfaces, swappable via DI
- **Event-Driven** - Domain events decouple side effects (e.g., email sending) from core business logic
- **Unified API** - Consistent response envelope and centralized error handling across all endpoints

## API

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/register | Register a new user |
| POST | /api/v1/auth/login | Login and get tokens |
| POST | /api/v1/auth/refresh | Refresh access token |
| POST | /api/v1/auth/logout | Revoke refresh token |
| GET | /api/v1/auth/verify-email?token= | Verify email address |
| POST | /api/v1/auth/forgot-password | Request password reset |
| POST | /api/v1/auth/reset-password | Reset password with token |
| POST | /api/v1/auth/resend-verification | Resend verification email |

### Protected (requires Bearer token)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/auth/sessions/me | Get current user profile |
| POST | /api/v1/auth/sessions/logout-all | Logout all sessions |

### Todos

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/todos | Create a todo |
| GET | /api/v1/todos | List all todos |
| GET | /api/v1/todos/{id} | Get a todo |
| PUT | /api/v1/todos/{id} | Update a todo |
| DELETE | /api/v1/todos/{id} | Delete a todo |
| PATCH | /api/v1/todos/{id}/complete | Complete a todo |
| GET | /api/v1/todos/search?q= | Search todos |

### Users

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/users | List users (paginated) |
| GET | /api/v1/users/{id} | Get a user |
| PUT | /api/v1/users/{id} | Update a user |
| DELETE | /api/v1/users/{id} | Delete a user |

### Authorization (RBAC)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/roles | Create a role |
| GET | /api/v1/auth/roles | List roles |
| GET | /api/v1/auth/roles/{id} | Get a role |
| PUT | /api/v1/auth/roles/{id} | Update a role |
| DELETE | /api/v1/auth/roles/{id} | Delete a role |
| POST | /api/v1/auth/permissions | Create a permission |
| GET | /api/v1/auth/permissions | List permissions |
| GET | /api/v1/auth/permissions/{id} | Get a permission |
| PUT | /api/v1/auth/permissions/{id} | Update a permission |
| DELETE | /api/v1/auth/permissions/{id} | Delete a permission |
| POST | /api/v1/auth/users/{userId}/roles | Assign role to user |
| DELETE | /api/v1/auth/users/{userId}/roles/{roleId} | Remove role from user |
| GET | /api/v1/auth/users/{userId}/roles | Get user roles |
| POST | /api/v1/auth/roles/{roleId}/permissions | Assign permission to role |
| DELETE | /api/v1/auth/roles/{roleId}/permissions/{permissionId} | Remove permission from role |
| GET | /api/v1/auth/roles/{roleId}/permissions | Get role permissions |
| POST | /api/v1/auth/check-permission | Check user permission |

### Infrastructure

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /health | Health check |
| GET | /ready | Readiness check |
| GET | /swagger/* | Swagger UI |

## Project Structure

```
cmd/api/              - Entry point, router, swagger metadata
internal/
  core/domain/        - Shared interfaces (Entity, Cache, Messenger, TokenService, Logger)
  shared/             - Shared kernel (config, database, middleware, telemetry, utils)
  infrastructure/     - Loosely coupled implementations (Redis, NATS, JWT, Zap, Email)
  shared/             - Shared kernel (config, database, events, middleware, telemetry, utils)
  todo/               - Todo module (domain, application, infrastructure, interfaces)
  authentication/     - Auth module (register, login, refresh, logout, email verification)
  authorization/      - RBAC module (Casbin enforcer, roles, permissions)
  user/               - User module (admin user CRUD)
migrations/           - Database migrations
docs/                 - Documentation + generated Swagger output
pkg/                  - Public packages (password, slug)
scripts/              - Utility scripts (migrate, seed)
configs/              - Configuration files
```

## Response Format

All API responses share a unified envelope:

```json
// Success (single resource)
{
  "success": true,
  "data": { ... },
  "meta": null
}

// Success (paginated list)
{
  "success": true,
  "data": [ ... ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}

// Error
{
  "success": false,
  "data": null,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "..."
  }
}
```

See [docs/API.md](docs/API.md) for the full list of error codes.

## Event-Driven Email

Email sending is decoupled from the authentication service through domain events. The service publishes `UserRegistered`, `EmailVerified`, and `PasswordResetRequested` events; an `EmailHandler` subscribes and calls the configured mailer (SMTP, SendGrid, or console). The `EventBus` is interface-based so an async adapter (RabbitMQ, Kafka, etc.) can be swapped in without changing services or handlers.

## Documentation

- [Architecture](docs/Architecture.md)
- [API](docs/API.md)
- [Folder Structure](docs/FolderStructure.md)
- [Development](docs/Development.md)
- [Deployment](docs/Deployment.md)
- [Contributing](docs/Contributing.md)

## License

MIT
