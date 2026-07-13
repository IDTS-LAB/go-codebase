# Error Hardening: SQL Injection Safety, Stacktrace in 500 Responses, Error Log Persistence

**Date:** 2026-07-13
**Status:** Spec

## Overview

Three targeted improvements:
1. **SQL injection audit** — confirm all raw SQL queries use parameterized placeholders
2. **LIKE character escaping** — escape `%` and `_` in todo search to prevent unintended wildcard matching
3. **500 error hardening** — show full `debug.Stack()` in non-production responses, persist all 500 errors to `error_logs` table with stacktraces

---

## 1. SQL Injection Audit

### Finding

All raw SQL queries across all repositories use PostgreSQL parameterized placeholders (`$1`, `$2`, ...). The `fmt.Sprintf` calls are used exclusively for building SQL structure (WHERE clause composition, placeholder numbering) — never for interpolating values. **No injection vulnerabilities exist.**

No changes required.

### Affected files (no-op, documented for reference)

| File | Pattern |
|------|---------|
| `internal/todo/infrastructure/persistence/todo_repository.go` | `GetAll`, `Search` — cursor + tenant filters |
| `internal/user/infrastructure/persistence/user_repository.go` | `List` — cursor + tenant filters |
| `internal/tenant/infrastructure/persistence/tenant_repository.go` | `List` — cursor filters |
| `internal/authorization/infrastructure/persistence/permission_repository.go` | `GetAll` — cursor + tenant filters |
| `internal/authorization/infrastructure/persistence/role_repository.go` | `GetAll` — cursor + tenant filters |
| `internal/authorization/infrastructure/casbin/adapter.go` | `RemovePolicy`, `RemoveFilteredPolicy` — column index placeholders |

---

## 2. LIKE Character Escaping

### Problem

In `internal/todo/infrastructure/persistence/todo_repository.go:168`:

```go
searchPattern := "%" + query + "%"
```

If a user searches for `"100%"`, the `%` acts as a SQL wildcard matching any string starting with `"100"`. Similarly `_` matches any single character.

### Change

Escape `%` → `\%` and `_` → `\_` before wrapping in LIKE pattern:

```go
replacer := strings.NewReplacer(`%`, `\%`, `_`, `\_`)
searchPattern := "%" + replacer.Replace(query) + "%"
```

PostgreSQL's `ILIKE` with escaped patterns and standard `ESCAPE '\'` (default in PostgreSQL) handles this correctly.

### File

`internal/todo/infrastructure/persistence/todo_repository.go`

---

## 3. 500 Error Hardening

### Current Problems

1. `MapError` default case → `RespondInternalError("internal server error")` — generic message in all environments
2. Non-panic 500s (from handlers) are saved to `error_logs` by `ErrorRecorder` middleware **without** error detail or stacktrace
3. `ErrorRecorder` is initialized with `nil` logger — if `persistError`'s DB insert fails, it panics on nil

### Target

- **Non-production** (`APP_ENV != "production"`): 500 response includes error message + full `debug.Stack()` in the response body
- **Production**: generic `"internal server error"` (unchanged)
- **All 500s**: persisted to `error_logs` table with error message + stacktrace (previously only panics had this)
- **Bug fix**: `ErrorRecorder` receives a proper logger

### Design

#### a. Error info context helpers

**New file:** `internal/shared/utils/error_context.go`

```go
type contextKey string

const errorInfoKey contextKey = "error_info"

type ErrorInfo struct {
    Err   error
    Stack string
}

func SetErrorInfo(ctx context.Context, err error, stack string) context.Context
func GetErrorInfo(ctx context.Context) (*ErrorInfo, bool)
```

Stores the originating error and stacktrace in the request context. Middleware reads it after the handler completes.

#### b. Production flag

**Modified:** `internal/shared/utils/utils.go`

Add package-level variable:
```go
var IsProduction bool
```

Set once at startup in `internal/shared/router/router.go`:
```go
utils.IsProduction = (cfg.App.Env == "production")
```

#### c. Env-aware 500 response

**Modified:** `internal/shared/utils/utils.go`

New private helper `respond500` used by both `RespondInternalError` and `MapError`:

```go
func respond500(w http.ResponseWriter, r *http.Request, message string) {
    if IsProduction {
        RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
        return
    }
    // Non-production: include stack if available
    info, ok := GetErrorInfo(r.Context())
    stack := ""
    if ok {
        stack = info.Stack
    }
    json.NewEncoder(w).Encode(APIResponse{
        Success: false,
        Error:   &ErrorBody{Code: "INTERNAL_ERROR", Message: message},
        Stack:   stack,  // new field — omitempty in production
    })
}
```

**New field on `APIResponse`:**
```go
type APIResponse struct {
    Success bool         `json:"success"`
    Data    interface{}  `json:"data,omitempty"`
    Meta    interface{}  `json:"meta,omitempty"`
    Error   *ErrorBody   `json:"error,omitempty"`
    Stack   string       `json:"stack,omitempty"`
}
```

`Stack` omitted in production because `IsProduction` never sets it (empty string → JSON `omitempty` omits it).

#### d. MapError enhancement

**Modified:** `internal/shared/utils/handler.go`

In the default case of `MapError`, capture the stacktrace and store in context:

```go
default:
    stack := string(debug.Stack())
    ctx := SetErrorInfo(r.Context(), err, stack)
    respond500(w, r.WithContext(ctx), "internal server error")
```

And signature changes: `MapError` now takes `*http.Request` instead of just `http.ResponseWriter`. This is the only breaking signature change across all callers.

Actually, to avoid changing all callers, add a new helper:
```go
func MapErrorFromRequest(w http.ResponseWriter, r *http.Request, err error)
```

And keep the old `MapError(w, err)` delegating to it with no request context (falls back to production-safe behavior).

#### e. ErrorRecorder middleware fixes

**Modified:** `internal/shared/middleware/middleware.go`

1. Read error info from context:
```go
if info, ok := utils.GetErrorInfo(r.Context()); ok {
    msg = info.Err.Error()
    stack = info.Stack
}
```

2. Pass error message + stack to `persistError` for accurate `error_logs` entries.

#### f. Nil logger bug fix

**Modified:** `internal/shared/middleware/registry.go`

```go
// Before:
ErrorRecorder: ErrorRecorder(nil, errorRepo),

// After:
ErrorRecorder: ErrorRecorder(log, errorRepo),
```

### Response shape — non-production

```json
{
  "success": false,
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "internal server error"
  },
  "stack": "goroutine 1 [running]:\nruntime/debug.Stack(0x0)\n\t/usr/local/go/src/runtime/debug/stack.go:24 +0x65\n..."
}
```

### Response shape — production

```json
{
  "success": false,
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "internal server error"
  }
}
```

### Files affected

| File | Change |
|------|--------|
| `internal/shared/utils/error_context.go` | NEW — context helpers for error info |
| `internal/shared/utils/utils.go` | Add `IsProduction`, `Stack` field to `APIResponse`, `Stack` field to `ErrorBody` or as separate field, modify `RespondInternalError` |
| `internal/shared/utils/handler.go` | Modify `MapError` default case (stack capture + context), add `MapErrorFromRequest` |
| `internal/shared/middleware/middleware.go` | ErrorRecorder reads context, passes stack to `persistError` |
| `internal/shared/middleware/registry.go` | Fix nil logger bug |
| `internal/shared/router/router.go` | Set `utils.IsProduction` from config |
| `internal/todo/infrastructure/persistence/todo_repository.go` | Escape LIKE chars |
| All handlers calling `MapError(w, err)` | Change to `MapErrorFromRequest(w, r, err)` (3 callers) |

---

## Implementation Order

1. LIKE escaping (1 file, 1 line)
2. Error info context helpers (new file)
3. `IsProduction` flag + `Stack` field in `APIResponse`
4. `respond500` helper + update `RespondInternalError`
5. `MapErrorFromRequest` + update callers
6. `ErrorRecorder` reads context
7. Fix nil logger bug in registry
8. Set `IsProduction` in router
9. Verify build + tests
