# Casbin, Cursor Pagination, sqlc Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate Casbin to standard `casbin_rule` table with custom adapter, replace offset/limit pagination with bidirectional cursor pagination across 5 domains, and migrate simple CRUD in `user` and `todo` repositories to sqlc.

**Architecture:** Three independent subsystems: (1) sqlc migration + pagination cleanup, (2) cursor pagination shared utility + repository/query/handler changes, (3) Casbin standard adapter + sync flow. Implemented in that order to minimize merge conflicts.

**Tech Stack:** Go 1.x, `database/sql`, `casbin/v2`, `sqlc`, `chi/v5`, `go.uber.org/fx`

## Global Constraints

- All new signatures must be backward-aware: old callers of `GetAll(ctx, offset, limit)` become callers of `GetAll(ctx, cursor, limit)`
- The `internal/shared/utils/APIResponse.Meta` field changes to `interface{}` (was `*PaginationMeta`, now either `*PaginationMeta` or `*CursorMeta`)
- All sqlc `.sql` files in `queries/` directories get pagination queries removed, then regenerated
- No new third-party dependencies — Casbin adapter is custom, cursor encoding is `encoding/json` + `base64`

---

### Task 1: Fix sqlc `UpdateUser` query before migration

**Files:**
- Modify: `internal/user/infrastructure/persistence/queries/queries.sql:12-13`
- Verify: `internal/user/infrastructure/persistence/sqlc/queries.sql.go` (auto-generated)

**Issue:** Current `UpdateUser` sets `deleted_at = $6` without `WHERE deleted_at IS NULL`, which could undelete soft-deleted users.

- [ ] **Step 1: Fix the SQL query**

Replace lines 12-13 of `internal/user/infrastructure/persistence/queries/queries.sql`:

```sql
-- name: UpdateUser :execrows
UPDATE users SET email = $2, name = $3, is_active = $4, updated_at = $5 WHERE id = $1 AND deleted_at IS NULL;
```

- [ ] **Step 2: Regenerate sqlc**

```bash
sqlc generate
```

Verify `internal/user/infrastructure/persistence/sqlc/queries.sql.go` now has `UpdateUser` with 5 params and `WHERE deleted_at IS NULL`.

- [ ] **Step 3: Run existing tests to confirm no regression**

```bash
go test ./internal/user/... -v -count=1 2>&1 | tail -20
```

Expected: tests pass (or skip if no tests exist).

- [ ] **Step 4: Commit**

```bash
git add internal/user/infrastructure/persistence/queries/queries.sql internal/user/infrastructure/persistence/sqlc/
git commit -m "fix: UpdateUser sqlc query uses WHERE deleted_at IS NULL and excludes deleted_at SET"
```

---

### Task 2: Create shared cursor package

**Files:**
- Create: `internal/shared/cursor/cursor.go`
- Create: `internal/shared/cursor/cursor_test.go`

**Interfaces:**
- Consumes: `time`, `uuid`
- Produces: `cursor.Encode(t time.Time, id uuid.UUID) string`, `cursor.Decode(s string) (Cursor, error)`, `Cursor{Timestamp time.Time, ID uuid.UUID}`

- [ ] **Step 1: Write the test first**

**File:** `internal/shared/cursor/cursor_test.go`

```go
package cursor

import (
    "testing"
    "time"
    "github.com/google/uuid"
)

func TestEncodeDecode(t *testing.T) {
    now := time.Now().UTC().Truncate(time.Microsecond)
    id := uuid.New()

    token := Encode(now, id)
    if token == "" {
        t.Fatal("expected non-empty token")
    }

    c, err := Decode(token)
    if err != nil {
        t.Fatalf("decode error: %v", err)
    }

    if !c.Timestamp.Equal(now) {
        t.Errorf("timestamp mismatch: got %v, want %v", c.Timestamp, now)
    }
    if c.ID != id {
        t.Errorf("id mismatch: got %v, want %v", c.ID, id)
    }
}

func TestDecodeInvalid(t *testing.T) {
    _, err := Decode("invalid-base64!")
    if err == nil {
        t.Fatal("expected error for invalid token")
    }

    _, err = Decode("")
    if err == nil {
        t.Fatal("expected error for empty token")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/shared/cursor/ -v -count=1
```

Expected: FAIL with "package does not exist" or compile error.

- [ ] **Step 3: Implement cursor package**

**File:** `internal/shared/cursor/cursor.go`

```go
package cursor

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "time"
    "github.com/google/uuid"
)

type Cursor struct {
    Timestamp time.Time `json:"t"`
    ID        uuid.UUID `json:"i"`
}

func Encode(t time.Time, id uuid.UUID) string {
    c := Cursor{Timestamp: t.UTC(), ID: id}
    b, _ := json.Marshal(c)
    return base64.URLEncoding.EncodeToString(b)
}

func Decode(s string) (Cursor, error) {
    if s == "" {
        return Cursor{}, fmt.Errorf("empty cursor")
    }
    b, err := base64.URLEncoding.DecodeString(s)
    if err != nil {
        return Cursor{}, fmt.Errorf("decode cursor: %w", err)
    }
    var c Cursor
    if err := json.Unmarshal(b, &c); err != nil {
        return Cursor{}, fmt.Errorf("unmarshal cursor: %w", err)
    }
    return c, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/shared/cursor/ -v -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shared/cursor/
git commit -m "feat: add shared cursor package for cursor-based pagination"
```

---

### Task 3: Add CursorMeta and RespondCursorPaginated to utils

**Files:**
- Modify: `internal/shared/utils/utils.go`
- Modify: `internal/shared/utils/handler.go`

**Interfaces:**
- Consumes: `cursor.Cursor`
- Produces: `utils.CursorMeta`, `utils.RespondCursorPaginated(w, data, nextCursor, prevCursor, hasNext, hasPrev, limit)`, `utils.HandleCursorPaginated(w, data, nextCursor, prevCursor, hasNext, hasPrev, limit, err)`

- [ ] **Step 1: Update `APIResponse.Meta` to `interface{}`**

Replace line 14 in `internal/shared/utils/utils.go`:

```go
Meta    interface{}     `json:"meta"`
```

- [ ] **Step 2: Add CursorMeta to `internal/shared/utils/utils.go`**

Add after `PaginationMeta` block (after line 23):

```go
type CursorMeta struct {
    NextCursor *string `json:"next_cursor"`
    PrevCursor *string `json:"prev_cursor"`
    HasNext    bool    `json:"has_next"`
    HasPrev    bool    `json:"has_prev"`
    Limit      int     `json:"limit"`
}
```

- [ ] **Step 3: Add `RespondCursorPaginated` to `internal/shared/utils/utils.go`**

Add after `RespondPaginated` (after line 71):

```go
func RespondCursorPaginated(w http.ResponseWriter, data interface{}, nextCursor, prevCursor *string, hasNext, hasPrev bool, limit int) {
    RespondJSON(w, http.StatusOK, APIResponse{
        Success: true,
        Data:    data,
        Meta: CursorMeta{
            NextCursor: nextCursor,
            PrevCursor: prevCursor,
            HasNext:    hasNext,
            HasPrev:    hasPrev,
            Limit:      limit,
        },
    })
}
```

- [ ] **Step 4: Add `HandleCursorPaginated` to `internal/shared/utils/handler.go`**

Add after `HandlePaginated` (after line 43):

```go
func HandleCursorPaginated(w http.ResponseWriter, data interface{}, nextCursor, prevCursor *string, hasNext, hasPrev bool, limit int, err error) {
    if err != nil {
        MapError(w, err)
        return
    }
    RespondCursorPaginated(w, data, nextCursor, prevCursor, hasNext, hasPrev, limit)
}
```

- [ ] **Step 5: Update formatter middleware to handle CursorMeta**

In `internal/shared/middleware/formatter.go`, update the pagination check section (lines 51-65) to also check for `cursor_meta`:

Replace lines 51-65:

```go
var paginated struct {
    Data       interface{} `json:"data"`
    Pagination interface{} `json:"pagination"`
}
if json.Unmarshal(fw.body, &paginated) == nil && paginated.Data != nil && paginated.Pagination != nil {
    var meta utils.PaginationMeta
    metaBytes, _ := json.Marshal(paginated.Pagination)
    json.Unmarshal(metaBytes, &meta)
    json.NewEncoder(w).Encode(utils.APIResponse{
        Success: true,
        Data:    paginated.Data,
        Meta:    &meta,
    })
    return
}

var cursorResp struct {
    Data interface{} `json:"data"`
    Meta interface{} `json:"meta"`
}
if json.Unmarshal(fw.body, &cursorResp) == nil && cursorResp.Data != nil && cursorResp.Meta != nil {
    var meta utils.CursorMeta
    metaBytes, _ := json.Marshal(cursorResp.Meta)
    json.Unmarshal(metaBytes, &meta)
    json.NewEncoder(w).Encode(utils.APIResponse{
        Success: true,
        Data:    cursorResp.Data,
        Meta:    &meta,
    })
    return
}
```

- [ ] **Step 6: Build check**

```bash
go build ./...
```

Expected: no compile errors.

- [ ] **Step 7: Commit**

```bash
git add internal/shared/utils/utils.go internal/shared/utils/handler.go internal/shared/middleware/formatter.go
git commit -m "feat: add CursorMeta, RespondCursorPaginated, update formatter"
```

---

### Task 4: Cleanup pagination queries from sqlc + regenerate

**Files:**
- Modify: `internal/todo/infrastructure/persistence/queries/todo.sql` (remove ListTodos, CountTodos, SearchTodos, CountSearchTodos)
- Modify: `internal/user/infrastructure/persistence/queries/queries.sql` (remove CountUsers, ListUsers)
- Modify: `internal/authorization/infrastructure/persistence/queries/queries.sql` (remove ListRoles, CountRoles, ListPermissions, CountPermissions)
- Modify: `internal/tenant/infrastructure/persistence/queries/queries.sql` (remove ListTenants, CountTenants)
- All `sqlc/` dirs will be regenerated

- [ ] **Step 1: Clean `todo.sql`**

Replace `internal/todo/infrastructure/persistence/queries/todo.sql` with:

```sql
-- name: CreateTodo :exec
INSERT INTO todos (id, title, description, completed, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetTodoByID :one
SELECT id, title, description, completed, created_at, updated_at, deleted_at
FROM todos
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateTodo :execrows
UPDATE todos
SET title = $2, description = $3, completed = $4, updated_at = $5
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteTodo :execrows
UPDATE todos
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
```

- [ ] **Step 2: Clean `user/queries.sql`**

Replace `internal/user/infrastructure/persistence/queries/queries.sql` with:

```sql
-- name: GetUserByID :one
SELECT id, email, name, is_active, created_at, updated_at, deleted_at
FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUser :execrows
UPDATE users SET email = $2, name = $3, is_active = $4, updated_at = $5 WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteUser :execrows
UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL;
```

- [ ] **Step 3: Clean `authorization/queries.sql`**

Remove lines 13-18 (ListRoles, CountRoles) and lines 38-43 (ListPermissions, CountPermissions) from `internal/authorization/infrastructure/persistence/queries/queries.sql`.

The remaining file should keep: CreateRole, GetRoleByID, GetRoleByName, UpdateRole, DeleteRole, CreatePermission, GetPermissionByID, GetPermissionByName, UpdatePermission, DeletePermission, AssignRolePermission, RemoveRolePermission, GetRolePermissionsByRoleID, GetPermissionsByRoleID, AssignUserRole, RemoveUserRole, GetUserRolesByUserID, GetRolesByUserID.

- [ ] **Step 4: Clean `tenant/queries.sql`**

Remove lines 13-18 (ListTenants, CountTenants) from `internal/tenant/infrastructure/persistence/queries/queries.sql`.

- [ ] **Step 5: Regenerate all sqlc**

```bash
sqlc generate
```

Verify no errors. Check that the generated files no longer contain ListTodos, CountTodos, SearchTodos, CountSearchTodos, ListUsers, CountUsers, ListRoles, CountRoles, ListPermissions, CountPermissions, ListTenants, CountTenants.

- [ ] **Step 6: Build check**

```bash
go build ./...
```

Expected: pass (the generated code removed those query functions, but they weren't being called).

- [ ] **Step 7: Commit**

```bash
git add internal/todo/infrastructure/persistence/queries/ internal/user/infrastructure/persistence/queries/ internal/authorization/infrastructure/persistence/queries/ internal/tenant/infrastructure/persistence/queries/ internal/todo/infrastructure/persistence/sqlc/ internal/user/infrastructure/persistence/sqlc/ internal/authorization/infrastructure/persistence/sqlc/ internal/tenant/infrastructure/persistence/sqlc/
git commit -m "refactor: remove pagination queries from sqlc, keep only CRUD"
```

---

### Task 5: Migrate user repository to sqlc (simple CRUD)

**Files:**
- Modify: `internal/user/infrastructure/persistence/user_repository.go`

**Interfaces:**
- Consumes: `sqlc.New(r.db)` from existing generated code
- Produces: `GetByID`, `Update`, `Delete` use sqlc; `List` stays raw SQL

- [ ] **Step 1: Update `GetByID` to use sqlc**

Replace lines 82-99 in `internal/user/infrastructure/persistence/user_repository.go`:

```go
func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
    q := sqlc.New(r.db)
    row, err := q.GetUserByID(ctx, id)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("user not found")
    }
    if err != nil {
        return nil, fmt.Errorf("get user: %w", err)
    }
    var u entity.User
    u.ID = row.ID
    u.Email = row.Email
    u.Name = row.Name
    u.IsActive = row.IsActive
    u.CreatedAt = row.CreatedAt
    u.UpdatedAt = row.UpdatedAt
    if row.DeletedAt.Valid {
        u.DeletedAt = &row.DeletedAt.Time
    }
    return &u, nil
}
```

- [ ] **Step 2: Update `Update` to use sqlc**

Replace lines 101-116:

```go
func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
    q := sqlc.New(r.db)
    rows, err := q.UpdateUser(ctx, sqlc.UpdateUserParams{
        ID:       user.ID,
        Email:    user.Email,
        Name:     user.Name,
        IsActive: user.IsActive,
        UpdatedAt: time.Now().UTC(),
    })
    if err != nil {
        return fmt.Errorf("update user: %w", err)
    }
    if rows == 0 {
        return fmt.Errorf("user not found")
    }
    return nil
}
```

Add `"time"` to the imports in the file.

- [ ] **Step 3: Update `Delete` to use sqlc**

Replace lines 118-133:

```go
func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
    q := sqlc.New(r.db)
    rows, err := q.DeleteUser(ctx, id)
    if err != nil {
        return fmt.Errorf("delete user: %w", err)
    }
    if rows == 0 {
        return fmt.Errorf("user not found")
    }
    return nil
}
```

- [ ] **Step 4: Build check**

```bash
go build ./...
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/user/infrastructure/persistence/user_repository.go
git commit -m "feat: migrate user repository CRUD (GetByID, Update, Delete) to sqlc"
```

---

### Task 6: Migrate todo repository to sqlc (simple CRUD)

**Files:**
- Modify: `internal/todo/infrastructure/persistence/todo_repository.go`

- [ ] **Step 1: Add sqlc import to todo_repository.go**

Add to the imports:

```go
"github.com/IDTS-LAB/go-codebase/internal/todo/infrastructure/persistence/sqlc"
```

- [ ] **Step 2: Update `Create` to use sqlc**

Replace lines 24-33:

```go
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
```

- [ ] **Step 3: Update `GetByID` to use sqlc**

Replace lines 35-52:

```go
func (r *todoRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Todo, error) {
    q := sqlc.New(r.db)
    row, err := q.GetTodoByID(ctx, id)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("todo not found")
    }
    if err != nil {
        return nil, fmt.Errorf("get todo: %w", err)
    }
    todo := &entity.Todo{
        ID:          row.ID,
        Title:       row.Title,
        Description: row.Description,
        Completed:   row.Completed,
        CreatedAt:   row.CreatedAt,
        UpdatedAt:   row.UpdatedAt,
    }
    if row.DeletedAt.Valid {
        todo.DeletedAt = &row.DeletedAt.Time
    }
    return todo, nil
}
```

- [ ] **Step 4: Update `Update` to use sqlc**

Replace lines 112-127:

```go
func (r *todoRepository) Update(ctx context.Context, todo *entity.Todo) error {
    q := sqlc.New(r.db)
    rows, err := q.UpdateTodo(ctx, sqlc.UpdateTodoParams{
        ID:          todo.ID,
        Title:       todo.Title,
        Description: todo.Description,
        Completed:   todo.Completed,
        UpdatedAt:   todo.UpdatedAt,
    })
    if err != nil {
        return fmt.Errorf("update todo: %w", err)
    }
    if rows == 0 {
        return fmt.Errorf("todo not found")
    }
    return nil
}
```

- [ ] **Step 5: Update `Delete` to use sqlc**

Replace lines 129-144:

```go
func (r *todoRepository) Delete(ctx context.Context, id uuid.UUID) error {
    q := sqlc.New(r.db)
    rows, err := q.SoftDeleteTodo(ctx, id)
    if err != nil {
        return fmt.Errorf("delete todo: %w", err)
    }
    if rows == 0 {
        return fmt.Errorf("todo not found")
    }
    return nil
}
```

- [ ] **Step 6: Build check**

```bash
go build ./...
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add internal/todo/infrastructure/persistence/todo_repository.go
git commit -m "feat: migrate todo repository CRUD (Create, GetByID, Update, Delete) to sqlc"
```

---

### Task 7: Update repository interfaces for cursor pagination

**Files:**
- Modify: `internal/todo/domain/repository/todo_repository.go`
- Modify: `internal/user/domain/repository/user_repository.go`
- Modify: `internal/tenant/domain/repository/tenant.go`
- Modify: `internal/authorization/domain/repository/authorization_repository.go`
- Modify: `internal/core/domain/repository.go` (optional, generic interface)

- [ ] **Step 1: Update `TodoRepository`**

Replace `internal/todo/domain/repository/todo_repository.go`:

```go
package repository

import (
    "context"
    "github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
    "github.com/google/uuid"
)

type TodoRepository interface {
    Create(ctx context.Context, todo *entity.Todo) error
    GetByID(ctx context.Context, id uuid.UUID) (*entity.Todo, error)
    GetAll(ctx context.Context, cursor *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error)
    Update(ctx context.Context, todo *entity.Todo) error
    Delete(ctx context.Context, id uuid.UUID) error
    Search(ctx context.Context, query string, cursor *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error)
}
```

- [ ] **Step 2: Update `UserRepository`**

Replace `internal/user/domain/repository/user_repository.go`:

```go
package repository

import (
    "context"
    "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
    "github.com/google/uuid"
)

type UserRepository interface {
    List(ctx context.Context, cursor *string, limit int) ([]*entity.User, *string, *string, bool, bool, error)
    GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
    Update(ctx context.Context, user *entity.User) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

- [ ] **Step 3: Update `TenantRepository`**

Replace `internal/tenant/domain/repository/tenant.go`:

```go
package repository

import (
    "context"
    "github.com/IDTS-LAB/go-codebase/internal/tenant/domain/entity"
    "github.com/google/uuid"
)

type TenantRepository interface {
    Create(ctx context.Context, t *entity.Tenant) error
    GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error)
    GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error)
    List(ctx context.Context, cursor *string, limit int) ([]entity.Tenant, *string, *string, bool, bool, error)
    Update(ctx context.Context, t *entity.Tenant) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

- [ ] **Step 4: Update `RoleRepository` and `PermissionRepository`**

Replace `internal/authorization/domain/repository/authorization_repository.go`:

```go
package repository

import (
    "context"
    "github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
    "github.com/google/uuid"
)

type RoleRepository interface {
    Create(ctx context.Context, role *entity.Role) error
    GetByID(ctx context.Context, id uuid.UUID) (*entity.Role, error)
    GetByName(ctx context.Context, name string) (*entity.Role, error)
    GetAll(ctx context.Context, cursor *string, limit int) ([]*entity.Role, *string, *string, bool, bool, error)
    Update(ctx context.Context, role *entity.Role) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type PermissionRepository interface {
    Create(ctx context.Context, perm *entity.Permission) error
    GetByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error)
    GetByName(ctx context.Context, name string) (*entity.Permission, error)
    GetAll(ctx context.Context, cursor *string, limit int) ([]*entity.Permission, *string, *string, bool, bool, error)
    Update(ctx context.Context, perm *entity.Permission) error
    Delete(ctx context.Context, id uuid.UUID) error
}

// UserRoleRepository and RolePermissionRepository remain unchanged
```

- [ ] **Step 5: Build check**

```bash
go build ./...
```

Expected: compile errors in repository implementations (they haven't been updated yet). That's expected.

- [ ] **Step 6: Commit**

```bash
git add internal/todo/domain/repository/todo_repository.go internal/user/domain/repository/user_repository.go internal/tenant/domain/repository/tenant.go internal/authorization/domain/repository/authorization_repository.go
git commit -m "feat: update repository interfaces for cursor pagination"
```

---

### Task 8: Implement cursor pagination in todo repository (GetAll, Search)

**Files:**
- Modify: `internal/todo/infrastructure/persistence/todo_repository.go`

- [ ] **Step 1: Rewrite `GetAll` with cursor pagination**

Replace lines 54-110 (the current `GetAll` method):

```go
func (r *todoRepository) GetAll(ctx context.Context, cursor *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error) {
    args := []interface{}{}
    whereClause := "WHERE deleted_at IS NULL"

    if r.tenantConfig != nil && r.tenantConfig.Enabled {
        tenantID := middleware.GetTenantID(ctx)
        if tenantID != "" {
            whereClause += fmt.Sprintf(" AND tenant_id = $%d", len(args)+1)
            args = append(args, tenantID)
        }
    }

    nextPos := len(args) + 1
    if cursor != nil {
        c, err := cursor.Decode(*cursor)
        if err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("invalid cursor: %w", err)
        }
        whereClause += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", nextPos, nextPos+1)
        args = append(args, c.Timestamp, c.ID)
        nextPos += 2
    }

    dataQuery := fmt.Sprintf("SELECT id, title, description, completed, created_at, updated_at, deleted_at FROM todos %s ORDER BY created_at DESC, id DESC LIMIT $%d", whereClause, nextPos)
    countArgs := make([]interface{}, len(args))
    copy(countArgs, args)
    queryArgs := append(args, limit+1)

    rows, err := r.db.QueryContext(ctx, dataQuery, queryArgs...)
    if err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("query todos: %w", err)
    }
    defer rows.Close()

    var todos []*entity.Todo
    for rows.Next() {
        var t entity.Todo
        var deletedAt sql.NullTime
        if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt, &deletedAt); err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("scan todo: %w", err)
        }
        if deletedAt.Valid {
            t.DeletedAt = &deletedAt.Time
        }
        todos = append(todos, &t)
    }
    if err := rows.Err(); err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("rows iteration: %w", err)
    }

    hasNext := len(todos) > limit
    if hasNext {
        todos = todos[:limit]
    }

    var nextCursor *string
    var prevCursor *string
    if len(todos) > 0 {
        last := todos[len(todos)-1]
        nc := cursor.Encode(last.CreatedAt, last.ID)
        nextCursor = &nc

        first := todos[0]
        pc := cursor.Encode(first.CreatedAt, first.ID)
        prevCursor = &pc
    }

    hasPrev := cursor != nil
    if hasPrev && len(todos) == 0 {
        hasPrev = false
    }

    return todos, nextCursor, prevCursor, hasNext, hasPrev, nil
}
```

Add imports for `cursor` and `fmt`:
```go
"github.com/IDTS-LAB/go-codebase/internal/shared/cursor"
```

- [ ] **Step 2: Rewrite `Search` with cursor pagination**

Replace lines 146-201 (current `Search` method):

```go
func (r *todoRepository) Search(ctx context.Context, query string, cursorArg *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error) {
    searchPattern := "%" + query + "%"

    args := []interface{}{searchPattern}
    whereClause := "WHERE deleted_at IS NULL AND (title ILIKE $1 OR description ILIKE $1)"
    nextPos := 2

    if r.tenantConfig != nil && r.tenantConfig.Enabled {
        tenantID := middleware.GetTenantID(ctx)
        if tenantID != "" {
            whereClause += fmt.Sprintf(" AND tenant_id = $%d", nextPos)
            args = append(args, tenantID)
            nextPos++
        }
    }

    if cursorArg != nil {
        c, err := cursor.Decode(*cursorArg)
        if err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("invalid cursor: %w", err)
        }
        whereClause += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", nextPos, nextPos+1)
        args = append(args, c.Timestamp, c.ID)
        nextPos += 2
    }

    dataQuery := fmt.Sprintf("SELECT id, title, description, completed, created_at, updated_at, deleted_at FROM todos %s ORDER BY created_at DESC, id DESC LIMIT $%d", whereClause, nextPos)
    dataArgs := append(args, limit+1)

    rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
    if err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("search todos: %w", err)
    }
    defer rows.Close()

    var todos []*entity.Todo
    for rows.Next() {
        var t entity.Todo
        var deletedAt sql.NullTime
        if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt, &deletedAt); err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("scan todo: %w", err)
        }
        if deletedAt.Valid {
            t.DeletedAt = &deletedAt.Time
        }
        todos = append(todos, &t)
    }
    if err := rows.Err(); err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("rows iteration: %w", err)
    }

    hasNext := len(todos) > limit
    if hasNext {
        todos = todos[:limit]
    }

    var nextCursor *string
    var prevCursor *string
    if len(todos) > 0 {
        last := todos[len(todos)-1]
        nc := cursor.Encode(last.CreatedAt, last.ID)
        nextCursor = &nc

        first := todos[0]
        pc := cursor.Encode(first.CreatedAt, first.ID)
        prevCursor = &pc
    }

    hasPrev := cursorArg != nil
    if hasPrev && len(todos) == 0 {
        hasPrev = false
    }

    return todos, nextCursor, prevCursor, hasNext, hasPrev, nil
}
```

- [ ] **Step 3: Build check**

```bash
go build ./...
```

Expected: pass.

- [ ] **Step 4: Commit**

```bash
git add internal/todo/infrastructure/persistence/todo_repository.go
git commit -m "feat: implement cursor pagination in todo repository (GetAll, Search)"
```

---

### Task 9: Implement cursor pagination in user repository (List)

**Files:**
- Modify: `internal/user/infrastructure/persistence/user_repository.go`

- [ ] **Step 1: Rewrite `List` with cursor pagination**

Replace lines 24-80 (the current `List` method):

```go
func (r *userRepository) List(ctx context.Context, cursorArg *string, limit int) ([]*entity.User, *string, *string, bool, bool, error) {
    args := []interface{}{}
    whereClause := "WHERE u.deleted_at IS NULL"

    if r.tenantConfig != nil && r.tenantConfig.Enabled {
        tenantID := middleware.GetTenantID(ctx)
        if tenantID != "" {
            whereClause += fmt.Sprintf(" AND u.tenant_id = $%d", len(args)+1)
            args = append(args, tenantID)
        }
    }

    nextPos := len(args) + 1
    if cursorArg != nil {
        c, err := cursor.Decode(*cursorArg)
        if err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("invalid cursor: %w", err)
        }
        whereClause += fmt.Sprintf(" AND (u.created_at, u.id) < ($%d, $%d)", nextPos, nextPos+1)
        args = append(args, c.Timestamp, c.ID)
        nextPos += 2
    }

    dataQuery := fmt.Sprintf("SELECT u.id, u.email, u.name, u.is_active, u.created_at, u.updated_at, u.deleted_at FROM users u %s ORDER BY u.created_at DESC, u.id DESC LIMIT $%d", whereClause, nextPos)
    dataArgs := append(args, limit+1)

    rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
    if err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("list users: %w", err)
    }
    defer rows.Close()

    var users []*entity.User
    for rows.Next() {
        var u entity.User
        var deletedAt sql.NullTime
        if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &deletedAt); err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("scan user: %w", err)
        }
        if deletedAt.Valid {
            u.DeletedAt = &deletedAt.Time
        }
        users = append(users, &u)
    }
    if err := rows.Err(); err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("rows iteration: %w", err)
    }

    hasNext := len(users) > limit
    if hasNext {
        users = users[:limit]
    }

    var nextCursor *string
    var prevCursor *string
    if len(users) > 0 {
        last := users[len(users)-1]
        nc := cursor.Encode(last.CreatedAt, last.ID)
        nextCursor = &nc

        first := users[0]
        pc := cursor.Encode(first.CreatedAt, first.ID)
        prevCursor = &pc
    }

    hasPrev := cursorArg != nil
    if hasPrev && len(users) == 0 {
        hasPrev = false
    }

    return users, nextCursor, prevCursor, hasNext, hasPrev, nil
}
```

Add imports:
```go
"github.com/IDTS-LAB/go-codebase/internal/shared/cursor"
```

- [ ] **Step 2: Build check**

```bash
go build ./...
```

Expected: pass.

- [ ] **Step 3: Commit**

```bash
git add internal/user/infrastructure/persistence/user_repository.go
git commit -m "feat: implement cursor pagination in user repository (List)"
```

---

### Task 10: Implement cursor pagination in tenant repository (List)

**Files:**
- Modify: `internal/tenant/infrastructure/persistence/tenant_repository.go`

- [ ] **Step 1: Rewrite `List` with cursor pagination**

Replace the current `List` method in `internal/tenant/infrastructure/persistence/tenant_repository.go` (around line 66):

```go
func (r *tenantRepository) List(ctx context.Context, cursorArg *string, limit int) ([]entity.Tenant, *string, *string, bool, bool, error) {
    args := []interface{}{}

    nextPos := 1
    query := "SELECT id, name, slug, domain, settings, is_active, created_at, updated_at FROM tenants"

    if cursorArg != nil {
        c, err := cursor.Decode(*cursorArg)
        if err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("invalid cursor: %w", err)
        }
        query += fmt.Sprintf(" WHERE (created_at, id) < ($%d, $%d)", nextPos, nextPos+1)
        args = append(args, c.Timestamp, c.ID)
        nextPos += 2
    }

    query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", nextPos)
    dataArgs := append(args, limit+1)

    rows, err := r.db.QueryContext(ctx, query, dataArgs...)
    if err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("list tenants: %w", err)
    }
    defer rows.Close()

    var tenants []entity.Tenant
    for rows.Next() {
        var t entity.Tenant
        if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Domain, &t.Settings, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("scan tenant: %w", err)
        }
        tenants = append(tenants, t)
    }
    if err := rows.Err(); err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("rows iteration: %w", err)
    }

    hasNext := len(tenants) > limit
    if hasNext {
        tenants = tenants[:limit]
    }

    var nextCursor *string
    var prevCursor *string
    if len(tenants) > 0 {
        last := tenants[len(tenants)-1]
        nc := cursor.Encode(last.CreatedAt, last.ID)
        nextCursor = &nc

        first := tenants[0]
        pc := cursor.Encode(first.CreatedAt, first.ID)
        prevCursor = &pc
    }

    hasPrev := cursorArg != nil
    if hasPrev && len(tenants) == 0 {
        hasPrev = false
    }

    return tenants, nextCursor, prevCursor, hasNext, hasPrev, nil
}
```

Add imports:
```go
"github.com/IDTS-LAB/go-codebase/internal/shared/cursor"
```

Remove unused `sqlc` import if `CountTenants` and sqlc `ListTenants` are no longer used.

- [ ] **Step 2: Build check**

```bash
go build ./...
```

Expected: pass.

- [ ] **Step 3: Commit**

```bash
git add internal/tenant/infrastructure/persistence/tenant_repository.go
git commit -m "feat: implement cursor pagination in tenant repository (List)"
```

---

### Task 11: Implement cursor pagination in authz repositories (role/permission GetAll)

**Files:**
- Modify: `internal/authorization/infrastructure/persistence/role_repository.go`
- Modify: `internal/authorization/infrastructure/persistence/permission_repository.go`

- [ ] **Step 1: Rewrite `roleRepository.GetAll`**

Replace the current `GetAll` in `internal/authorization/infrastructure/persistence/role_repository.go` (lines 66-122):

```go
func (r *roleRepository) GetAll(ctx context.Context, cursorArg *string, limit int) ([]*entity.Role, *string, *string, bool, bool, error) {
    args := []interface{}{}
    whereClause := "WHERE deleted_at IS NULL"

    if r.tenantConfig != nil && r.tenantConfig.Enabled {
        tenantID := middleware.GetTenantID(ctx)
        if tenantID != "" {
            whereClause += fmt.Sprintf(" AND tenant_id = $%d", len(args)+1)
            args = append(args, tenantID)
        }
    }

    nextPos := len(args) + 1
    if cursorArg != nil {
        c, err := cursor.Decode(*cursorArg)
        if err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("invalid cursor: %w", err)
        }
        whereClause += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", nextPos, nextPos+1)
        args = append(args, c.Timestamp, c.ID)
        nextPos += 2
    }

    dataQuery := fmt.Sprintf("SELECT id, name, description, created_at, updated_at, deleted_at FROM roles %s ORDER BY created_at DESC, id DESC LIMIT $%d", whereClause, nextPos)
    dataArgs := append(args, limit+1)

    rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
    if err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("query roles: %w", err)
    }
    defer rows.Close()

    var roles []*entity.Role
    for rows.Next() {
        var rl entity.Role
        var deletedAt sql.NullTime
        if err := rows.Scan(&rl.ID, &rl.Name, &rl.Description, &rl.CreatedAt, &rl.UpdatedAt, &deletedAt); err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("scan role: %w", err)
        }
        if deletedAt.Valid {
            rl.DeletedAt = &deletedAt.Time
        }
        roles = append(roles, &rl)
    }
    if err := rows.Err(); err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("rows iteration: %w", err)
    }

    hasNext := len(roles) > limit
    if hasNext {
        roles = roles[:limit]
    }

    var nextCursor *string
    var prevCursor *string
    if len(roles) > 0 {
        last := roles[len(roles)-1]
        nc := cursor.Encode(last.CreatedAt, last.ID)
        nextCursor = &nc

        first := roles[0]
        pc := cursor.Encode(first.CreatedAt, first.ID)
        prevCursor = &pc
    }

    hasPrev := cursorArg != nil
    if hasPrev && len(roles) == 0 {
        hasPrev = false
    }

    return roles, nextCursor, prevCursor, hasNext, hasPrev, nil
}
```

Add imports:
```go
"github.com/IDTS-LAB/go-codebase/internal/shared/cursor"
```

- [ ] **Step 2: Rewrite `permissionRepository.GetAll`**

Same pattern as role — replace the current `GetAll` in `internal/authorization/infrastructure/persistence/permission_repository.go` (lines 68-124) with the same cursor pagination pattern, adjusting the SELECT columns to include `resource, action`:

```go
func (r *permissionRepository) GetAll(ctx context.Context, cursorArg *string, limit int) ([]*entity.Permission, *string, *string, bool, bool, error) {
    args := []interface{}{}
    whereClause := "WHERE deleted_at IS NULL"

    if r.tenantConfig != nil && r.tenantConfig.Enabled {
        tenantID := middleware.GetTenantID(ctx)
        if tenantID != "" {
            whereClause += fmt.Sprintf(" AND tenant_id = $%d", len(args)+1)
            args = append(args, tenantID)
        }
    }

    nextPos := len(args) + 1
    if cursorArg != nil {
        c, err := cursor.Decode(*cursorArg)
        if err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("invalid cursor: %w", err)
        }
        whereClause += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", nextPos, nextPos+1)
        args = append(args, c.Timestamp, c.ID)
        nextPos += 2
    }

    dataQuery := fmt.Sprintf("SELECT id, name, description, resource, action, created_at, updated_at, deleted_at FROM permissions %s ORDER BY created_at DESC, id DESC LIMIT $%d", whereClause, nextPos)
    dataArgs := append(args, limit+1)

    rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
    if err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("query permissions: %w", err)
    }
    defer rows.Close()

    var perms []*entity.Permission
    for rows.Next() {
        var p entity.Permission
        var deletedAt sql.NullTime
        if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Resource, &p.Action, &p.CreatedAt, &p.UpdatedAt, &deletedAt); err != nil {
            return nil, nil, nil, false, false, fmt.Errorf("scan permission: %w", err)
        }
        if deletedAt.Valid {
            p.DeletedAt = &deletedAt.Time
        }
        perms = append(perms, &p)
    }
    if err := rows.Err(); err != nil {
        return nil, nil, nil, false, false, fmt.Errorf("rows iteration: %w", err)
    }

    hasNext := len(perms) > limit
    if hasNext {
        perms = perms[:limit]
    }

    var nextCursor *string
    var prevCursor *string
    if len(perms) > 0 {
        last := perms[len(perms)-1]
        nc := cursor.Encode(last.CreatedAt, last.ID)
        nextCursor = &nc

        first := perms[0]
        pc := cursor.Encode(first.CreatedAt, first.ID)
        prevCursor = &pc
    }

    hasPrev := cursorArg != nil
    if hasPrev && len(perms) == 0 {
        hasPrev = false
    }

    return perms, nextCursor, prevCursor, hasNext, hasPrev, nil
}
```

- [ ] **Step 3: Build check**

```bash
go build ./...
```

Expected: pass.

- [ ] **Step 4: Commit**

```bash
git add internal/authorization/infrastructure/persistence/role_repository.go internal/authorization/infrastructure/persistence/permission_repository.go
git commit -m "feat: implement cursor pagination in authz repositories (role, permission GetAll)"
```

---

### Task 12: Update todo domain service for cursor pagination

**Files:**
- Modify: `internal/todo/domain/service/todo_domain_service.go`

- [ ] **Step 1: Update `ListTodos` and `SearchTodos` signatures**

Replace lines 46-48 and 88-89:

```go
func (s *TodoDomainService) ListTodos(ctx context.Context, cursor *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error) {
    return s.repo.GetAll(ctx, cursor, limit)
}

func (s *TodoDomainService) SearchTodos(ctx context.Context, query string, cursor *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error) {
    return s.repo.Search(ctx, query, cursor, limit)
}
```

- [ ] **Step 2: Build check**

```bash
go build ./...
```

Expected: pass.

- [ ] **Step 3: Commit**

```bash
git add internal/todo/domain/service/todo_domain_service.go
git commit -m "feat: update todo domain service for cursor pagination signatures"
```

---

### Task 13: Update CQRS query handlers for cursor pagination

**Files:**
- Modify: `internal/todo/application/query/list_todos.go`
- Modify: `internal/todo/application/query/search_todos.go`
- Modify: `internal/user/application/query/list_users.go`
- Modify: `internal/tenant/application/query/list_tenants.go`
- Modify: `internal/authorization/application/query/list_roles.go`
- Modify: `internal/authorization/application/query/list_permissions.go`

- [ ] **Step 1: Update `ListTodosHandler`**

Replace `internal/todo/application/query/list_todos.go`:

```go
package query

import (
    "context"
    "github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
)

type ListTodosQuery struct {
    Cursor *string
    Limit  int
}

type ListTodosResult struct {
    Todos      []any
    NextCursor *string
    PrevCursor *string
    HasNext    bool
    HasPrev    bool
    Limit      int
}

type ListTodosHandler struct {
    domainSvc *service.TodoDomainService
}

func NewListTodosHandler(domainSvc *service.TodoDomainService) *ListTodosHandler {
    return &ListTodosHandler{domainSvc: domainSvc}
}

func (h *ListTodosHandler) Handle(ctx context.Context, q any) (any, error) {
    query := q.(ListTodosQuery)
    todos, nextCursor, prevCursor, hasNext, hasPrev, err := h.domainSvc.ListTodos(ctx, query.Cursor, query.Limit)
    if err != nil {
        return nil, err
    }
    return ListTodosResult{
        Todos:      todos,
        NextCursor: nextCursor,
        PrevCursor: prevCursor,
        HasNext:    hasNext,
        HasPrev:    hasPrev,
        Limit:      query.Limit,
    }, nil
}
```

- [ ] **Step 2: Update `SearchTodosHandler`**

Replace `internal/todo/application/query/search_todos.go`:

```go
package query

import (
    "context"
    "github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
)

type SearchTodosQuery struct {
    Query  string
    Cursor *string
    Limit  int
}

type SearchTodosResult struct {
    Todos      []any
    NextCursor *string
    PrevCursor *string
    HasNext    bool
    HasPrev    bool
    Limit      int
}

type SearchTodosHandler struct {
    domainSvc *service.TodoDomainService
}

func NewSearchTodosHandler(domainSvc *service.TodoDomainService) *SearchTodosHandler {
    return &SearchTodosHandler{domainSvc: domainSvc}
}

func (h *SearchTodosHandler) Handle(ctx context.Context, q any) (any, error) {
    query := q.(SearchTodosQuery)
    todos, nextCursor, prevCursor, hasNext, hasPrev, err := h.domainSvc.SearchTodos(ctx, query.Query, query.Cursor, query.Limit)
    if err != nil {
        return nil, err
    }
    return SearchTodosResult{
        Todos:      todos,
        NextCursor: nextCursor,
        PrevCursor: prevCursor,
        HasNext:    hasNext,
        HasPrev:    hasPrev,
        Limit:      query.Limit,
    }, nil
}
```

- [ ] **Step 3: Update `ListUsersHandler`**

Replace `internal/user/application/query/list_users.go`:

```go
package query

import (
    "context"
    authEntity "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
    "github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
)

type ListUsersQuery struct {
    Cursor *string
    Limit  int
}

type ListUsersResult struct {
    Users      []*authEntity.User
    NextCursor *string
    PrevCursor *string
    HasNext    bool
    HasPrev    bool
    Limit      int
}

type ListUsersHandler struct {
    repo repository.UserRepository
}

func NewListUsersHandler(repo repository.UserRepository) *ListUsersHandler {
    return &ListUsersHandler{repo: repo}
}

func (h *ListUsersHandler) Handle(ctx context.Context, query any) (any, error) {
    q := query.(ListUsersQuery)
    users, nextCursor, prevCursor, hasNext, hasPrev, err := h.repo.List(ctx, q.Cursor, q.Limit)
    if err != nil {
        return nil, err
    }
    return ListUsersResult{
        Users:      users,
        NextCursor: nextCursor,
        PrevCursor: prevCursor,
        HasNext:    hasNext,
        HasPrev:    hasPrev,
        Limit:      q.Limit,
    }, nil
}
```

- [ ] **Step 4: Update `ListTenantsHandler`**

Replace `internal/tenant/application/query/list_tenants.go`:

```go
package query

import (
    "context"
    "github.com/IDTS-LAB/go-codebase/internal/tenant/application/dto"
    "github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
)

type ListTenantsQuery struct {
    Cursor *string
    Limit  int
}

type ListTenantsHandler struct {
    repo repository.TenantRepository
}

func NewListTenantsHandler(repo repository.TenantRepository) *ListTenantsHandler {
    return &ListTenantsHandler{repo: repo}
}

func (h *ListTenantsHandler) Handle(ctx context.Context, query any) (any, error) {
    q := query.(ListTenantsQuery)
    tenants, nextCursor, prevCursor, hasNext, hasPrev, err := h.repo.List(ctx, q.Cursor, q.Limit)
    if err != nil {
        return nil, err
    }

    responses := make([]dto.TenantResponse, len(tenants))
    for i, t := range tenants {
        responses[i] = dto.TenantResponse{
            ID:        t.ID.String(),
            Name:      t.Name,
            Slug:      t.Slug,
            Domain:    t.Domain,
            Settings:  t.Settings,
            IsActive:  t.IsActive,
            CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
            UpdatedAt: t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
        }
    }

    return dto.TenantListResponse{
        Tenants:    responses,
        NextCursor: nextCursor,
        PrevCursor: prevCursor,
        HasNext:    hasNext,
        HasPrev:    hasPrev,
        Limit:      q.Limit,
    }, nil
}
```

- [ ] **Step 5: Update `ListRolesHandler`**

Replace `internal/authorization/application/query/list_roles.go`:

```go
package query

import (
    "context"
    "github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
    "github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
)

type ListRolesQuery struct {
    Cursor *string
    Limit  int
}

type ListRolesResult struct {
    Roles      []*entity.Role
    NextCursor *string
    PrevCursor *string
    HasNext    bool
    HasPrev    bool
}

type ListRolesHandler struct {
    roleRepo repository.RoleRepository
}

func NewListRolesHandler(roleRepo repository.RoleRepository) *ListRolesHandler {
    return &ListRolesHandler{roleRepo: roleRepo}
}

func (h *ListRolesHandler) Handle(ctx context.Context, query any) (any, error) {
    q := query.(ListRolesQuery)
    roles, nextCursor, prevCursor, hasNext, hasPrev, err := h.roleRepo.GetAll(ctx, q.Cursor, q.Limit)
    if err != nil {
        return nil, err
    }
    return ListRolesResult{
        Roles:      roles,
        NextCursor: nextCursor,
        PrevCursor: prevCursor,
        HasNext:    hasNext,
        HasPrev:    hasPrev,
    }, nil
}
```

- [ ] **Step 6: Update `ListPermissionsHandler`**

Replace `internal/authorization/application/query/list_permissions.go`:

```go
package query

import (
    "context"
    "github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
    "github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
)

type ListPermissionsQuery struct {
    Cursor *string
    Limit  int
}

type ListPermissionsResult struct {
    Permissions []*entity.Permission
    NextCursor  *string
    PrevCursor  *string
    HasNext     bool
    HasPrev     bool
}

type ListPermissionsHandler struct {
    permRepo repository.PermissionRepository
}

func NewListPermissionsHandler(permRepo repository.PermissionRepository) *ListPermissionsHandler {
    return &ListPermissionsHandler{permRepo: permRepo}
}

func (h *ListPermissionsHandler) Handle(ctx context.Context, query any) (any, error) {
    q := query.(ListPermissionsQuery)
    permissions, nextCursor, prevCursor, hasNext, hasPrev, err := h.permRepo.GetAll(ctx, q.Cursor, q.Limit)
    if err != nil {
        return nil, err
    }
    return ListPermissionsResult{
        Permissions: permissions,
        NextCursor:  nextCursor,
        PrevCursor:  prevCursor,
        HasNext:     hasNext,
        HasPrev:     hasPrev,
    }, nil
}
```

- [ ] **Step 7: Update `TenantListResponse` DTO**

Add cursor fields to `internal/tenant/application/dto/tenant.go`:

```go
type TenantListResponse struct {
    Tenants    []TenantResponse `json:"tenants"`
    NextCursor *string          `json:"next_cursor,omitempty"`
    PrevCursor *string          `json:"prev_cursor,omitempty"`
    HasNext    bool             `json:"has_next"`
    HasPrev    bool             `json:"has_prev"`
    Limit      int              `json:"limit"`
}
```

- [ ] **Step 8: Build check**

```bash
go build ./...
```

Expected: pass (some compilation errors remaining in http handlers, which will be fixed next).

- [ ] **Step 9: Commit**

```bash
git add internal/todo/application/query/ internal/user/application/query/ internal/tenant/application/query/ internal/tenant/application/dto/ internal/authorization/application/query/
git commit -m "feat: update CQRS query handlers for cursor pagination"
```

---

### Task 14: Update HTTP handlers for cursor pagination

**Files:**
- Modify: `internal/todo/interfaces/http/handlers.go`
- Modify: `internal/user/interfaces/http/handler.go`
- Modify: `internal/tenant/interfaces/http/handlers.go`
- Modify: `internal/authorization/interfaces/http/handlers.go`

- [ ] **Step 1: Update `TodoHandler.ListTodos` and `SearchTodos`**

Replace lines 76-95 (`ListTodos`) and 243-264 (`SearchTodos`) in `internal/todo/interfaces/http/handlers.go`:

```go
func (h *Handler) ListTodos(w http.ResponseWriter, r *http.Request) {
    cursorStr := r.URL.Query().Get("cursor")
    limit := 20
    if l := r.URL.Query().Get("limit"); l != "" {
        if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
            limit = n
        }
    }

    var cursor *string
    if cursorStr != "" {
        cursor = &cursorStr
    }

    resp, err := h.queryBus.Ask(r.Context(), query.ListTodosQuery{Cursor: cursor, Limit: limit})
    if err != nil {
        utils.MapError(w, err)
        return
    }
    result := resp.(query.ListTodosResult)
    utils.RespondCursorPaginated(w, result.Todos, result.NextCursor, result.PrevCursor, result.HasNext, result.HasPrev, result.Limit)
}

func (h *Handler) SearchTodos(w http.ResponseWriter, r *http.Request) {
    queryStr := r.URL.Query().Get("q")
    if queryStr == "" {
        utils.RespondBadRequest(w, "search query is required")
        return
    }
    cursorStr := r.URL.Query().Get("cursor")
    limit := 20
    if l := r.URL.Query().Get("limit"); l != "" {
        if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
            limit = n
        }
    }

    var cursor *string
    if cursorStr != "" {
        cursor = &cursorStr
    }

    resp, err := h.queryBus.Ask(r.Context(), query.SearchTodosQuery{Query: queryStr, Cursor: cursor, Limit: limit})
    if err != nil {
        utils.MapError(w, err)
        return
    }
    result := resp.(query.SearchTodosResult)
    utils.RespondCursorPaginated(w, result.Todos, result.NextCursor, result.PrevCursor, result.HasNext, result.HasPrev, result.Limit)
}
```

Add `"strconv"` and `"github.com/IDTS-LAB/go-codebase/internal/todo/application/query"` to imports.

- [ ] **Step 2: Update `UserHandler.List`**

Replace lines 72-99 in `internal/user/interfaces/http/handler.go`:

```go
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
    cursorStr := r.URL.Query().Get("cursor")
    limit := 20
    if l := r.URL.Query().Get("limit"); l != "" {
        if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
            limit = n
        }
    }

    var cursor *string
    if cursorStr != "" {
        cursor = &cursorStr
    }

    resp, err := h.queryBus.Ask(r.Context(), query.ListUsersQuery{Cursor: cursor, Limit: limit})
    if err != nil {
        utils.MapError(w, err)
        return
    }

    result := resp.(query.ListUsersResult)
    usersResp := make([]UserResponse, len(result.Users))
    for i, u := range result.Users {
        usersResp[i] = userToResponse(u)
    }

    utils.RespondCursorPaginated(w, usersResp, result.NextCursor, result.PrevCursor, result.HasNext, result.HasPrev, result.Limit)
}
```

- [ ] **Step 3: Update `TenantHandler.List`**

Replace lines 53-72 in `internal/tenant/interfaces/http/handlers.go`:

```go
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
    cursorStr := r.URL.Query().Get("cursor")
    limit := 20
    if l := r.URL.Query().Get("limit"); l != "" {
        if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
            limit = n
        }
    }

    var cursor *string
    if cursorStr != "" {
        cursor = &cursorStr
    }

    resp, err := h.queryBus.Ask(r.Context(), query.ListTenantsQuery{Cursor: cursor, Limit: limit})
    if err != nil {
        utils.MapError(w, err)
        return
    }
    listResp := resp.(dto.TenantListResponse)
    utils.RespondCursorPaginated(w, listResp.Tenants, listResp.NextCursor, listResp.PrevCursor, listResp.HasNext, listResp.HasPrev, listResp.Limit)
}
```

- [ ] **Step 4: Update `AuthzHandler.ListRoles` and `ListPermissions`**

Replace lines 69-84 and 199-214 in `internal/authorization/interfaces/http/handlers.go`:

```go
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
    cursorStr := r.URL.Query().Get("cursor")
    limit := 20
    if l := r.URL.Query().Get("limit"); l != "" {
        if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
            limit = n
        }
    }

    var cursor *string
    if cursorStr != "" {
        cursor = &cursorStr
    }

    resp, err := h.queryBus.Ask(r.Context(), query.ListRolesQuery{Cursor: cursor, Limit: limit})
    if err != nil {
        utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        return
    }
    result := resp.(query.ListRolesResult)
    utils.RespondCursorPaginated(w, result.Roles, result.NextCursor, result.PrevCursor, result.HasNext, result.HasPrev, limit)
}

func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
    cursorStr := r.URL.Query().Get("cursor")
    limit := 20
    if l := r.URL.Query().Get("limit"); l != "" {
        if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
            limit = n
        }
    }

    var cursor *string
    if cursorStr != "" {
        cursor = &cursorStr
    }

    resp, err := h.queryBus.Ask(r.Context(), query.ListPermissionsQuery{Cursor: cursor, Limit: limit})
    if err != nil {
        utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        return
    }
    result := resp.(query.ListPermissionsResult)
    utils.RespondCursorPaginated(w, result.Permissions, result.NextCursor, result.PrevCursor, result.HasNext, result.HasPrev, limit)
}
```

- [ ] **Step 5: Build check**

```bash
go build ./...
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/todo/interfaces/http/handlers.go internal/user/interfaces/http/handler.go internal/tenant/interfaces/http/handlers.go internal/authorization/interfaces/http/handlers.go
git commit -m "feat: update HTTP handlers for cursor pagination"
```

---

### Task 15: Add migration for casbin_rule table

**Files:**
- Create: `migrations/010_add_casbin_rule_table.sql`

- [ ] **Step 1: Create migration**

**File:** `migrations/010_add_casbin_rule_table.sql`

```sql
-- +goose Up
CREATE TABLE casbin_rule (
    id SERIAL PRIMARY KEY,
    ptype VARCHAR(100) NOT NULL,
    v0 VARCHAR(255),
    v1 VARCHAR(255),
    v2 VARCHAR(255),
    v3 VARCHAR(255),
    v4 VARCHAR(255),
    v5 VARCHAR(255)
);

CREATE INDEX idx_casbin_rule_ptype ON casbin_rule(ptype);
CREATE INDEX idx_casbin_rule_v0 ON casbin_rule(v0);

-- +goose Down
DROP TABLE IF EXISTS casbin_rule;
```

- [ ] **Step 2: Commit**

```bash
git add migrations/010_add_casbin_rule_table.sql
git commit -m "feat: add casbin_rule table migration"
```

---

### Task 16: Implement custom Casbin adapter

**Files:**
- Rewrite: `internal/authorization/infrastructure/casbin/adapter.go`

**Interfaces:**
- Consumes: `*sql.DB`, `casbin/model.Model`
- Produces: `*Adapter` implementing `persist.Adapter` (LoadPolicy, SavePolicy, AddPolicy, RemovePolicy, RemoveFilteredPolicy)

- [ ] **Step 1: Rewrite adapter.go**

Replace entire `internal/authorization/infrastructure/casbin/adapter.go`:

```go
package casbin

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    "github.com/casbin/casbin/v2/model"
    "github.com/casbin/casbin/v2/persist"
)

type Adapter struct {
    db *sql.DB
}

func NewAdapter(db *sql.DB) *Adapter {
    return &Adapter{db: db}
}

func (a *Adapter) LoadPolicy(model model.Model) error {
    rows, err := a.db.QueryContext(context.Background(),
        "SELECT ptype, v0, v1, v2, v3, v4, v5 FROM casbin_rule")
    if err != nil {
        return fmt.Errorf("load casbin policies: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        var ptype string
        var v0, v1, v2, v3, v4, v5 sql.NullString
        if err := rows.Scan(&ptype, &v0, &v1, &v2, &v3, &v4, &v5); err != nil {
            return fmt.Errorf("scan casbin rule: %w", err)
        }
        line := persist.ValuesToSlice([]string{ptype, v0.String, v1.String, v2.String, v3.String, v4.String, v5.String})
        persist.LoadPolicyLine(line, model)
    }
    return rows.Err()
}

func (a *Adapter) SavePolicy(model model.Model) error {
    // Not needed — policies are synced via AddPolicy/RemovePolicy in command handlers
    return nil
}

func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
    args := make([]interface{}, 7)
    args[0] = ptype
    for i := 1; i <= 6; i++ {
        if i-1 < len(rule) {
            args[i] = rule[i-1]
        } else {
            args[i] = ""
        }
    }
    _, err := a.db.ExecContext(context.Background(),
        "INSERT INTO casbin_rule (ptype, v0, v1, v2, v3, v4, v5) VALUES ($1, $2, $3, $4, $5, $6, $7)",
        args...)
    if err != nil {
        return fmt.Errorf("add casbin policy: %w", err)
    }
    return nil
}

func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
    query := "DELETE FROM casbin_rule WHERE ptype = $1"
    args := []interface{}{ptype}

    for i, r := range rule {
        pos := i + 2
        query += fmt.Sprintf(" AND v%d = $%d", i, pos)
        args = append(args, r)
    }

    _, err := a.db.ExecContext(context.Background(), query, args...)
    if err != nil {
        return fmt.Errorf("remove casbin policy: %w", err)
    }
    return nil
}

func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
    query := "DELETE FROM casbin_rule WHERE ptype = $1"
    args := []interface{}{ptype}

    for i, fv := range fieldValues {
        if fv == "" {
            continue
        }
        vi := fieldIndex + i
        query += fmt.Sprintf(" AND v%d = $%d", vi, len(args)+1)
        args = append(args, fv)
    }

    if len(args) == 1 {
        return nil // no filter values provided
    }

    _, err := a.db.ExecContext(context.Background(), query, args...)
    if err != nil {
        return fmt.Errorf("remove filtered casbin policy: %w", err)
    }
    return nil
}

func (a *Adapter) AddPolicies(sec string, ptype string, rules [][]string) error {
    tx, err := a.db.Begin()
    if err != nil {
        return fmt.Errorf("begin tx for add policies: %w", err)
    }
    defer tx.Rollback()

    stmt, err := tx.PrepareContext(context.Background(),
        "INSERT INTO casbin_rule (ptype, v0, v1, v2, v3, v4, v5) VALUES ($1, $2, $3, $4, $5, $6, $7)")
    if err != nil {
        return fmt.Errorf("prepare add policies: %w", err)
    }
    defer stmt.Close()

    for _, rule := range rules {
        args := make([]interface{}, 7)
        args[0] = ptype
        for i := 1; i <= 6; i++ {
            if i-1 < len(rule) {
                args[i] = rule[i-1]
            } else {
                args[i] = ""
            }
        }
        if _, err := stmt.Exec(args...); err != nil {
            return fmt.Errorf("add policy in batch: %w", err)
        }
    }

    return tx.Commit()
}

func (a *Adapter) RemovePolicies(sec string, ptype string, rules [][]string) error {
    tx, err := a.db.Begin()
    if err != nil {
        return fmt.Errorf("begin tx for remove policies: %w", err)
    }
    defer tx.Rollback()

    for _, rule := range rules {
        query := "DELETE FROM casbin_rule WHERE ptype = $1"
        args := []interface{}{ptype}
        for i, r := range rule {
            pos := i + 2
            query += fmt.Sprintf(" AND v%d = $%d", i, pos)
            args = append(args, r)
        }
        if _, err := tx.ExecContext(context.Background(), query, args...); err != nil {
            return fmt.Errorf("remove policy in batch: %w", err)
        }
    }

    return tx.Commit()
}

var _ persist.Adapter = (*Adapter)(nil)
```

- [ ] **Step 2: Build check**

```bash
go build ./...
```

Expected: pass.

- [ ] **Step 3: Commit**

```bash
git add internal/authorization/infrastructure/casbin/adapter.go
git commit -m "feat: implement custom Casbin adapter for casbin_rule table"
```

---

### Task 17: Update Casbin enforcer to use adapter

**Files:**
- Modify: `internal/authorization/infrastructure/casbin/enforcer.go`
- Delete: `internal/authorization/infrastructure/casbin/model.conf` (keep as-is)
- Create/Modify wire dependency for `Adapter`

- [ ] **Step 1: Update `Enforcer` to use adapter instead of PolicyLoader**

Replace `internal/authorization/infrastructure/casbin/enforcer.go`:

```go
package casbin

import (
    "context"
    _ "embed"
    "fmt"

    "github.com/casbin/casbin/v2"
    "github.com/casbin/casbin/v2/model"
    "github.com/google/uuid"
    "go.uber.org/fx"
)

//go:embed model.conf
var modelConf string

var Module = fx.Module("casbin", fx.Provide(
    NewAdapter,
    NewEnforcer,
))

type Enforcer struct {
    enforcer *casbin.CachedEnforcer
    adapter  *Adapter
}

func NewEnforcer(adapter *Adapter) (*Enforcer, error) {
    m, err := model.NewModelFromString(modelConf)
    if err != nil {
        return nil, fmt.Errorf("parse casbin model: %w", err)
    }

    enforcer, err := casbin.NewCachedEnforcer(m, adapter)
    if err != nil {
        return nil, fmt.Errorf("create casbin enforcer: %w", err)
    }

    e := &Enforcer{
        enforcer: enforcer,
        adapter:  adapter,
    }

    if err := e.ReloadPolicies(); err != nil {
        return nil, fmt.Errorf("load initial policies: %w", err)
    }

    return e, nil
}

func (e *Enforcer) ReloadPolicies() error {
    if err := e.enforcer.LoadPolicy(); err != nil {
        return fmt.Errorf("reload policies: %w", err)
    }
    return nil
}

func (e *Enforcer) ReloadUserPolicies(ctx context.Context, userID uuid.UUID) error {
    subject := userID.String()
    _, err := e.enforcer.RemoveFilteredPolicy(0, subject)
    if err != nil {
        return fmt.Errorf("remove user policies: %w", err)
    }

    policies, err := loadUserPolicies(ctx, e.adapter.db, userID)
    if err != nil {
        return fmt.Errorf("load user policies from db: %w", err)
    }

    for _, p := range policies {
        if _, err := e.enforcer.AddPolicy(p.Subject, p.Object, p.Action); err != nil {
            return fmt.Errorf("add user policy: %w", err)
        }
    }

    return nil
}

func (e *Enforcer) Enforce(userID uuid.UUID, resource, action string) (bool, error) {
    return e.enforcer.Enforce(userID.String(), resource, action)
}
```

- [ ] **Step 2: Add `loadUserPolicies` helper**

Add to the same file after the module var block:

```go
func loadUserPolicies(ctx context.Context, db *sql.DB, userID uuid.UUID) ([]Policy, error) {
    query := `
        SELECT ur.user_id::text, p.resource, p.action
        FROM user_roles ur
        JOIN role_permissions rp ON ur.role_id = rp.role_id
        JOIN permissions p ON rp.permission_id = p.id
        JOIN roles r ON ur.role_id = r.id
        WHERE ur.user_id = $1 AND r.deleted_at IS NULL AND p.deleted_at IS NULL`

    rows, err := db.QueryContext(ctx, query, userID)
    if err != nil {
        return nil, fmt.Errorf("load user policies: %w", err)
    }
    defer rows.Close()

    var policies []Policy
    for rows.Next() {
        var pol Policy
        if err := rows.Scan(&pol.Subject, &pol.Object, &pol.Action); err != nil {
            return nil, fmt.Errorf("scan policy: %w", err)
        }
        policies = append(policies, pol)
    }
    return policies, rows.Err()
}
```

Add `"database/sql"` to imports.

- [ ] **Step 3: Build check**

```bash
go build ./...
```

Expected: pass.

- [ ] **Step 4: Commit**

```bash
git add internal/authorization/infrastructure/casbin/enforcer.go
git commit -m "feat: update Casbin enforcer to use custom adapter"
```

---

### Task 18: Sync casbin_rule on RBAC mutations

**Files:**
- Modify: `internal/authorization/application/command/assign_role.go`
- Modify: `internal/authorization/application/command/unassign_role.go`
- Modify: `internal/authorization/application/command/assign_permission.go`
- Modify: `internal/authorization/application/command/unassign_permission.go`

Each command handler needs to call `enforcer.ReloadUserPolicies()` or `enforcer.ReloadPolicies()` after the DB mutation commits.

- [ ] **Step 1: Update `AssignRoleCommandHandler`**

In `internal/authorization/application/command/assign_role.go`, after `userRoleRepo.Assign(...)` succeeds, add:

```go
if err := h.enforcer.ReloadUserPolicies(ctx, cmd.UserID); err != nil {
    return nil, fmt.Errorf("reload user policies after role assign: %w", err)
}
```

- [ ] **Step 2: Update `UnassignRoleCommandHandler`**

Same pattern — after `userRoleRepo.Remove(...)`, add:

```go
if err := h.enforcer.ReloadUserPolicies(ctx, cmd.UserID); err != nil {
    return nil, fmt.Errorf("reload user policies after role unassign: %w", err)
}
```

- [ ] **Step 3: Update `AssignPermissionCommandHandler`**

After `rolePermRepo.Assign(...)`, call full reload since multiple users may be affected:

```go
if err := h.enforcer.ReloadPolicies(); err != nil {
    return nil, fmt.Errorf("reload policies after permission assign: %w", err)
}
```

- [ ] **Step 4: Update `UnassignPermissionCommandHandler`**

Same — after `rolePermRepo.Remove(...)`, call full reload:

```go
if err := h.enforcer.ReloadPolicies(); err != nil {
    return nil, fmt.Errorf("reload policies after permission unassign: %w", err)
}
```

- [ ] **Step 5: Build check**

```bash
go build ./...
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/authorization/application/command/
git commit -m "feat: sync casbin_rule policies on RBAC mutations"
```

---

### Task 19: Cleanup unused imports and verify build

**Files:**
- Check all modified files for unused imports
- Run full build and test

- [ ] **Step 1: Run full build**

```bash
go build ./...
```

Expected: pass with no errors.

- [ ] **Step 2: Run go vet**

```bash
go vet ./...
```

Expected: pass.

- [ ] **Step 3: Run tests**

```bash
go test ./... -count=1 2>&1 | tail -30
```

Expected: all existing tests pass. Note: some tests may reference old signatures (e.g., mock repos) — fix any test compilation errors.

- [ ] **Step 4: Commit any fixups**

```bash
git add -A
git commit -m "chore: fix tests and unused imports after cursor/sqlc/casbin changes"
```
