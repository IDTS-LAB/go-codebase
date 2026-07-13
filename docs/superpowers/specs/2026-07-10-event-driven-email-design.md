# Event-Driven Email, Unified Response, Global Error Handling

**Goal:** Decouple email sending from business logic via domain events, standardize all HTTP responses into a single envelope, and centralize error handling with consistent error codes.

## Architecture

```
                   ┌──────────────────────┐
                   │    EventBus (interface)│
                   │  Publish / Subscribe  │
                   └──────┬───────────────┘
                          │
            ┌─────────────┼─────────────┐
            ▼             ▼             ▼
    InMemoryEventBus  RabbitMQBus   KafkaBus
    (sync, now)      (future)      (future)

Service → bus.Publish(event) → EventBus → EventHandler → mailer.Send*()
```

Response flow:
```
Handler → service → bus.Publish()
  ↓
utils.RespondSuccess/RespondError/RespondPaginated
  ↓
{"success": true, "data": {...}, "meta": null}
```

Global error middleware catches panics, maps domain errors to HTTP codes.

## Tech Stack

- Go 1.25, Fx DI, Chi router, PostgreSQL
- Module path: `github.com/IDTS-LAB/go-codebase`
- In-memory sync bus for now (`InMemoryEventBus` with `sync.RWMutex`)
- No external deps added

## Components

### 1. EventBus Interface

**File:** `internal/shared/events/events.go` (existing file, refactored)

```go
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(eventType string, handler Handler) error
}
```

Existing `Event` struct and `Handler` type stay unchanged. Rename current struct to `InMemoryEventBus` implementing `EventBus`.

### 2. Domain Events (New)

**File:** `internal/authentication/domain/event/auth_events.go`

| Event | Type | Fields | Trigger |
|-------|------|--------|---------|
| UserRegistered | `auth.user.registered` | Email, Name, VerificationToken | Register, ResendVerification |
| EmailVerified | `auth.user.email_verified` | UserID, Email, Name | VerifyEmail |
| PasswordResetRequested | `auth.user.password_reset_requested` | Email, Name, ResetToken | ForgotPassword |

**File:** `internal/todo/domain/event/todo_events.go` (existing — defined but never published)

Wire existing events to actually publish from command handlers.

### 3. Email Event Handler

**File:** `internal/authentication/infrastructure/eventbus/email_handler.go`

```go
type EmailHandler struct {
    mailer domain.Emailer
    log    domain.Logger
}

func (h *EmailHandler) Register(bus events.EventBus) { ... }
```

- `onUserRegistered` → `mailer.SendVerification(to, name, token)`
- `onEmailVerified` → `mailer.SendWelcome(to, name)`
- `onPasswordResetRequested` → `mailer.SendPasswordReset(to, name, token)`

### 4. Authentication Service Changes

**File:** `internal/authentication/application/service/authentication_service.go`

- Remove `domain.Emailer mailer` field, add `events.EventBus bus`
- Constructor: `NewAuthenticationService(repos..., bus events.EventBus)` — removes `mailer`
- Methods changed: `Register`, `VerifyEmail`, `ForgotPassword`, `ResendVerification` all publish events instead of calling `mailer.Send*()`

### 5. Todo Event Publishing

**File:** `internal/todo/application/command/*.go` — 4 command handlers publish `todo.created`, `todo.updated`, `todo.completed`, `todo.deleted`

Each handler gets `events.EventBus` injected. No changes to existing `TodoEventHandler` (still logs).

### 6. Unified Response Envelope

**File:** `internal/shared/utils/utils.go` (modified)

```json
// Success (single resource)
{"success": true, "data": {...}, "meta": null}

// Success (paginated list)
{"success": true, "data": [...], "meta": {"page": 1, "per_page": 20, "total": 100, "total_pages": 5}}

// Error
{"success": false, "data": null, "error": {"code": "VALIDATION_ERROR", "message": "..."}}
```

Structs:

```go
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data"`
    Meta    *PaginationMeta `json:"meta"`
    Error   *ErrorBody  `json:"error,omitempty"`
}

type PaginationMeta struct {
    Page       int `json:"page"`
    PerPage    int `json:"per_page"`
    Total      int `json:"total"`
    TotalPages int `json:"total_pages"`
}

type ErrorBody struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

`Data` and `Meta` are always in the JSON output (data is null for errors, meta is null for non-paginated responses). `Error` omitted on success via `omitempty`.

New helper functions:
- `RespondSuccess(w, data)` — 200, data, meta=null
- `RespondCreated(w, data)` — 201, data, meta=null
- `RespondPaginated(w, data, page, perPage, total)` — 200, data, meta={page, per_page, total, total_pages}
- `RespondError(w, status, code, message)` — generic error
- `RespondBadRequest(w, message)` — 400, VALIDATION_ERROR
- `RespondUnauthorized(w, message)` — 401, UNAUTHORIZED
- `RespondForbidden(w, code, message)` — 403, with custom code (FORBIDDEN, ACCOUNT_LOCKED, EMAIL_NOT_VERIFIED)
- `RespondNotFound(w, message)` — 404, NOT_FOUND
- `RespondConflict(w, message)` — 409, CONFLICT
- `RespondInternalError(w, message)` — 500, INTERNAL_ERROR

### 7. Global Error Handling

**Error-to-HTTP mapper** (`internal/shared/utils/utils.go`):

```go
func MapError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, domain.ErrNotFound):
        RespondNotFound(w, err.Error())
    case errors.Is(err, domain.ErrAlreadyExists):
        RespondConflict(w, err.Error())
    case errors.Is(err, domain.ErrValidation):
        RespondBadRequest(w, err.Error())
    case errors.Is(err, domain.ErrForbidden):
        RespondForbidden(w, "FORBIDDEN", err.Error())
    case errors.Is(err, domain.ErrUnauthorized):
        RespondUnauthorized(w, err.Error())
    default:
        RespondInternalError(w, "internal server error")
    }
}
```

**Standardized error codes:**

| HTTP | Code | When |
|------|------|------|
| 400 | `VALIDATION_ERROR` | Invalid body/params/validation |
| 401 | `UNAUTHORIZED` | Missing/invalid/expired token |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 403 | `ACCOUNT_LOCKED` | Account temporarily locked |
| 403 | `EMAIL_NOT_VERIFIED` | Email not verified |
| 404 | `NOT_FOUND` | Resource not found |
| 409 | `CONFLICT` | Duplicate/state conflict |
| 429 | `RATE_LIMITED` | Rate limit exceeded |
| 500 | `INTERNAL_ERROR` | Unexpected error |

**Middleware cleanup:**

- `ErrorHandler` (panic recovery) uses `utils.RespondInternalError` instead of hardcoded JSON
- `Authentication` middleware uses `utils.RespondUnauthorized` instead of hardcoded JSON
- `AuthenticationWithDenylist` uses `utils.RespondUnauthorized`  
- `Authorization` middleware uses `utils.RespondUnauthorized` / `utils.RespondForbidden`
- Rate limit middleware uses `utils.RespondError(w, 429, "RATE_LIMITED", ...)`

**File:** `internal/shared/middleware/middleware.go`, `internal/shared/middleware/ratelimit.go`

### 8. Fx Wiring

- `EventBus` provided as shared singleton from `internal/shared/events/module.go`
- `NewInMemoryEventBus()` returns `events.EventBus` interface
- `EmailHandler` registered via `fx.Invoke` in `cmd/api/main.go`
- Todo command handlers get `events.EventBus` injected
- `AuthenticationService` no longer takes `domain.Emailer`

### 9. Pagination

Current `internal/shared/pagination/pagination.go` already has a `Pagination` struct. List handlers that return paginated responses use `RespondPaginated` instead of `RespondSuccess`. Meta is null for non-list responses.

Handlers that currently return paginated lists:
- `ListTodos` (todo handler)
- `SearchTodos` (todo handler)
- `ListRoles` (authorization handler)
- `ListPermissions` (authorization handler)
- `List` users (user handler)

### 10. Documentation Updates

- `docs/Architecture.md` — Add event-driven email + unified response + error handling sections
- `docs/FolderStructure.md` — Add new directories
- `docs/API.md` — Document response envelope, error codes, pagination format
- `docs/superpowers/specs/2026-07-09-email-service-design.md` — Reference this spec

## Error Handling Strategy

- Email sending: best-effort — publish errors logged, never crash handlers
- HTTP errors: `MapError` translates domain errors to consistent HTTP codes
- Panics: recovery middleware returns standardized `INTERNAL_ERROR`
- All middleware uses shared `utils.Respond*` functions (no hardcoded JSON)

## Global Constraints

- Do NOT modify database schemas or migrations
- Do NOT modify domain entities
- Do NOT add new external dependencies
- Module path: `github.com/IDTS-LAB/go-codebase`

## File Structure

| File | Status | Responsibility |
|------|--------|---------------|
| `internal/shared/events/events.go` | **Modify** | Extract EventBus interface, rename struct to InMemoryEventBus |
| `internal/shared/events/module.go` | **Create** | Fx module providing EventBus as singleton |
| `internal/shared/utils/utils.go` | **Modify** | Unified response envelope, pagination meta, error-to-HTTP mapper, new helper functions |
| `internal/shared/middleware/middleware.go` | **Modify** | Use utils.Respond* instead of hardcoded JSON |
| `internal/shared/middleware/ratelimit.go` | **Modify** | Use utils.RespondError instead of hardcoded JSON |
| `internal/authentication/domain/event/auth_events.go` | **Create** | UserRegistered, EmailVerified, PasswordResetRequested |
| `internal/authentication/infrastructure/eventbus/email_handler.go` | **Create** | Email event handler |
| `internal/authentication/application/service/authentication_service.go` | **Modify** | Remove mailer dep, add EventBus, publish events |
| `internal/authentication/application/service/authentication_service_test.go` | **Modify** | Remove mailer mock from tests |
| `internal/authentication/interfaces/http/handlers.go` | **Modify** | Use MapError + standardized codes |
| `internal/authentication/interfaces/http/handlers_test.go` | **Modify** | Update for new response structure |
| `internal/todo/application/command/create_todo_handler.go` | **Modify** | Publish todo.created |
| `internal/todo/application/command/update_todo_handler.go` | **Modify** | Publish todo.updated |
| `internal/todo/application/command/complete_todo_handler.go` | **Modify** | Publish todo.completed |
| `internal/todo/application/command/delete_todo_handler.go` | **Modify** | Publish todo.deleted |
| `internal/todo/interfaces/http/handlers.go` | **Modify** | Use MapError + standardized codes, RespondPaginated for lists |
| `internal/todo/interfaces/http/handlers_test.go` | **Modify** | Update for new response structure |
| `internal/todo/module.go` | **Modify** | Remove EventBus provide |
| `internal/authentication/module.go` | **Modify** | Remove mailer from auth service constructor params |
| `internal/authorization/interfaces/http/handlers.go` | **Modify** | Use MapError + standardized codes, RespondPaginated for lists |
| `internal/user/interfaces/http/handler.go` | **Modify** | Use MapError + standardized codes, RespondPaginated for lists |
| `cmd/api/main.go` | **Modify** | Email handler registration + shared EventBus provide |
| `docs/Architecture.md` | **Modify** | Add event-driven + response sections |
| `docs/FolderStructure.md` | **Modify** | Add new directories |
| `docs/API.md` | **Modify** | Document response envelope and error codes |

## Future (Async) Support

The `EventBus` interface is the extension point. A future `RabbitMQEventBus` implements the same `Publish`/`Subscribe` contract — no changes needed to services, events, or handlers.
