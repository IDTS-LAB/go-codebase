# Casbin Standard Table, Cursor Pagination, and sqlc Migration

**Date:** 2026-07-13
**Status:** Spec

## Overview

Refactor the codebase with three major changes:
1. Casbin authorization — migrate from custom PolicyLoader to standard `casbin_rule` table + DB adapter
2. Pagination — replace offset/limit with bidirectional cursor-based pagination across all domains
3. sqlc adoption — migrate simple CRUD in `user` and `todo` repositories to sqlc, keep raw SQL for dynamic filters

---

## 1. Casbin Standard Table

### Current State

- Custom `PolicyLoader` queries 4 tables (`user_roles` → `role_permissions` → `permissions` → `roles`) and loads flat `(user_id, resource, action)` policies into in-memory `CachedEnforcer`
- No `casbin_rule` table exists
- Policy reloads: full reload (all users) or per-user reload on RBAC changes

### Target State

#### New migration: `010_add_casbin_rule_table.sql`
```sql
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
```

#### Enforcer changes
- Implement custom adapter that implements `persist.Adapter` using `database/sql`
  - `LoadPolicy(model)` — loads all rules from `casbin_rule` into the enforcer model
  - `SavePolicy(model)` — persisted via direct writes from sync flow (atomic: DELETE + INSERT batch)
  - `AddPolicy(sec, ptype, rule)` — inserts single row via `casbin_rule`
  - `RemovePolicy(sec, ptype, rule)` — deletes from `casbin_rule`
- Keep `CachedEnforcer` for performance
- Enforcer initializes with `model.conf` + custom adapter
- `ReloadPolicies()` → calls `adapter.LoadPolicy()` (clears cache + reloads from `casbin_rule`)
- `ReloadUserPolicies(userID)` → deleted + re-added via adapter

#### Sync flow on RBAC mutation
When any RBAC assignment changes (assign/unassign role or permission):

1. Query the flattened user permissions: `SELECT ur.user_id::text, p.resource, p.action FROM user_roles ur JOIN role_permissions rp ...`
2. Delete existing `casbin_rule` entries for affected user(s) where `ptype = 'p'`
3. Insert new `(ptype='p', v0=user_id, v1=resource, v2=action)` tuples
4. Call `enforcer.LoadPolicy()` to sync in-memory cache

This happens in the command handlers after the DB mutation commits.

#### Model.conf (unchanged)
```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
```

The `g(r.sub, p.sub)` matcher still works — Casbin's DB adapter supports `g` grouping rules if we ever need role-based policies in the future. For now, policies remain flat `(user_id, resource, action)` tuples with `ptype = 'p'`.

#### Files affected
| File | Change |
|------|--------|
| `internal/authorization/infrastructure/casbin/adapter.go` | Rewrite as custom `persist.Adapter` implementation using `database/sql` + `casbin_rule` table |
| `internal/authorization/infrastructure/casbin/enforcer.go` | Initialize with custom adapter; update sync methods |
| `internal/authorization/application/command/assign_role.go` | After DB commit, sync user policies to `casbin_rule` |
| `internal/authorization/application/command/unassign_role.go` | Same |
| `internal/authorization/application/command/assign_permission.go` | Sync affected users to `casbin_rule` |
| `internal/authorization/application/command/unassign_permission.go` | Same |
| `migrations/010_add_casbin_rule_table.sql` | New migration |

---

## 2. Cursor Pagination

### Current State
- All list endpoints use offset/limit pagination with `page`/`per_page` query params
- Response returns `total` and `total_pages`
- Repository methods: `GetAll(ctx, offset, limit int) (items, int, error)`
- Inconsistent: User module uses `limit/offset` params instead of `page/per_page`

### Target State

#### New shared package: `internal/shared/cursor/`

```go
package cursor

type Cursor struct {
    Timestamp time.Time `json:"t"`
    ID        uuid.UUID `json:"i"`
}

func Encode(t time.Time, id uuid.UUID) string { ... }     // base64 JSON → opaque
func Decode(s string) (Cursor, error) { ... }
```

#### Repository interface changes

Before:
```go
GetAll(ctx, offset, limit int) ([]*T, int, error)
List(ctx, offset, limit int) ([]*T, int, error)
Search(ctx, query string, offset, limit int) ([]*T, int, error)
```

After:
```go
GetAll(ctx, cursor *string, limit int) (items []*T, nextCursor, prevCursor *string, hasNext, hasPrev bool, err error)
List(ctx, cursor *string, limit int) (items []*T, nextCursor, prevCursor *string, hasNext, hasPrev bool, err error)
Search(ctx, query string, cursor *string, limit int) (items []*T, nextCursor, prevCursor *string, hasNext, hasPrev bool, err error)
```

#### SQL pattern for cursor pagination (descending)

```sql
-- First page (cursor = nil):
SELECT id, ..., created_at
FROM table
WHERE deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $1

-- Subsequent pages (cursor = <timestamp, uuid>):
SELECT id, ..., created_at
FROM table
WHERE deleted_at IS NULL
  AND (created_at, id) < ($1, $2)
ORDER BY created_at DESC, id DESC
LIMIT $3

-- Previous page (reverse direction):
SELECT id, ..., created_at
FROM table
WHERE deleted_at IS NULL
  AND (created_at, id) > ($1, $2)
ORDER BY created_at ASC, id ASC
LIMIT $3
-- Then reverse the result order back to DESC
```

Implementation detail:
- Query `limit + 1` rows to detect `has_next`
- `next_cursor` = encode(created_at, id) of last item returned
- `prev_cursor` = encode(created_at, id) of first item returned (technically from the previous page)
- For forward pagination: `prev_cursor` is always computed if we store the first item's cursor from the current page
- For backward pagination (prev): query in ASC order with `>` condition, reverse results, `has_prev` = true only if there are items before the original cursor

#### CursorMeta response envelope

```go
type CursorMeta struct {
    NextCursor *string `json:"next_cursor"`
    PrevCursor *string `json:"prev_cursor"`
    HasNext    bool    `json:"has_next"`
    HasPrev    bool    `json:"has_prev"`
    Limit      int     `json:"limit"`
}
```

#### HTTP request params
```
?cursor=<base64>&limit=20
```

Default limit = 20, max = 100.

#### Dynamic filter handling (tenant + search)

For methods with dynamic WHERE clauses (`GetAll`/`List`/`Search` with optional `AND tenant_id = $N`), the cursor condition and order clause are appended after the dynamic parts. Parameter positioning is handled with `$N` incrementors — same as current approach, but with additional cursor params.

Example pattern:
```go
args := []interface{}{}
whereClause := "WHERE deleted_at IS NULL"

if tenantEnabled {
    whereClause += " AND tenant_id = $1"
    args = append(args, tenantID)
}

// Cursor position
if cursor != nil {
    c, _ := cursor.Decode(cursorStr)
    whereClause += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", len(args)+1, len(args)+2)
    args = append(args, c.Timestamp, c.ID)
}

orderClause := "ORDER BY created_at DESC, id DESC"
limitClause := fmt.Sprintf("LIMIT $%d", len(args)+1)
args = append(args, limit+1)
```

#### HTTP handler pattern

```go
cursorStr := r.URL.Query().Get("cursor")
limit := 20
if l := r.URL.Query().Get("limit"); l != "" {
    if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
        limit = n
    }
}

q := query.ListTodosQuery{Cursor: nil, Limit: limit}
if cursorStr != "" {
    q.Cursor = &cursorStr
}
result, err := bus.Dispatch(ctx, q)
```

#### DTO Response types
```go
type TodoListResponse struct {
    Todos []TodoResponse  `json:"todos"`
}
// The cursor meta is added by the utils response helper
```

#### Utils changes

New response helper:
```go
func RespondCursorPaginated[T any](w http.ResponseWriter, data []T, nextCursor, prevCursor *string, hasNext, hasPrev bool, limit int) {
    resp := APIResponse{
        Success: true,
        Data:    data,
        Meta: CursorMeta{
            NextCursor: nextCursor,
            PrevCursor: prevCursor,
            HasNext:    hasNext,
            HasPrev:    hasPrev,
            Limit:      limit,
        },
    }
    writeJSON(w, http.StatusOK, resp)
}
```

#### Domains and files affected

| Domain | Repository interface | Repo impl file | Query handlers | HTTP handlers | Routes |
|--------|---------------------|----------------|----------------|---------------|--------|
| **Todo** | `todo/domain/repository/todo_repository.go` | `todo/.../todo_repository.go` | `list_todos.go`, `search_todos.go` | `handlers.go` | — |
| **User** | `user/domain/repository/user_repository.go` | `user/.../user_repository.go` | `list_users.go` | `handler.go` | — |
| **Tenant** | `tenant/domain/repository/tenant.go` | `tenant/.../tenant_repository.go` | `list_tenants.go` | `handlers.go` | — |
| **Authz (role)** | `authorization/domain/repository/authorization_repository.go` | `authorization/.../role_repository.go` | `list_roles.go` | `handlers.go` | — |
| **Authz (perm)** | `authorization/domain/repository/authorization_repository.go` | `authorization/.../permission_repository.go` | `list_permissions.go` | `handlers.go` | — |

Also:
- `todo/domain/service/todo_domain_service.go` — update `ListTodos`/`SearchTodos` signatures
- `internal/shared/utils/utils.go` — add `CursorMeta` struct and `RespondCursorPaginated`
- `internal/shared/router/web.go` — `ResponseFormatter` middleware handles cursor meta
- `internal/shared/middleware/formatter.go` — handle cursor meta in response formatting

---

## 3. sqlc Migration for User and Todo Repositories

### Current State
Both `user` and `todo` repositories have:
- sqlc queries defined in `.sql` files
- sqlc code generated
- But repositories ignore it and use raw `database/sql` for everything

### Target

#### `internal/user/infrastructure/persistence/user_repository.go`

| Method | Current | Target |
|--------|---------|--------|
| `List` | raw SQL (dynamic tenant filter) | **Unchanged** (raw SQL) |
| `GetByID` | raw `QueryRowContext` | `sqlc.New(r.db).GetUserByID(ctx, id)` |
| `Update` | raw `ExecContext` | `sqlc.New(r.db).UpdateUser(ctx, params)` |
| `Delete` | raw `ExecContext` | `sqlc.New(r.db).DeleteUser(ctx, id)` |

#### `internal/todo/infrastructure/persistence/todo_repository.go`

| Method | Current | Target |
|--------|---------|--------|
| `GetAll` | raw SQL (dynamic tenant + cursor) | **Unchanged** (raw SQL) |
| `Search` | raw SQL (dynamic tenant + cursor) | **Unchanged** (raw SQL) |
| `Create` | raw `ExecContext` | `sqlc.New(r.db).CreateTodo(ctx, params)` |
| `GetByID` | raw `QueryRowContext` | `sqlc.New(r.db).GetTodoByID(ctx, id)` |
| `Update` | raw `ExecContext` | `sqlc.New(r.db).UpdateTodo(ctx, params)` |
| `Delete` | raw `ExecContext` | `sqlc.New(r.db).SoftDeleteTodo(ctx, id)` |

#### Cleanup sqlc query files

Remove these paginated queries from `.sql` files, then regenerate:

**`todo.sql` — remove:**
- `ListTodos`
- `CountTodos`
- `SearchTodos`
- `CountSearchTodos`

**`queries.sql` (user) — remove:**
- `ListUsers`
- `CountUsers`

**`queries.sql` (authorization) — remove:**
- `ListRoles`
- `CountRoles`
- `ListPermissions`
- `CountPermissions`

**`queries.sql` (tenant) — remove:**
- `ListTenants`
- `CountTenants`

#### Regenerate sqlc
```bash
sqlc generate
```

---

## Dependency Changes

**Add to go.mod:**
- `github.com/casbin/casbin/v2` (already in go.mod)

**No new third-party adapter** — implement custom `persist.Adapter` with existing `database/sql`.

---

## Migration Order

This work should be done in this order to minimize conflicts:

1. **Cursor package** (`internal/shared/cursor/`) — no dependencies
2. **sqlc regenerate** (cleanup paginated queries first) — no runtime impact
3. **User repository sqlc migration** — mechanical, isolated
4. **Todo repository sqlc migration** — mechanical, isolated
5. **Cursor pagination — repository layer** — change all 6 repository interfaces + implementations
6. **Cursor pagination — application layer** — update CQRS query handlers and DTOs
7. **Cursor pagination — HTTP layer** — update handlers + utils + formatter
8. **Casbin standard table** — migration + enforcer + sync commands
9. **Integration test / manual verification**
