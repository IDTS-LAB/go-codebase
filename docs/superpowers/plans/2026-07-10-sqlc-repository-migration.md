# SQLc Repository Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate all 9 hand-written `database/sql` repository implementations (42 methods) across 5 domains to sqlc-generated code while preserving all domain interfaces, entities, Fx wiring, and clean architecture.

**Architecture:** One sqlc `sql` block per domain queries file, each outputting to its own `sqlc/` generated package. Repositories wrap `sqlc.New(r.db)` per method with mapping helpers to convert generated structs to domain entities. Generated code is gitignored and regenerated via `make sqlc`.

**Tech Stack:** Go 1.25, sqlc v2, PostgreSQL 16 (via `lib/pq`), `database/sql` driver, `emit_pointers_for_null_types: true`, `emit_json_tags: true`, `emit_empty_slices: true`

## Global Constraints

- Module path: `github.com/IDTS-LAB/go-codebase`
- Do NOT modify any domain entity files (`internal/*/domain/entity/*.go`)
- Do NOT modify any domain repository interface files (`internal/*/domain/repository/*.go`)
- Do NOT modify any Fx module files (`internal/*/module.go`, `cmd/api/main.go`)
- Do NOT modify migration files (`migrations/*.sql`)
- Do NOT add `github.com/sqlc-dev/pqtype` dependency — use `::jsonb` casts for JSONB columns
- All generated `sqlc/` directories are gitignored
- Every task ends with `go build ./...` and a commit
- SQL queries use `-- name: PascalCase :one/:many/:exec` annotations exactly matching method semantics

---

## File Structure

| File | Status | Responsibility |
|------|--------|---------------|
| `sqlc.yaml` | **Modify** | Multi-domain sqlc config, one `sql` block per queries file |
| `.gitignore` | **Modify** | Add `**/infrastructure/persistence/sqlc/` and `internal/shared/auditlog/sqlc/` |
| `internal/todo/infrastructure/persistence/queries.sql` | **Delete** | Replaced by `queries/todo.sql` |
| `internal/todo/infrastructure/persistence/queries/todo.sql` | **Create** | Todo-domain sqlc queries |
| `internal/todo/infrastructure/persistence/todo_repository.go` | **Modify** | Refactor to wrap sqlc |
| `internal/authentication/infrastructure/persistence/queries/queries.sql` | **Create** | Auth-domain sqlc queries (user + refresh_token) |
| `internal/authentication/infrastructure/persistence/user_repository.go` | **Modify** | Refactor to wrap sqlc |
| `internal/authentication/infrastructure/persistence/refresh_token_repository.go` | **Modify** | Refactor to wrap sqlc |
| `internal/authorization/infrastructure/persistence/queries/queries.sql` | **Create** | Authorization-domain sqlc queries (role, permission, role_permission, user_role) |
| `internal/authorization/infrastructure/persistence/role_repository.go` | **Modify** | Refactor to wrap sqlc |
| `internal/authorization/infrastructure/persistence/permission_repository.go` | **Modify** | Refactor to wrap sqlc |
| `internal/authorization/infrastructure/persistence/role_permission_repository.go` | **Modify** | Refactor to wrap sqlc |
| `internal/authorization/infrastructure/persistence/user_role_repository.go` | **Modify** | Refactor to wrap sqlc |
| `internal/user/infrastructure/persistence/queries/queries.sql` | **Create** | User-domain sqlc queries |
| `internal/user/infrastructure/persistence/user_repository.go` | **Modify** | Refactor to wrap sqlc |
| `internal/shared/auditlog/queries/queries.sql` | **Create** | Auditlog sqlc queries (JSONB handling via `::jsonb`) |
| `internal/shared/auditlog/repository.go` | **Modify** | Refactor to wrap sqlc |

**Generated (gitignored):**

| Directory | Contents |
|-----------|----------|
| `internal/todo/infrastructure/persistence/sqlc/` | Generated from `queries/todo.sql` |
| `internal/authentication/infrastructure/persistence/sqlc/` | Generated from `queries/queries.sql` |
| `internal/authorization/infrastructure/persistence/sqlc/` | Generated from `queries/queries.sql` |
| `internal/user/infrastructure/persistence/sqlc/` | Generated from `queries/queries.sql` |
| `internal/shared/auditlog/sqlc/` | Generated from `queries/queries.sql` |

---

### Task 1: Foundation — sqlc.yaml rewrite, .gitignore, make sqlc

**Files:**
- Modify: `sqlc.yaml` — rewrite with all 5 domain sql blocks
- Modify: `.gitignore` — add sqlc/ directories

- [ ] **Step 1: Rewrite `sqlc.yaml`**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/todo/infrastructure/persistence/queries/todo.sql"
    schema: "migrations"
    gen:
      go:
        package: "sqlc"
        out: "internal/todo/infrastructure/persistence/sqlc"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_empty_slices: true
        emit_pointers_for_null_types: true

  - engine: "postgresql"
    queries: "internal/authentication/infrastructure/persistence/queries/queries.sql"
    schema: "migrations"
    gen:
      go:
        package: "sqlc"
        out: "internal/authentication/infrastructure/persistence/sqlc"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_empty_slices: true
        emit_pointers_for_null_types: true

  - engine: "postgresql"
    queries: "internal/authorization/infrastructure/persistence/queries/queries.sql"
    schema: "migrations"
    gen:
      go:
        package: "sqlc"
        out: "internal/authorization/infrastructure/persistence/sqlc"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_empty_slices: true
        emit_pointers_for_null_types: true

  - engine: "postgresql"
    queries: "internal/user/infrastructure/persistence/queries/queries.sql"
    schema: "migrations"
    gen:
      go:
        package: "sqlc"
        out: "internal/user/infrastructure/persistence/sqlc"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_empty_slices: true
        emit_pointers_for_null_types: true

  - engine: "postgresql"
    queries: "internal/shared/auditlog/queries/queries.sql"
    schema: "migrations"
    gen:
      go:
        package: "sqlc"
        out: "internal/shared/auditlog/sqlc"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_empty_slices: true
        emit_pointers_for_null_types: true
```

- [ ] **Step 2: Update `.gitignore`**

Add after the swagger section:

```
# SQLc generated code (regenerate with `make sqlc`)
**/infrastructure/persistence/sqlc/
internal/shared/auditlog/sqlc/
```

- [ ] **Step 3: Create query directories**

```bash
mkdir -p internal/todo/infrastructure/persistence/queries
mkdir -p internal/authentication/infrastructure/persistence/queries
mkdir -p internal/authorization/infrastructure/persistence/queries
mkdir -p internal/user/infrastructure/persistence/queries
mkdir -p internal/shared/auditlog/queries
```

- [ ] **Step 4: Create minimal placeholder query files for all domains except todo (which gets real queries in Task 2)**

Create each file with a single placeholder query so `make sqlc` succeeds:

`internal/authentication/infrastructure/persistence/queries/queries.sql`:
```sql
-- name: Ping :one
SELECT 1;
```

`internal/authorization/infrastructure/persistence/queries/queries.sql`:
```sql
-- name: Ping :one
SELECT 1;
```

`internal/user/infrastructure/persistence/queries/queries.sql`:
```sql
-- name: Ping :one
SELECT 1;
```

`internal/shared/auditlog/queries/queries.sql`:
```sql
-- name: Ping :one
SELECT 1;
```

- [ ] **Step 5: Run `make sqlc`**

```bash
make sqlc
```

Expected: Clean exit, generates 5 sqlc packages.

- [ ] **Step 6: Verify build still passes**

```bash
go build ./...
```

Expected: Clean build. The existing hand-written repos still work, and the new sqlc packages exist but aren't imported yet.

- [ ] **Step 7: Commit**

```bash
git add sqlc.yaml .gitignore internal/todo/infrastructure/persistence/queries/ internal/authentication/infrastructure/persistence/queries/ internal/authorization/infrastructure/persistence/queries/ internal/user/infrastructure/persistence/queries/ internal/shared/auditlog/queries/ internal/todo/infrastructure/persistence/sqlc/ internal/authentication/infrastructure/persistence/sqlc/ internal/authorization/infrastructure/persistence/sqlc/ internal/user/infrastructure/persistence/sqlc/ internal/shared/auditlog/sqlc/
git commit -m "feat: configure multi-domain sqlc with 5 generation targets"
```

---

### Task 2: Migrate Todo Domain

**Files:**
- Create: `internal/todo/infrastructure/persistence/queries/todo.sql` — real todo queries
- Delete: `internal/todo/infrastructure/persistence/queries.sql` — old single-file
- Modify: `internal/todo/infrastructure/persistence/todo_repository.go` — refactor to sqlc

- [ ] **Step 1: Move existing queries to new location**

Read the existing `internal/todo/infrastructure/persistence/queries.sql` and write it to `internal/todo/infrastructure/persistence/queries/todo.sql` (content is identical):

```sql
-- name: CreateTodo :exec
INSERT INTO todos (id, title, description, completed, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetTodoByID :one
SELECT id, title, description, completed, created_at, updated_at, deleted_at
FROM todos
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListTodos :many
SELECT id, title, description, completed, created_at, updated_at, deleted_at
FROM todos
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountTodos :one
SELECT COUNT(*) FROM todos WHERE deleted_at IS NULL;

-- name: UpdateTodo :exec
UPDATE todos
SET title = $2, description = $3, completed = $4, updated_at = $5
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteTodo :exec
UPDATE todos
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: SearchTodos :many
SELECT id, title, description, completed, created_at, updated_at, deleted_at
FROM todos
WHERE deleted_at IS NULL AND (title ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSearchTodos :one
SELECT COUNT(*) FROM todos
WHERE deleted_at IS NULL AND (title ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%');
```

- [ ] **Step 2: Delete old queries.sql**

```bash
rm internal/todo/infrastructure/persistence/queries.sql
```

- [ ] **Step 3: Run `make sqlc` to regenerate**

```bash
make sqlc
```

Expected: Clean exit. Todo sqlc package now has real query methods.

- [ ] **Step 4: Rewrite `internal/todo/infrastructure/persistence/todo_repository.go`**

```go
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/todo/infrastructure/persistence/sqlc"
	"github.com/google/uuid"
)

type todoRepository struct {
	db *sql.DB
}

func NewTodoRepository(db *sql.DB) repository.TodoRepository {
	return &todoRepository{db: db}
}

func (r *todoRepository) Create(ctx context.Context, todo *entity.Todo) error {
	q := sqlc.New(r.db)
	err := q.CreateTodo(ctx, sqlc.CreateTodoParams{
		ID:          todo.ID,
		Title:       todo.Title,
		Description: todo.Description,
		Completed:   todo.Completed,
		CreatedAt:   todo.CreatedAt,
		UpdatedAt:   todo.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert todo: %w", err)
	}
	return nil
}

func (r *todoRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Todo, error) {
	q := sqlc.New(r.db)
	row, err := q.GetTodoByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("todo not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get todo: %w", err)
	}
	return mapSqlcTodoToEntity(row), nil
}

func (r *todoRepository) GetAll(ctx context.Context, offset, limit int) ([]*entity.Todo, int, error) {
	q := sqlc.New(r.db)

	total, err := q.CountTodos(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count todos: %w", err)
	}

	rows, err := q.ListTodos(ctx, sqlc.ListTodosParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("query todos: %w", err)
	}

	todos := make([]*entity.Todo, len(rows))
	for i, row := range rows {
		todos[i] = mapSqlcTodoToEntity(row)
	}
	return todos, int(total), nil
}

func (r *todoRepository) Update(ctx context.Context, todo *entity.Todo) error {
	q := sqlc.New(r.db)
	result, err := q.UpdateTodo(ctx, sqlc.UpdateTodoParams{
		ID:          todo.ID,
		Title:       todo.Title,
		Description: todo.Description,
		Completed:   todo.Completed,
		UpdatedAt:   todo.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("update todo: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("todo not found")
	}
	return nil
}

func (r *todoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := sqlc.New(r.db)
	result, err := q.SoftDeleteTodo(ctx, id)
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("todo not found")
	}
	return nil
}

func (r *todoRepository) Search(ctx context.Context, query string, offset, limit int) ([]*entity.Todo, int, error) {
	q := sqlc.New(r.db)
	searchPattern := "%" + query + "%"

	total, err := q.CountSearchTodos(ctx, searchPattern)
	if err != nil {
		return nil, 0, fmt.Errorf("count search results: %w", err)
	}

	rows, err := q.SearchTodos(ctx, sqlc.SearchTodosParams{
		Column1: searchPattern,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("search todos: %w", err)
	}

	todos := make([]*entity.Todo, len(rows))
	for i, row := range rows {
		todos[i] = mapSqlcTodoToEntity(row)
	}
	return todos, int(total), nil
}

func mapSqlcTodoToEntity(row sqlc.Todo) *entity.Todo {
	return &entity.Todo{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: row.DeletedAt,
		},
		Title:       row.Title,
		Description: row.Description,
		Completed:   row.Completed,
	}
}
```

**Note on `SearchTodos` column naming:** Sqlc may use `Column1`, `Limit`, `Offset` as parameter field names for the `$1`, `$2`, `$3` placeholders when they appear in complex expressions. After generating, check the actual generated param struct in `internal/todo/infrastructure/persistence/sqlc/queries.sql.go` and adjust the Search call if the field names differ (e.g., `Column1` might be `Search` or `Pattern`).

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: Clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/todo/infrastructure/persistence
git commit -m "feat(todo): migrate todo repository from database/sql to sqlc"
```

---

### Task 3: Migrate Authentication Domain

**Files:**
- Modify: `sqlc.yaml` — auth block already exists from Task 1
- Modify: `internal/authentication/infrastructure/persistence/queries/queries.sql` — replace placeholder with real queries
- Modify: `internal/authentication/infrastructure/persistence/user_repository.go` — refactor to sqlc
- Modify: `internal/authentication/infrastructure/persistence/refresh_token_repository.go` — refactor to sqlc

- [ ] **Step 1: Write real queries to `internal/authentication/infrastructure/persistence/queries/queries.sql`**

```sql
-- name: CreateUser :exec
INSERT INTO users (id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: GetUserByID :one
SELECT id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at, deleted_at
FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at, deleted_at
FROM users WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByVerifyToken :one
SELECT id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at, deleted_at
FROM users WHERE email_verify_token = $1 AND deleted_at IS NULL;

-- name: GetUserByResetToken :one
SELECT id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at, deleted_at
FROM users WHERE password_reset_token = $1 AND deleted_at IS NULL;

-- name: UpdateUser :exec
UPDATE users SET email = $2, password = $3, name = $4, is_active = $5, updated_at = $6, failed_login_attempts = $7, locked_until = $8, email_verified = $9, email_verify_token = $10, email_verify_expires = $11, password_reset_token = $12, password_reset_expires = $13 WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetRefreshTokenByToken :one
SELECT id, user_id, token, expires_at, revoked_at, created_at, updated_at, deleted_at
FROM refresh_tokens WHERE token = $1 AND deleted_at IS NULL;

-- name: GetRefreshTokensByUserID :many
SELECT id, user_id, token, expires_at, revoked_at, created_at, updated_at, deleted_at
FROM refresh_tokens WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = NOW(), updated_at = NOW() WHERE token = $1 AND deleted_at IS NULL AND revoked_at IS NULL;

-- name: RevokeAllRefreshTokensByUserID :exec
UPDATE refresh_tokens SET revoked_at = NOW(), updated_at = NOW() WHERE user_id = $1 AND deleted_at IS NULL AND revoked_at IS NULL;

-- name: DeleteExpiredRefreshTokens :exec
UPDATE refresh_tokens SET deleted_at = NOW() WHERE expires_at < NOW() AND deleted_at IS NULL;
```

- [ ] **Step 2: Run `make sqlc` to regenerate**

```bash
make sqlc
```

Expected: Clean exit. Auth sqlc package now has user + refresh_token query methods.

- [ ] **Step 3: Rewrite `internal/authentication/infrastructure/persistence/user_repository.go`**

Read the generated sqlc types first to confirm parameter struct names.

```go
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/infrastructure/persistence/sqlc"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	q := sqlc.New(r.db)
	err := q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:                  user.ID,
		Email:               user.Email,
		Password:            user.Password,
		Name:                user.Name,
		IsActive:            user.IsActive,
		FailedLoginAttempts: int32(user.FailedLoginAttempts),
		LockedUntil:         user.LockedUntil,
		EmailVerified:       user.EmailVerified,
		EmailVerifyToken:    user.EmailVerifyToken,
		EmailVerifyExpires:  user.EmailVerifyExpires,
		PasswordResetToken:  user.PasswordResetToken,
		PasswordResetExpires: user.PasswordResetExpires,
		CreatedAt:           user.CreatedAt,
		UpdatedAt:           user.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	q := sqlc.New(r.db)
	row, err := q.GetUserByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return mapSqlcUserToEntity(row), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	q := sqlc.New(r.db)
	row, err := q.GetUserByEmail(ctx, email)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return mapSqlcUserToEntity(row), nil
}

func (r *userRepository) GetByVerifyToken(ctx context.Context, token string) (*entity.User, error) {
	q := sqlc.New(r.db)
	row, err := q.GetUserByVerifyToken(ctx, token)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by verify token: %w", err)
	}
	return mapSqlcUserToEntity(row), nil
}

func (r *userRepository) GetByResetToken(ctx context.Context, token string) (*entity.User, error) {
	q := sqlc.New(r.db)
	row, err := q.GetUserByResetToken(ctx, token)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by reset token: %w", err)
	}
	return mapSqlcUserToEntity(row), nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	q := sqlc.New(r.db)
	result, err := q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:                  user.ID,
		Email:               user.Email,
		Password:            user.Password,
		Name:                user.Name,
		IsActive:            user.IsActive,
		UpdatedAt:           user.UpdatedAt,
		FailedLoginAttempts: int32(user.FailedLoginAttempts),
		LockedUntil:         user.LockedUntil,
		EmailVerified:       user.EmailVerified,
		EmailVerifyToken:    user.EmailVerifyToken,
		EmailVerifyExpires:  user.EmailVerifyExpires,
		PasswordResetToken:  user.PasswordResetToken,
		PasswordResetExpires: user.PasswordResetExpires,
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func mapSqlcUserToEntity(row sqlc.User) *entity.User {
	return &entity.User{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: row.DeletedAt,
		},
		Email:                row.Email,
		Password:             row.Password,
		Name:                 row.Name,
		IsActive:             row.IsActive,
		FailedLoginAttempts:  int(row.FailedLoginAttempts),
		LockedUntil:          row.LockedUntil,
		EmailVerified:        row.EmailVerified,
		EmailVerifyToken:     row.EmailVerifyToken,
		EmailVerifyExpires:   row.EmailVerifyExpires,
		PasswordResetToken:   row.PasswordResetToken,
		PasswordResetExpires: row.PasswordResetExpires,
	}
}
```

- [ ] **Step 4: Rewrite `internal/authentication/infrastructure/persistence/refresh_token_repository.go`**

```go
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/infrastructure/persistence/sqlc"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type refreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) repository.RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	q := sqlc.New(r.db)
	err := q.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		ID:        token.ID,
		UserID:    token.UserID,
		Token:     token.Token,
		ExpiresAt: token.ExpiresAt,
		CreatedAt: token.CreatedAt,
		UpdatedAt: token.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	q := sqlc.New(r.db)
	row, err := q.GetRefreshTokenByToken(ctx, token)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("refresh token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return mapSqlcRefreshTokenToEntity(row), nil
}

func (r *refreshTokenRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.RefreshToken, error) {
	q := sqlc.New(r.db)
	rows, err := q.GetRefreshTokensByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get refresh tokens: %w", err)
	}
	tokens := make([]*entity.RefreshToken, len(rows))
	for i, row := range rows {
		tokens[i] = mapSqlcRefreshTokenToEntity(row)
	}
	return tokens, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, token string) error {
	q := sqlc.New(r.db)
	err := q.RevokeRefreshToken(ctx, token)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	q := sqlc.New(r.db)
	err := q.RevokeAllRefreshTokensByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("revoke all refresh tokens: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) DeleteExpired(ctx context.Context) error {
	q := sqlc.New(r.db)
	err := q.DeleteExpiredRefreshTokens(ctx)
	if err != nil {
		return fmt.Errorf("delete expired tokens: %w", err)
	}
	return nil
}

func mapSqlcRefreshTokenToEntity(row sqlc.RefreshToken) *entity.RefreshToken {
	return &entity.RefreshToken{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: row.DeletedAt,
		},
		UserID:    row.UserID,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		RevokedAt: row.RevokedAt,
	}
}
```

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: Clean build.

- [ ] **Step 6: Commit**

```bash
git add sqlc.yaml internal/authentication/infrastructure/persistence
git commit -m "feat(authentication): migrate user and refresh_token repositories from database/sql to sqlc"
```

---

### Task 4: Migrate Authorization Domain (4 repos)

**Files:**
- Modify: `internal/authorization/infrastructure/persistence/queries/queries.sql` — replace placeholder with real queries
- Modify: `internal/authorization/infrastructure/persistence/role_repository.go` — refactor to sqlc
- Modify: `internal/authorization/infrastructure/persistence/permission_repository.go` — refactor to sqlc
- Modify: `internal/authorization/infrastructure/persistence/role_permission_repository.go` — refactor to sqlc
- Modify: `internal/authorization/infrastructure/persistence/user_role_repository.go` — refactor to sqlc

- [ ] **Step 1: Write real queries to `internal/authorization/infrastructure/persistence/queries/queries.sql`**

```sql
-- name: CreateRole :exec
INSERT INTO roles (id, name, description, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetRoleByID :one
SELECT id, name, description, created_at, updated_at, deleted_at
FROM roles WHERE id = $1 AND deleted_at IS NULL;

-- name: GetRoleByName :one
SELECT id, name, description, created_at, updated_at, deleted_at
FROM roles WHERE name = $1 AND deleted_at IS NULL;

-- name: ListRoles :many
SELECT id, name, description, created_at, updated_at, deleted_at
FROM roles WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountRoles :one
SELECT COUNT(*) FROM roles WHERE deleted_at IS NULL;

-- name: UpdateRole :exec
UPDATE roles SET name = $2, description = $3, updated_at = $4 WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteRole :exec
UPDATE roles SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreatePermission :exec
INSERT INTO permissions (id, name, description, resource, action, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetPermissionByID :one
SELECT id, name, description, resource, action, created_at, updated_at, deleted_at
FROM permissions WHERE id = $1 AND deleted_at IS NULL;

-- name: GetPermissionByName :one
SELECT id, name, description, resource, action, created_at, updated_at, deleted_at
FROM permissions WHERE name = $1 AND deleted_at IS NULL;

-- name: ListPermissions :many
SELECT id, name, description, resource, action, created_at, updated_at, deleted_at
FROM permissions WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountPermissions :one
SELECT COUNT(*) FROM permissions WHERE deleted_at IS NULL;

-- name: UpdatePermission :exec
UPDATE permissions SET name = $2, description = $3, resource = $4, action = $5, updated_at = $6 WHERE id = $1 AND deleted_at IS NULL;

-- name: DeletePermission :exec
UPDATE permissions SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL;

-- name: AssignRolePermission :exec
INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT (role_id, permission_id) DO NOTHING;

-- name: RemoveRolePermission :exec
DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2;

-- name: GetRolePermissionsByRoleID :many
SELECT role_id, permission_id FROM role_permissions WHERE role_id = $1;

-- name: GetPermissionsByRoleID :many
SELECT p.id, p.name, p.description, p.resource, p.action, p.created_at, p.updated_at, p.deleted_at
FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = $1 AND p.deleted_at IS NULL
ORDER BY p.created_at DESC;

-- name: AssignUserRole :exec
INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT (user_id, role_id) DO NOTHING;

-- name: RemoveUserRole :exec
DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2;

-- name: GetUserRolesByUserID :many
SELECT user_id, role_id FROM user_roles WHERE user_id = $1;

-- name: GetRolesByUserID :many
SELECT r.id, r.name, r.description, r.created_at, r.updated_at, r.deleted_at
FROM roles r
JOIN user_roles ur ON r.id = ur.role_id
WHERE ur.user_id = $1 AND r.deleted_at IS NULL
ORDER BY r.created_at DESC;
```

- [ ] **Step 2: Run `make sqlc` to regenerate**

```bash
make sqlc
```

Expected: Clean exit. Authorization sqlc package now has role + permission + role_permission + user_role query methods.

- [ ] **Step 3: Rewrite `internal/authorization/infrastructure/persistence/role_repository.go`**

```go
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence/sqlc"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type roleRepository struct {
	db *sql.DB
}

func NewRoleRepository(db *sql.DB) repository.RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Create(ctx context.Context, role *entity.Role) error {
	q := sqlc.New(r.db)
	err := q.CreateRole(ctx, sqlc.CreateRoleParams{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert role: %w", err)
	}
	return nil
}

func (r *roleRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Role, error) {
	q := sqlc.New(r.db)
	row, err := q.GetRoleByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}
	return mapSqlcRoleToEntity(row), nil
}

func (r *roleRepository) GetByName(ctx context.Context, name string) (*entity.Role, error) {
	q := sqlc.New(r.db)
	row, err := q.GetRoleByName(ctx, name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get role by name: %w", err)
	}
	return mapSqlcRoleToEntity(row), nil
}

func (r *roleRepository) GetAll(ctx context.Context, offset, limit int) ([]*entity.Role, int, error) {
	q := sqlc.New(r.db)

	total, err := q.CountRoles(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count roles: %w", err)
	}

	rows, err := q.ListRoles(ctx, sqlc.ListRolesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("query roles: %w", err)
	}

	roles := make([]*entity.Role, len(rows))
	for i, row := range rows {
		roles[i] = mapSqlcRoleToEntity(row)
	}
	return roles, int(total), nil
}

func (r *roleRepository) Update(ctx context.Context, role *entity.Role) error {
	q := sqlc.New(r.db)
	result, err := q.UpdateRole(ctx, sqlc.UpdateRoleParams{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		UpdatedAt:   role.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("role not found")
	}
	return nil
}

func (r *roleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := sqlc.New(r.db)
	result, err := q.DeleteRole(ctx, id)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("role not found")
	}
	return nil
}

func mapSqlcRoleToEntity(row sqlc.Role) *entity.Role {
	return &entity.Role{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: row.DeletedAt,
		},
		Name:        row.Name,
		Description: row.Description,
	}
}
```

- [ ] **Step 4: Rewrite `internal/authorization/infrastructure/persistence/permission_repository.go`**

```go
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence/sqlc"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type permissionRepository struct {
	db *sql.DB
}

func NewPermissionRepository(db *sql.DB) repository.PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) Create(ctx context.Context, perm *entity.Permission) error {
	q := sqlc.New(r.db)
	err := q.CreatePermission(ctx, sqlc.CreatePermissionParams{
		ID:          perm.ID,
		Name:        perm.Name,
		Description: perm.Description,
		Resource:    perm.Resource,
		Action:      perm.Action,
		CreatedAt:   perm.CreatedAt,
		UpdatedAt:   perm.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert permission: %w", err)
	}
	return nil
}

func (r *permissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error) {
	q := sqlc.New(r.db)
	row, err := q.GetPermissionByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("permission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get permission: %w", err)
	}
	return mapSqlcPermissionToEntity(row), nil
}

func (r *permissionRepository) GetByName(ctx context.Context, name string) (*entity.Permission, error) {
	q := sqlc.New(r.db)
	row, err := q.GetPermissionByName(ctx, name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("permission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get permission by name: %w", err)
	}
	return mapSqlcPermissionToEntity(row), nil
}

func (r *permissionRepository) GetAll(ctx context.Context, offset, limit int) ([]*entity.Permission, int, error) {
	q := sqlc.New(r.db)

	total, err := q.CountPermissions(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count permissions: %w", err)
	}

	rows, err := q.ListPermissions(ctx, sqlc.ListPermissionsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("query permissions: %w", err)
	}

	perms := make([]*entity.Permission, len(rows))
	for i, row := range rows {
		perms[i] = mapSqlcPermissionToEntity(row)
	}
	return perms, int(total), nil
}

func (r *permissionRepository) Update(ctx context.Context, perm *entity.Permission) error {
	q := sqlc.New(r.db)
	result, err := q.UpdatePermission(ctx, sqlc.UpdatePermissionParams{
		ID:          perm.ID,
		Name:        perm.Name,
		Description: perm.Description,
		Resource:    perm.Resource,
		Action:      perm.Action,
		UpdatedAt:   perm.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("update permission: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("permission not found")
	}
	return nil
}

func (r *permissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := sqlc.New(r.db)
	result, err := q.DeletePermission(ctx, id)
	if err != nil {
		return fmt.Errorf("delete permission: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("permission not found")
	}
	return nil
}

func mapSqlcPermissionToEntity(row sqlc.Permission) *entity.Permission {
	return &entity.Permission{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: row.DeletedAt,
		},
		Name:        row.Name,
		Description: row.Description,
		Resource:    row.Resource,
		Action:      row.Action,
	}
}
```

- [ ] **Step 5: Rewrite `internal/authorization/infrastructure/persistence/role_permission_repository.go`**

```go
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence/sqlc"
	"github.com/google/uuid"
)

type rolePermissionRepository struct {
	db *sql.DB
}

func NewRolePermissionRepository(db *sql.DB) repository.RolePermissionRepository {
	return &rolePermissionRepository{db: db}
}

func (r *rolePermissionRepository) Assign(ctx context.Context, rp entity.RolePermission) error {
	q := sqlc.New(r.db)
	err := q.AssignRolePermission(ctx, sqlc.AssignRolePermissionParams{
		RoleID:       rp.RoleID,
		PermissionID: rp.PermissionID,
	})
	if err != nil {
		return fmt.Errorf("assign permission: %w", err)
	}
	return nil
}

func (r *rolePermissionRepository) Remove(ctx context.Context, roleID, permissionID uuid.UUID) error {
	q := sqlc.New(r.db)
	err := q.RemoveRolePermission(ctx, sqlc.RemoveRolePermissionParams{
		RoleID:       roleID,
		PermissionID: permissionID,
	})
	if err != nil {
		return fmt.Errorf("remove permission: %w", err)
	}
	return nil
}

func (r *rolePermissionRepository) GetByRoleID(ctx context.Context, roleID uuid.UUID) ([]entity.RolePermission, error) {
	q := sqlc.New(r.db)
	rows, err := q.GetRolePermissionsByRoleID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("get role permissions: %w", err)
	}
	rps := make([]entity.RolePermission, len(rows))
	for i, row := range rows {
		rps[i] = mapSqlcRolePermissionToEntity(row)
	}
	return rps, nil
}

func (r *rolePermissionRepository) GetPermissionsByRoleID(ctx context.Context, roleID uuid.UUID) ([]*entity.Permission, error) {
	q := sqlc.New(r.db)
	rows, err := q.GetPermissionsByRoleID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("get permissions by role: %w", err)
	}
	perms := make([]*entity.Permission, len(rows))
	for i, row := range rows {
		perms[i] = mapSqlcPermissionToEntity(row)
	}
	return perms, nil
}

func mapSqlcRolePermissionToEntity(row sqlc.RolePermission) entity.RolePermission {
	return entity.RolePermission{
		RoleID:       row.RoleID,
		PermissionID: row.PermissionID,
	}
}
```

- [ ] **Step 6: Rewrite `internal/authorization/infrastructure/persistence/user_role_repository.go`**

```go
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence/sqlc"
	"github.com/google/uuid"
)

type userRoleRepository struct {
	db *sql.DB
}

func NewUserRoleRepository(db *sql.DB) repository.UserRoleRepository {
	return &userRoleRepository{db: db}
}

func (r *userRoleRepository) Assign(ctx context.Context, ur entity.UserRole) error {
	q := sqlc.New(r.db)
	err := q.AssignUserRole(ctx, sqlc.AssignUserRoleParams{
		UserID: ur.UserID,
		RoleID: ur.RoleID,
	})
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	return nil
}

func (r *userRoleRepository) Remove(ctx context.Context, userID, roleID uuid.UUID) error {
	q := sqlc.New(r.db)
	err := q.RemoveUserRole(ctx, sqlc.RemoveUserRoleParams{
		UserID: userID,
		RoleID: roleID,
	})
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}
	return nil
}

func (r *userRoleRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]entity.UserRole, error) {
	q := sqlc.New(r.db)
	rows, err := q.GetUserRolesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	urs := make([]entity.UserRole, len(rows))
	for i, row := range rows {
		urs[i] = mapSqlcUserRoleToEntity(row)
	}
	return urs, nil
}

func (r *userRoleRepository) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error) {
	q := sqlc.New(r.db)
	rows, err := q.GetRolesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get roles by user: %w", err)
	}
	roles := make([]*entity.Role, len(rows))
	for i, row := range rows {
		roles[i] = mapSqlcRoleToEntity(row)
	}
	return roles, nil
}

func mapSqlcUserRoleToEntity(row sqlc.UserRole) entity.UserRole {
	return entity.UserRole{
		UserID: row.UserID,
		RoleID: row.RoleID,
	}
}
```

- [ ] **Step 7: Verify build**

```bash
go build ./...
```

Expected: Clean build.

- [ ] **Step 8: Commit**

```bash
git add sqlc.yaml internal/authorization/infrastructure/persistence
git commit -m "feat(authorization): migrate role, permission, role_permission, user_role repositories from database/sql to sqlc"
```

---

### Task 5: Migrate User Domain

**Files:**
- Modify: `internal/user/infrastructure/persistence/queries/queries.sql` — replace placeholder with real queries
- Modify: `internal/user/infrastructure/persistence/user_repository.go` — refactor to sqlc

- [ ] **Step 1: Write real queries to `internal/user/infrastructure/persistence/queries/queries.sql`**

```sql
-- name: CountUsers :one
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;

-- name: ListUsers :many
SELECT id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at, deleted_at
FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: GetUserByID :one
SELECT id, email, password, name, is_active, failed_login_attempts, locked_until, email_verified, email_verify_token, email_verify_expires, password_reset_token, password_reset_expires, created_at, updated_at, deleted_at
FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUser :exec
UPDATE users SET email = $2, name = $3, is_active = $4, updated_at = $5, deleted_at = $6 WHERE id = $1;

-- name: DeleteUser :exec
UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL;
```

**Note on user domain queries:** We select all columns from `users` (same as authentication domain) for schema consistency — sqlc generates a single `User` struct per `sqlc/` package. The mapping function only populates fields used by the user domain service.

- [ ] **Step 2: Run `make sqlc` to regenerate**

```bash
make sqlc
```

Expected: Clean exit. User sqlc package now has user list/get/update/delete query methods.

- [ ] **Step 3: Rewrite `internal/user/infrastructure/persistence/user_repository.go`**

```go
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/user/infrastructure/persistence/sqlc"
	"github.com/google/uuid"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) List(ctx context.Context, offset, limit int) ([]*entity.User, int, error) {
	q := sqlc.New(r.db)

	total, err := q.CountUsers(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	rows, err := q.ListUsers(ctx, sqlc.ListUsersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	users := make([]*entity.User, len(rows))
	for i, row := range rows {
		users[i] = mapSqlcUserToEntityForAdmin(row)
	}
	return users, int(total), nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	q := sqlc.New(r.db)
	row, err := q.GetUserByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return mapSqlcUserToEntityForAdmin(row), nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	q := sqlc.New(r.db)
	result, err := q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		IsActive:  user.IsActive,
		UpdatedAt: user.UpdatedAt,
		DeletedAt: user.DeletedAt,
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := sqlc.New(r.db)
	result, err := q.DeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func mapSqlcUserToEntityForAdmin(row sqlc.User) *entity.User {
	return &entity.User{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: row.DeletedAt,
		},
		Email:    row.Email,
		Name:     row.Name,
		IsActive: row.IsActive,
	}
}
```

**Note on `UpdateUser` field mapping:** The sqlc-generated `UpdateUserParams` struct for the user domain's `UpdateUser` query only contains the fields used in the SET clause (`email`, `name`, `is_active`, `updated_at`, `deleted_at`) and WHERE clause (`id`). Check the exact param struct in `internal/user/infrastructure/persistence/sqlc/queries.sql.go` after generation and adjust the field names if needed.

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add sqlc.yaml internal/user/infrastructure/persistence
git commit -m "feat(user): migrate user repository from database/sql to sqlc"
```

---

### Task 6: Migrate AuditLog Domain

**Files:**
- Modify: `internal/shared/auditlog/queries/queries.sql` — replace placeholder with real queries (JSONB handling)
- Modify: `internal/shared/auditlog/repository.go` — refactor to sqlc

- [ ] **Step 1: Write real queries to `internal/shared/auditlog/queries/queries.sql`**

Use `::jsonb` cast on the metadata parameter to avoid importing `github.com/sqlc-dev/pqtype`. Sqlc will generate a `[]byte` field which is compatible with `json.RawMessage`.

```sql
-- name: InsertAuditLog :exec
INSERT INTO audit_logs (id, request_id, user_id, user_email, method, path, status_code, duration_ms, ip, user_agent, request_body, response_size, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: InsertErrorLog :exec
INSERT INTO error_logs (id, request_id, user_id, user_email, level, message, error, stack_trace, method, path, status_code, ip, user_agent, request_body, metadata, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, $16);
```

- [ ] **Step 2: Run `make sqlc` to regenerate**

```bash
make sqlc
```

Expected: Clean exit. Auditlog sqlc package now has insert queries.

- [ ] **Step 3: Rewrite `internal/shared/auditlog/repository.go`**

```go
package auditlog

import (
	"context"
	"database/sql"

	"github.com/IDTS-LAB/go-codebase/internal/shared/auditlog/sqlc"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) InsertAuditLog(ctx context.Context, log *AuditLog) error {
	q := sqlc.New(r.db)
	return q.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ID:           log.ID,
		RequestID:    log.RequestID,
		UserID:       log.UserID,
		UserEmail:    log.UserEmail,
		Method:       log.Method,
		Path:         log.Path,
		StatusCode:   int32(log.StatusCode),
		DurationMs:   log.DurationMs,
		Ip:           log.IP,
		UserAgent:    log.UserAgent,
		RequestBody:  log.RequestBody,
		ResponseSize: int32(log.ResponseSize),
		CreatedAt:    log.CreatedAt,
	})
}

func (r *Repository) InsertErrorLog(ctx context.Context, log *ErrorLog) error {
	q := sqlc.New(r.db)
	return q.InsertErrorLog(ctx, sqlc.InsertErrorLogParams{
		ID:          log.ID,
		RequestID:   log.RequestID,
		UserID:      log.UserID,
		UserEmail:   log.UserEmail,
		Level:       log.Level,
		Message:     log.Message,
		Error:       log.Error,
		StackTrace:  log.StackTrace,
		Method:      log.Method,
		Path:        log.Path,
		StatusCode:  int32(log.StatusCode),
		Ip:          log.IP,
		UserAgent:   log.UserAgent,
		RequestBody: log.RequestBody,
		Metadata:    []byte(log.Metadata),
		CreatedAt:   log.CreatedAt,
	})
}
```

**Note on JSONB handling:** The `::jsonb` cast on `$15` makes sqlc generate the `Metadata` field as `[]byte`. `json.RawMessage` has underlying type `[]byte`, so `[]byte(log.Metadata)` works as a type conversion. The `StatusCode` field needs `int32()` cast from the entity's `int`. Check the generated param struct in `internal/shared/auditlog/sqlc/queries.sql.go` after generation and adjust field names if needed (e.g., `Ip` vs `IP`, `StatusCode` type).

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

Expected: Clean build.

- [ ] **Step 5: Remove placeholder query files (optional — harmless to keep)**

```bash
# No need to remove; the generated Ping method just sits unused but causes no harm.
```

- [ ] **Step 6: Commit**

```bash
git add sqlc.yaml internal/shared/auditlog
git commit -m "feat(auditlog): migrate auditlog repository from database/sql to sqlc with JSONB handling"
```

---

### Task 7: Final Verification

**Files:** None

- [ ] **Step 1: Run `make sqlc` cleanly**

```bash
make sqlc
```

Expected: All 5 sqlc packages regenerate without errors.

- [ ] **Step 2: Full build**

```bash
go build ./...
```

Expected: Clean build across all packages.

- [ ] **Step 3: Run tests**

```bash
go test ./... 2>&1
```

Expected: All existing tests pass. No new tests added; the migration changes only the internal implementation, keeping the same interface contracts and behavior.

- [ ] **Step 4: Run vet**

```bash
go vet ./...
```

Expected: Clean.

- [ ] **Step 5: Run lint**

```bash
make lint
```

Expected: Clean or only pre-existing lint issues.

- [ ] **Step 6: Final verification commit**

```bash
git add -A
git commit -m "chore: final verification — sqlc generation, build, test, vet, lint all clean"
```

---

## Verification Summary

| Check | Expected |
|-------|----------|
| `make sqlc` | 5 packages generated, no errors |
| `go build ./...` | No compilation errors |
| `go test ./...` | All tests pass (no behavioral changes) |
| `go vet ./...` | No vet issues |
| `make lint` | No new lint issues |
| `go.mod` | No new dependencies (pqtype avoided) |
