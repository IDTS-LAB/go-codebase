# Folder Structure

```
cmd/
    api/                     # Application entrypoint
        main.go              # fx wiring, root router, server startup
        swagger.go           # Swagger API metadata annotations

configs/                     # Configuration files
    config.yaml

.github/                     # GitHub Actions workflows
    workflows/
        ci.yml                 # CI pipeline
        cd.yml                 # CD pipeline

deployments/                 # Deployment configs
    prometheus.yml

k8s/                         # Kubernetes manifests
    base/                    # Base Kustomize resources
    overlays/                # Environment overlays (staging, production)

docs/                        # Documentation
    README.md
    Architecture.md
    API.md
    FolderStructure.md
    Development.md
    Deployment.md
    Contributing.md
    docs.go                  # Generated swagger Go code
    swagger.json             # Generated swagger JSON
    swagger.yaml             # Generated swagger YAML

migrations/                  # Database migrations
    001_create_todos.sql
    002_create_rbac_tables.sql
    003_create_users_and_refresh_tokens.sql

scripts/                     # Utility scripts
    migrate.sh               # Database migration helper
    seed.sh                  # Seed default roles and permissions

pkg/                         # Public packages
    password/                # Bcrypt password hashing
    slug/                    # URL-safe slug generation

internal/
    core/                    # Core interfaces (no dependencies)
        domain/              # Shared interfaces and types
            entity.go        # Entity, AggregateRoot, Identifier, Timestamp
            errors.go        # Domain errors
            repository.go    # Repository[T] generic interface
            cache.go         # Cache interface
            messenger.go     # Messenger interface
            token.go         # TokenService interface
            logger.go        # Logger interface

    shared/                  # Shared kernel (cross-module)
        config/              # Koanf configuration loading
        database/            # PostgreSQL connection, sqlc, goose
        events/              # EventBus interface + InMemoryEventBus + LoggingEventBus + Fx module
        middleware/           # HTTP middleware
            recovery.go      # Panic recovery
            request_id.go    # Request ID propagation
            logger.go        # Request logging
            authentication.go # JWT auth middleware
            authorization.go # Casbin RBAC middleware
            cors.go          # CORS headers
            timeout.go       # Request timeout
        validator/           # Input validation
        telemetry/           # OpenTelemetry setup
        utils/               # Response helpers (RespondSuccess, RespondError)

    infrastructure/          # Loosely coupled implementations
        auth/                # JWT token service (behind TokenService interface)
        cache/               # Redis cache (behind Cache interface)
        logger/              # Zap logger (behind Logger interface)
        messaging/           # NATS messenger (behind Messenger interface)

    todo/                    # Todo module (example)
        module.go            # fx.Module registration

        domain/              # Business rules
            entity/          # Entities (Todo)
            repository/      # Repository interfaces
            valueobject/     # Value objects (TodoFilters)
            service/         # Domain services
            specification/   # Business rules
            event/           # Domain events

        application/         # Use cases
            command/         # Command handlers (Create, Update, Delete, Complete)
            query/           # Query handlers (Get, List, Search)
            dto/             # Data transfer objects
            service/         # Application services

        infrastructure/      # External implementations
            persistence/     # SQLC repositories
            cache/           # Redis cache
            eventbus/        # Domain event handlers

        interfaces/          # Delivery mechanisms
            http/            # HTTP handlers and routes (Swagger annotated)

        tests/               # Module tests

    authentication/          # Auth module
        module.go            # fx.Module registration

        domain/              # Business rules
            entity/          # Entities (User, RefreshToken)
            repository/      # Repository interfaces
            service/         # Auth domain service
            event/           # Domain events (UserRegistered, EmailVerified, PasswordResetRequested)

        application/         # Use cases
            dto/             # DTOs (RegisterRequest, LoginRequest, TokenResponse)
            service/         # Auth application service

        infrastructure/      # External implementations
            persistence/     # SQLC repositories
            eventbus/        # Event handler implementations (EmailHandler)

        interfaces/          # Delivery mechanisms
            http/            # HTTP handlers (Register, Login, Refresh, Logout, Me)

    authorization/           # RBAC module
        module.go            # fx.Module registration

        domain/              # Business rules
            entity/          # Entities (Role, Permission, UserRole, RolePermission)
            repository/      # Repository interfaces

        application/         # Use cases
            dto/             # DTOs (CreateRoleRequest, AssignRoleRequest)
            service/         # Authorization service

        infrastructure/      # External implementations
            persistence/     # SQLC repositories
            casbin/          # Casbin enforcer (DB-backed policies)

        interfaces/          # Delivery mechanisms
            http/            # HTTP handlers (CRUD roles/permissions, assignments)

test/                        # Integration tests
```

## Adding a New Module

1. Create directory: `internal/mymodule/`
2. Create subdirectories: `domain/`, `application/`, `infrastructure/`, `interfaces/`
3. Create `module.go` with `fx.Module`
4. Add module to `main.go`
