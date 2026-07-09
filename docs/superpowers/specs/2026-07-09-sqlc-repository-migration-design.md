# sqlc Repository Migration Design

**Goal:** Migrate all domain repository implementations from hand-written `database/sql` queries to sqlc-generated code, keeping the existing domain interfaces, Fx wiring, and clean architecture intact.

## Context

The codebase already has `sqlc.yaml` (scoped to the todo domain only) and a `make sqlc` target, but the generated code was never wired in. All 9 repository implementations (42 methods) use raw `database/sql` with inline SQL strings. This migration replaces the inline SQL with sqlc-generated query methods while preserving the existing repository interface contracts.

## Architecture

### sqlc Configuration

Rewrite `sqlc.yaml` with one `sql` block per domain — each produces a separate generated package. All blocks read schema from the shared `migrations/` directory.

```yaml
version: "2"
sql:
  - schema: "migrations"
    queries: "internal/todo/infrastructure/persistence/queries/todo.sql"
    gen:
      go:
        package: "sqlc"
        out: "internal/todo/infrastructure/persistence/sqlc"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_empty_slices: true

  - schema: "migrations"
    queries: "internal/authentication/infrastructure/persistence/queries/user.sql"
    gen:
      go:
        package: "sqlc"
        out: "internal/authentication/infrastructure/persistence/sqlc"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_empty_slices: true

  # ... one block per domain (authorization role/permission/etc, user, auditlog)
```

Each `sql` block can only point to one `queries` path. Domains with multiple query files (e.g. authorization has role.sql, permission.sql, role_permission.sql, user_role.sql) get one `sql` block per query file, all outputting to the same domain `sqlc/` package — OR the queries are consolidated into a single `queries.sql` per domain. We consolidate: one `queries.sql` per domain (or per sub-package for authorization), one `sql` block per domain.

### Query Files

One consolidated `queries.sql` per domain (in a `queries/` subdirectory of each persistence package):
- `internal/todo/infrastructure/persistence/queries/todo.sql` (replaces existing `queries.sql`)
- `internal/authentication/infrastructure/persistence/queries/queries.sql` (user + refresh_token queries)
- `internal/authorization/infrastructure/persistence/queries/queries.sql` (role + permission + role_permission + user_role queries)
- `internal/user/infrastructure/persistence/queries/queries.sql`
- `internal/shared/auditlog/queries/queries.sql`

Each `.sql` file contains the `-- name: MethodName :one/:many/:exec` annotations that sqlc parses. The SQL mirrors the exact queries currently inlined in the hand-written repositories.

### Generated Output

Each domain gets a `sqlc/` subdirectory in its persistence package:
- `internal/todo/infrastructure/persistence/sqlc/`
- `internal/authentication/infrastructure/persistence/sqlc/`
- `internal/authorization/infrastructure/persistence/sqlc/`
- `internal/user/infrastructure/persistence/sqlc/`
- `internal/shared/auditlog/sqlc/`

All `sqlc/` directories are gitignored (regenerated via `make sqlc`), matching the existing convention for generated code.

### JSONB Handling (pqtype avoidance)

The generated `models.go` would import `github.com/sqlc-dev/pqtype` because `audit_logs.metadata` and `error_logs.metadata` are `JSONB`. To avoid adding a new dependency, cast the JSONB columns to `[]byte` in the query SQL:

```sql
-- name: InsertAuditLog :exec
INSERT INTO audit_logs (..., metadata) VALUES (..., $N::jsonb);

-- name: GetAuditLog :one
SELECT id, ..., metadata::text as metadata FROM audit_logs WHERE id = $1;
```

This makes sqlc generate `[]byte` (or `string`) fields instead of `pqtype.JSONValue`.

### Repository Refactoring Pattern

Each existing repository implementation gets refactored to delegate to sqlc's generated `Queries` struct:

```go
type userRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
    q := sqlc.New(r.db)
    _, err := q.CreateUser(ctx, sqlc.CreateUserParams{
        ID: user.ID, Email: user.Email, ...
    })
    return err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
    q := sqlc.New(r.db)
    row, err := q.GetUserByEmail(ctx, email)
    if err != nil { return nil, fmt.Errorf("get user by email: %w", err) }
    return mapRowToUser(row), nil
}
```

### Mapping Helpers

A `mapRowTo<Entity>(row sqlc.<RowStruct>) *entity.<Entity>` helper per repository converts generated structs to domain entities. This keeps the domain layer decoupled from sqlc's generated types. Generated structs (e.g. `sqlc.User`) are transport types; domain entities (`entity.User`) are the canonical types.

### Transactions

For multi-operation transactions, the `internal/shared/transaction` package provides a `*sql.Tx`. Since `*sql.Tx` satisfies sqlc's `DBTX` interface, repos build `sqlc.New(tx)` when inside a transaction context. The per-method `sqlc.New(r.db)` pattern means non-transactional calls use the pool directly, and transactional calls pass the `*sql.Tx`.

### Fx Wiring

Unchanged. Constructors still take `*sql.DB`, return the domain interface. Fx does not see sqlc. The `database.Module` continues to provide the single `*sql.DB`.

### Domain Interfaces

All existing interfaces stay exactly as they are:
- `internal/core/domain/repository.go` (generic `Repository[T]`)
- `internal/user/domain/repository/user_repository.go`
- `internal/authentication/domain/repository/authentication_repository.go`
- `internal/authorization/domain/repository/authorization_repository.go`
- `internal/todo/domain/repository/todo_repository.go`

Only the implementations change internally.

## Migration Order

Least risk first:

1. **Foundation:** Rewrite `sqlc.yaml`, add `sqlc/` dirs to `.gitignore`, document the workflow. Verify `make sqlc` runs clean.
2. **todo:** Migrate `todo_repository.go` to wrap sqlc. Replace existing `queries.sql` with `queries/todo.sql`.
3. **authentication:** Migrate `user_repository.go` + `refresh_token_repository.go` (includes email verification fields).
4. **authorization:** Migrate 4 repos (role, permission, role_permission, user_role).
5. **user:** Migrate `user_repository.go`.
6. **shared/auditlog:** Migrate `repository.go` (includes JSONB handling).
7. **Verification:** `make sqlc` + full build + test suite + vet.

## What Does NOT Change

- Domain interfaces, entities, services, handlers, routes, Fx modules
- Migrations (we only READ schema, we don't change it)
- Config, database connection provider
- The existing `internal/shared/transaction` package (sqlc is compatible with it)

## Verification

- `make sqlc` regenerates all domains cleanly
- `go build ./...` passes after each domain
- `go test ./...` passes at the end
- `go vet ./...` clean
- No new dependencies added to `go.mod` (pqtype avoided via cast)
