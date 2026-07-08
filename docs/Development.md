# Development

## Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Make
- SQLC (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)
- Goose (`go install github.com/pressly/goose/v3/cmd/goose@latest`)
- Swag (`go install github.com/swaggo/swag/cmd/swag@latest`)

## Setup

```bash
# Clone the repository
git clone <repo-url>
cd go-codebase

# Install dependencies
go mod tidy

# Start infrastructure
make docker-up

# Run migrations
make migrate-up

# Seed default roles and permissions
make seed

# Generate SQLC code
make sqlc

# Start the server
make run
```

The server starts at `http://localhost:8080`.

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Health check**: `http://localhost:8080/health`

## Available Commands

| Command | Description |
|---------|-------------|
| `make run` | Start the API server |
| `make build` | Build the binary |
| `make test` | Run all tests |
| `make test-coverage` | Run tests with coverage |
| `make lint` | Run linter |
| `make fmt` | Format code |
| `make migrate-up` | Run database migrations |
| `make migrate-down` | Rollback migrations |
| `make sqlc` | Generate SQLC code |
| `make swagger` | Generate Swagger docs |
| `make seed` | Seed default roles and permissions |
| `make docker-up` | Start Docker services |
| `make docker-down` | Stop Docker services |

## Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test ./internal/todo/...
```

## Configuration

Configuration is loaded from (in order):
1. `configs/config.yaml`
2. Environment variables

Environment variables override config file values. See `.env.example` for available variables.

## Code Generation

### SQLC

After modifying SQL queries in `queries.sql`:

```bash
make sqlc
```

### Swagger

After modifying swagger annotations in handler files:

```bash
make swagger
```

This regenerates `docs/docs.go`, `docs/swagger.json`, and `docs/swagger.yaml`.

### Goose Migrations

Create a new migration:

```bash
goose -dir migrations create <name> sql
```

## Authentication Setup

1. Register a user via `POST /api/v1/auth/register`
2. Use the returned access token in the `Authorization: Bearer <token>` header
3. When the token expires, call `POST /api/v1/auth/refresh` with the refresh token

## RBAC Setup

1. Create roles via `POST /api/v1/auth/roles`
2. Create permissions via `POST /api/v1/auth/permissions`
3. Assign permissions to roles via `POST /api/v1/auth/roles/{roleId}/permissions`
4. Assign roles to users via `POST /api/v1/auth/users/{userId}/roles`
5. The seed script creates default roles: `admin`, `user`

## Adding a New Module

See [Architecture.md](Architecture.md#adding-a-new-module)
