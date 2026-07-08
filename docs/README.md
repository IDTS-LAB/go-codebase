# Go Codebase

A production-ready Golang backend template built with Domain-Driven Design (DDD), Vertical Slice Architecture, CQRS, Clean Architecture, and Modular Monolith principles.

## Features

- **DDD** - Entities, Value Objects, Domain Services, Specifications, Domain Events
- **CQRS** - Separate Command and Query handlers
- **Clean Architecture** - Strict dependency direction: Interface → Application → Domain ← Infrastructure
- **Modular Monolith** - Each module is independent with its own domain, application, infrastructure, and interface layers
- **Vertical Slice** - Each feature is a self-contained slice through all layers

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
| Auth | JWT |
| Cache | Redis |
| Messaging | NATS |
| Tracing | OpenTelemetry |
| Testing | Testify + GoMock |
| Container | Docker + Docker Compose |

## Quick Start

```bash
# Start all services
make docker-up

# Run migrations
make migrate-up

# Generate SQLC code
make sqlc

# Start the API server
make run
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/todos | Create a todo |
| GET | /api/v1/todos | List all todos |
| GET | /api/v1/todos/{id} | Get a todo by ID |
| PUT | /api/v1/todos/{id} | Update a todo |
| DELETE | /api/v1/todos/{id} | Delete a todo |
| PATCH | /api/v1/todos/{id}/complete | Complete a todo |
| GET | /api/v1/todos/search?q= | Search todos |
| GET | /health | Health check |
| GET | /ready | Readiness check |

## Project Structure

See [FolderStructure.md](FolderStructure.md)

## Architecture

See [Architecture.md](Architecture.md)

## API Documentation

See [API.md](API.md)

## Development

See [Development.md](Development.md)

## Deployment

See [Deployment.md](Deployment.md)
