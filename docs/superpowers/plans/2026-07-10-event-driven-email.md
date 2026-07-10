# Event-Driven Email, Unified Response, Global Error Handling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decouple email from business logic via domain events, standardize all HTTP responses into a single envelope with pagination meta, and centralize error handling with consistent codes.

**Architecture:** Services publish domain events to an `EventBus` interface (in-memory sync now, swappable for RabbitMQ/Kafka later). An `EmailHandler` subscribes and calls the mailer. A unified `APIResponse` envelope carries `data`, `meta` (for paginated lists), and `error` fields. Middleware uses shared response helpers instead of hardcoded JSON. `MapError` translates domain errors to HTTP codes.

**Tech Stack:** Go 1.25, Fx DI, Chi router, PostgreSQL, `database/sql`, `lib/pq`

## Global Constraints

- Module path: `github.com/IDTS-LAB/go-codebase`
- Do NOT modify database schemas or migrations
- Do NOT modify domain entities
- Do NOT add new external dependencies
- Existing `Event` struct and `Handler` type in `internal/shared/events/` stay unchanged

---

## File Structure

| File | Status | Resp. |
|------|--------|-------|
| `internal/shared/events/events.go` | Modify | Extract EventBus interface, rename struct to InMemoryEventBus |
| `internal/shared/events/module.go` | Create | Fx module providing EventBus |
| `internal/shared/utils/utils.go` | Modify | APIResponse, PaginationMeta, MapError, new helpers |
| `internal/shared/middleware/middleware.go` | Modify | Use utils.Respond* instead of hardcoded JSON |
| `internal/shared/middleware/ratelimit.go` | Modify | Use utils.RespondError |
| `internal/authentication/domain/event/auth_events.go` | Create | UserRegistered, EmailVerified, PasswordResetRequested |
| `internal/authentication/infrastructure/eventbus/email_handler.go` | Create | Email event handler |
| `internal/authentication/application/service/authentication_service.go` | Modify | Remove mailer, publish events |
| `internal/authentication/application/service/authentication_service_test.go` | Modify | Remove mailer mock |
| `internal/authentication/interfaces/http/handlers.go` | Modify | MapError + standardized codes |
| `internal/authentication/interfaces/http/handlers_test.go` | Modify | New response structure |
| `internal/todo/application/command/create_todo_handler.go` | Modify | Publish todo.created |
| `internal/todo/application/command/update_todo_handler.go` | Modify | Publish todo.updated |
| `internal/todo/application/command/complete_todo_handler.go` | Modify | Publish todo.completed |
| `internal/todo/application/command/delete_todo_handler.go` | Modify | Publish todo.deleted |
| `internal/todo/interfaces/http/handlers.go` | Modify | MapError + RespondPaginated |
| `internal/todo/interfaces/http/handlers_test.go` | Modify | New response structure |
| `internal/todo/module.go` | Modify | Remove EventBus provide |
| `internal/authentication/module.go` | Modify | Remove mailer param from auth service |
| `internal/authorization/interfaces/http/handlers.go` | Modify | MapError + RespondPaginated |
| `internal/user/interfaces/http/handler.go` | Modify | MapError + RespondPaginated |
| `cmd/api/main.go` | Modify | Add email handler + shared EventBus |
| `docs/Architecture.md` | Modify | Add event-driven + response sections |
| `docs/FolderStructure.md` | Modify | Add new dirs |
| `docs/API.md` | Modify | Document envelope + error codes |

---

### Task 1: EventBus Interface + Shared Module

**Files:**
- Modify: `internal/shared/events/events.go`
- Create: `internal/shared/events/module.go`

- [ ] **Step 1: Read existing events.go**

```bash
cat internal/shared/events/events.go
```

- [ ] **Step 2: Rewrite `internal/shared/events/events.go`**

Extract `EventBus` interface, add doc comments, rename current struct to `InMemoryEventBus`:

```go
package events

import (
	"context"
	"sync"
)

type Event struct {
	Type    string
	Payload interface{}
}

type Handler func(ctx context.Context, event Event) error

type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventType string, handler Handler)
}

type InMemoryEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]Handler),
	}
}

func (eb *InMemoryEventBus) Subscribe(eventType string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *InMemoryEventBus) Publish(ctx context.Context, event Event) error {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 3: Create `internal/shared/events/module.go`**

```go
package events

import "go.uber.org/fx"

var Module = fx.Module("events",
	fx.Provide(
		fx.Annotate(NewInMemoryEventBus, fx.As(new(EventBus))),
	),
)
```

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/shared/events/
git commit -m "feat(events): extract EventBus interface, rename InMemoryEventBus"
```

---

### Task 2: Auth Domain Events + Email Handler

**Files:**
- Create: `internal/authentication/domain/event/auth_events.go`
- Create: `internal/authentication/infrastructure/eventbus/email_handler.go`

- [ ] **Step 1: Create `internal/authentication/domain/event/auth_events.go`**

```go
package event

type UserRegistered struct {
	Email            string
	Name             string
	VerificationToken string
}

type EmailVerified struct {
	UserID string
	Email  string
	Name   string
}

type PasswordResetRequested struct {
	Email     string
	Name      string
	ResetToken string
}

const (
	UserRegisteredEvent        = "auth.user.registered"
	EmailVerifiedEvent         = "auth.user.email_verified"
	PasswordResetRequestedEvent = "auth.user.password_reset_requested"
)
```

- [ ] **Step 2: Create `internal/authentication/infrastructure/eventbus/email_handler.go`**

```go
package eventbus

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
)

type EmailHandler struct {
	mailer domain.Emailer
	log    domain.Logger
}

func NewEmailHandler(mailer domain.Emailer, log domain.Logger) *EmailHandler {
	return &EmailHandler{mailer: mailer, log: log}
}

func (h *EmailHandler) Register(bus events.EventBus) {
	bus.Subscribe(event.UserRegisteredEvent, h.onUserRegistered)
	bus.Subscribe(event.EmailVerifiedEvent, h.onEmailVerified)
	bus.Subscribe(event.PasswordResetRequestedEvent, h.onPasswordResetRequested)
}

func (h *EmailHandler) onUserRegistered(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(event.UserRegistered)
	if !ok {
		return nil
	}
	if err := h.mailer.SendVerification(payload.Email, payload.Name, payload.VerificationToken); err != nil {
		h.log.Error(ctx, "failed to send verification email", domain.Error(err))
	}
	return nil
}

func (h *EmailHandler) onEmailVerified(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(event.EmailVerified)
	if !ok {
		return nil
	}
	if err := h.mailer.SendWelcome(payload.Email, payload.Name); err != nil {
		h.log.Error(ctx, "failed to send welcome email", domain.Error(err))
	}
	return nil
}

func (h *EmailHandler) onPasswordResetRequested(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(event.PasswordResetRequested)
	if !ok {
		return nil
	}
	if err := h.mailer.SendPasswordReset(payload.Email, payload.Name, payload.ResetToken); err != nil {
		h.log.Error(ctx, "failed to send password reset email", domain.Error(err))
	}
	return nil
}
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/authentication/domain/event/ internal/authentication/infrastructure/eventbus/
git commit -m "feat(auth): add domain events and email event handler"
```

---

### Task 3: Auth Service — Publish Events, Remove Mailer

**Files:**
- Modify: `internal/authentication/application/service/authentication_service.go`
- Modify: `internal/authentication/application/service/authentication_service_test.go`

- [ ] **Step 1: Read current `authentication_service.go`**

```bash
cat internal/authentication/application/service/authentication_service.go
```

- [ ] **Step 2: Modify constructor**

Replace `mailer domain.Emailer` param with `bus events.EventBus`. Add imports for `events` and `authentication/domain/event`.

```go
type AuthenticationService struct {
	userRepo         repository.UserRepository
	tokenRepo        repository.RefreshTokenRepository
	passwordSvc      password.Service
	tokenSvc         token.Service
	bus              events.EventBus
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
	maxFailedAttempts  int
	lockoutDuration    time.Duration
}

func NewAuthenticationService(
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	passwordSvc password.Service,
	tokenSvc token.Service,
	bus events.EventBus,
	cfg *config.Config,
) *AuthenticationService {
	return &AuthenticationService{
		userRepo:           userRepo,
		tokenRepo:          tokenRepo,
		passwordSvc:        passwordSvc,
		tokenSvc:           tokenSvc,
		bus:                bus,
		accessTokenExpiry:  cfg.JWT.AccessTokenExpiry,
		refreshTokenExpiry: cfg.JWT.RefreshTokenExpiry,
		maxFailedAttempts:  cfg.Auth.MaxFailedAttempts,
		lockoutDuration:    cfg.Auth.LockoutDuration,
	}
}
```

- [ ] **Step 3: Modify `Register` method — publish `UserRegistered` instead of calling mailer**

In the `Register` method, replace `s.mailer.SendVerification(...)` with:

```go
_ = s.bus.Publish(ctx, events.Event{
	Type: event.UserRegisteredEvent,
	Payload: event.UserRegistered{
		Email:             user.Email,
		Name:              user.Name,
		VerificationToken: token,
	},
})
```

Remove the `_ = s.mailer.SendVerification(...)` line entirely. Remove `mailer.SendWelcome` from `VerifyEmail`, `mailer.SendPasswordReset` from `ForgotPassword`, and `mailer.SendVerification` from `ResendVerification`.

- [ ] **Step 4: Modify `VerifyEmail` — publish `EmailVerified`**

```go
_ = s.bus.Publish(ctx, events.Event{
	Type: event.EmailVerifiedEvent,
	Payload: event.EmailVerified{
		UserID: user.ID.String(),
		Email:  user.Email,
		Name:   user.Name,
	},
})
```

- [ ] **Step 5: Modify `ForgotPassword` — publish `PasswordResetRequested`**

```go
_ = s.bus.Publish(ctx, events.Event{
	Type: event.PasswordResetRequestedEvent,
	Payload: event.PasswordResetRequested{
		Email:     user.Email,
		Name:      user.Name,
		ResetToken: token,
	},
})
```

- [ ] **Step 6: Modify `ResendVerification` — publish `UserRegistered` (same as Register)**

```go
return s.bus.Publish(ctx, events.Event{
	Type: event.UserRegisteredEvent,
	Payload: event.UserRegistered{
		Email:             user.Email,
		Name:              user.Name,
		VerificationToken: token,
	},
})
```

- [ ] **Step 7: Remove unused imports** (`domain.Emailer` not used anymore) and remove the `mailer` field reference.

- [ ] **Step 8: Update `authentication_service_test.go`**

Read the current test file. The tests create `AuthenticationService` with a mailer mock. Replace the mailer mock with a nil bus or a simple in-memory bus. Remove the mailer assertions (we test email sending separately via the handler).

Replace test setup:

```go
bus := events.NewInMemoryEventBus()
svc := service.NewAuthenticationService(userRepo, tokenRepo, passwordSvc, tokenSvc, bus, cfg)
```

Remove any `mockMailer` references and `mailer.AssertExpectations` calls.

- [ ] **Step 9: Verify build + tests**

```bash
go build ./...
go test ./internal/authentication/... 2>&1
```

Expected: Build and tests pass.

- [ ] **Step 10: Commit**

```bash
git add internal/authentication/application/service/
git commit -m "feat(auth): replace direct mailer calls with event publishing"
```

---

### Task 4: Todo Event Publishing

**Files:**
- Modify: `internal/todo/application/command/create_todo_handler.go`
- Modify: `internal/todo/application/command/update_todo_handler.go`
- Modify: `internal/todo/application/command/complete_todo_handler.go`
- Modify: `internal/todo/application/command/delete_todo_handler.go`

- [ ] **Step 1: Modify `create_todo_handler.go`**

Add `events.EventBus` to the constructor. In the `Handle` method, publish `todo.created` after successful creation:

```go
type CreateTodoHandler struct {
	todoRepo repository.TodoRepository
	eventBus events.EventBus
}

func NewCreateTodoHandler(todoRepo repository.TodoRepository, eventBus events.EventBus) *CreateTodoHandler {
	return &CreateTodoHandler{todoRepo: todoRepo, eventBus: eventBus}
}

func (h *CreateTodoHandler) Handle(ctx context.Context, cmd command.CreateTodoCommand) (*entity.Todo, error) {
	// ... existing validation and creation code ...

	if err := h.todoRepo.Create(ctx, todo); err != nil {
		return nil, err
	}

	_ = h.eventBus.Publish(ctx, events.Event{
		Type: event.TodoCreatedEvent,
		Payload: event.TodoCreated{
			ID:        todo.ID,
			Title:     todo.Title,
			CreatedAt: todo.CreatedAt,
		},
	})

	return todo, nil
}
```

- [ ] **Step 2: Modify `update_todo_handler.go`**

Similar pattern — inject `events.EventBus`, publish `todo.updated` after update.

- [ ] **Step 3: Modify `complete_todo_handler.go`**

Inject `events.EventBus`, publish `todo.completed` after completion.

- [ ] **Step 4: Modify `delete_todo_handler.go`**

Inject `events.EventBus`, publish `todo.deleted` after deletion.

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: Clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/todo/application/command/
git commit -m "feat(todo): wire event publishing in command handlers"
```

---

### Task 5: Unified Response Envelope + Global Error Middleware

**Files:**
- Modify: `internal/shared/utils/utils.go`
- Modify: `internal/shared/middleware/middleware.go`
- Modify: `internal/shared/middleware/ratelimit.go`

- [ ] **Step 1: Rewrite `internal/shared/utils/utils.go`**

```go
package utils

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
)

type APIResponse struct {
	Success bool            `json:"success"`
	Data    interface{}     `json:"data"`
	Meta    *PaginationMeta `json:"meta"`
	Error   *ErrorBody      `json:"error,omitempty"`
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

func RespondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func RespondSuccess(w http.ResponseWriter, data interface{}) {
	RespondJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

func RespondCreated(w http.ResponseWriter, data interface{}) {
	RespondJSON(w, http.StatusCreated, APIResponse{Success: true, Data: data})
}

func RespondPaginated(w http.ResponseWriter, data interface{}, page, perPage, total int) {
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 0 {
		totalPages = 0
	}
	RespondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta: &PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func RespondError(w http.ResponseWriter, status int, code, message string) {
	RespondJSON(w, status, APIResponse{
		Success: false,
		Error: &ErrorBody{Code: code, Message: message},
	})
}

func RespondBadRequest(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", message)
}

func RespondUnauthorized(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func RespondForbidden(w http.ResponseWriter, code, message string) {
	RespondError(w, http.StatusForbidden, code, message)
}

func RespondNotFound(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusNotFound, "NOT_FOUND", message)
}

func RespondConflict(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusConflict, "CONFLICT", message)
}

func RespondInternalError(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

func MapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		RespondNotFound(w, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists) || errors.Is(err, domain.ErrConflict):
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

- [ ] **Step 2: Update `internal/shared/middleware/middleware.go`**

Replace all hardcoded JSON strings with `utils.Respond*` calls.

Replace the `ErrorHandler` panic recovery (line ~43):

```go
func ErrorHandler(log domain.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error(r.Context(), "panic recovered", domain.String("panic", fmt.Sprintf("%v", rec)))
					utils.RespondInternalError(w, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
```

Replace all `http.Error(w, hardcodedJSON, status)` in the `Authentication` and `Authorization` middleware with:

```go
utils.RespondUnauthorized(w, "missing token")
// or
utils.RespondUnauthorized(w, "invalid token")
// or
utils.RespondForbidden(w, "FORBIDDEN", "insufficient permissions")
```

The specific replacements (exact error messages may vary slightly):

| Middleware | Old hardcoded JSON | New call |
|---|---|---|
| Authentication (missing token) | `{"success":false,"error":{"code":"UNAUTHORIZED","message":"missing token"}}` | `utils.RespondUnauthorized(w, "missing token")` |
| Authentication (invalid token) | `{"success":false,"error":{"code":"UNAUTHORIZED","message":"invalid token"}}` | `utils.RespondUnauthorized(w, "invalid token")` |
| AuthWithDenylist (missing token) | same as above | `utils.RespondUnauthorized(w, "missing token")` |
| AuthWithDenylist (invalid token) | same as above | `utils.RespondUnauthorized(w, "invalid token")` |
| AuthWithDenylist (revoked) | `{"success":false,"error":{"code":"UNAUTHORIZED","message":"token has been revoked"}}` | `utils.RespondUnauthorized(w, "token has been revoked")` |
| Authorization (not authenticated) | `{"success":false,"error":{"code":"UNAUTHORIZED","message":"user not authenticated"}}` | `utils.RespondUnauthorized(w, "user not authenticated")` |
| Authorization (invalid user ID) | `{"success":false,"error":{"code":"UNAUTHORIZED","message":"invalid user ID"}}` | `utils.RespondUnauthorized(w, "invalid user ID")` |
| Authorization (check failed) | `{"success":false,"error":{"code":"INTERNAL_ERROR","message":"authorization check failed"}}` | `utils.RespondInternalError(w, "authorization check failed")` |
| Authorization (insufficient perms) | `{"success":false,"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}` | `utils.RespondForbidden(w, "FORBIDDEN", "insufficient permissions")` |

Add `utils` import: `"github.com/IDTS-LAB/go-codebase/internal/shared/utils"`.

- [ ] **Step 3: Update `internal/shared/middleware/ratelimit.go`**

Replace the hardcoded JSON:

```go
utils.RespondError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
```

Add the `utils` import.

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/shared/utils/ internal/shared/middleware/
git commit -m "feat(http): unified response envelope, MapError, middleware cleanup"
```

---

### Task 6: Handler Updates — All 4 Domains

**Files:**
- Modify: `internal/authentication/interfaces/http/handlers.go`
- Modify: `internal/authentication/interfaces/http/handlers_test.go`
- Modify: `internal/todo/interfaces/http/handlers.go`
- Modify: `internal/todo/interfaces/http/handlers_test.go`
- Modify: `internal/authorization/interfaces/http/handlers.go`
- Modify: `internal/user/interfaces/http/handler.go`

- [ ] **Step 1: Update `authentication/interfaces/http/handlers.go`**

Replace the existing per-handler `switch err` patterns with `utils.MapError` for common errors. Keep domain-specific error handling (like `ACCOUNT_LOCKED`, `EMAIL_NOT_VERIFIED`) as explicit checks before falling through to `MapError`.

Pattern:

```go
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// ... decode, validate ...
	user, err := h.authService.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			utils.RespondConflict(w, err.Error())
			return
		}
		utils.RespondInternalError(w, "internal server error")
		return
	}
	utils.RespondCreated(w, toUserResponse(user))
}
```

Replace all handlers similarly. Key changes:
- `Login`: check `ErrInvalidCredentials` → `RespondUnauthorized`, `ErrAccountDisabled` → `RespondUnauthorized`, `ErrAccountLocked` → `RespondForbidden("ACCOUNT_LOCKED", ...)`, `ErrEmailNotVerified` → `RespondForbidden("EMAIL_NOT_VERIFIED", ...)`, else `RespondInternalError`
- `RefreshToken`: `ErrInvalidRefreshToken` → `RespondUnauthorized`, else `RespondInternalError`
- `VerifyEmail`: `ErrInvalidVerifyToken` / `ErrVerifyTokenExpired` → `RespondBadRequest`, else `RespondInternalError`
- `ForgotPassword`: always `RespondSuccess` (vague on purpose)
- `ResetPassword`: `ErrInvalidResetToken` / `ErrResetTokenExpired` → `RespondBadRequest`, else `RespondInternalError`
- `ResendVerification`: always `RespondSuccess` (vague on purpose)

- [ ] **Step 2: Update `authentication/interfaces/http/handlers_test.go`**

Read the current test file. The tests assert on response JSON bodies using struct fields. Update all test assertions to match the new `APIResponse` envelope structure. Key changes:
- Old: `{"success": true, "data": {...}}`
- New: `{"success": true, "data": {...}, "meta": null}`
- Old: `{"success": false, "error": {"code": "...", "message": "..."}}`
- New: `{"success": false, "data": null, "error": {"code": "...", "message": "..."}}`

- [ ] **Step 3: Update `todo/interfaces/http/handlers.go`**

Replace `switch err` with `MapError` pattern. For `ListTodos` and `SearchTodos`, use `RespondPaginated`:

```go
utils.RespondPaginated(w, resp.Todos, req.Page, req.PerPage, resp.Total)
```

- [ ] **Step 4: Update `todo/interfaces/http/handlers_test.go`**

Update response JSON assertions to match new envelope. Paginated responses now include `meta`.

- [ ] **Step 5: Update `authorization/interfaces/http/handlers.go`**

Replace blanket error handling with proper sentinel errors + `MapError`. For `CreateRole`, `CreatePermission` — use `MapError`. For `ListRoles`, `ListPermissions` — use `RespondPaginated`.

- [ ] **Step 6: Update `user/interfaces/http/handler.go`**

Replace error handling with `MapError`. Use `RespondPaginated` for `List`.

- [ ] **Step 7: Verify build + tests**

```bash
go build ./...
go test ./... 2>&1
```

Expected: All tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/authentication/interfaces/http/ internal/todo/interfaces/http/ internal/authorization/interfaces/http/ internal/user/interfaces/http/
git commit -m "feat(handlers): use MapError, standardized codes, RespondPaginated for lists"
```

---

### Task 7: Fx Wiring

**Files:**
- Modify: `internal/todo/module.go`
- Modify: `internal/authentication/module.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Update `internal/todo/module.go`**

Remove `events.NewEventBus` and `eventbus.NewTodoEventHandler` provides (they move to shared module or main). Remove the `fx.Invoke` that registers the todo event handler (it's still needed but depends on the shared EventBus). Actually keep the TodoEventHandler but remove the EventBus provide.

```go
package todo

import (
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/todo/infrastructure/eventbus"
	"go.uber.org/fx"
)

var Module = fx.Module("todo",
	fx.Provide(
		// ... persistence, services, handlers ...
		eventbus.NewTodoEventHandler,
	),
	fx.Invoke(
		func(bus events.EventBus, eh *eventbus.TodoEventHandler) {
			eh.Register(bus)
		},
	),
)
```

Remove `events.NewEventBus` from the `fx.Provide` list. The EventBus is now provided by the shared events module.

- [ ] **Step 2: Update `internal/authentication/module.go`**

Remove `domain.Emailer` from the auth service constructor params. The auth service no longer depends on the mailer.

Read the current module file and change the `fx.Provide` line from:

```go
fx.Annotate(service.NewAuthenticationService, fx.ParamTags(...)),
```

to match the new constructor signature (replacing mailer param with EventBus). If there were no `fx.ParamTags`, the change is simply that the constructor signature changed and Fx resolves EventBus from the shared events module.

- [ ] **Step 3: Update `cmd/api/main.go`**

Add the shared events module and email handler registration.

```go
import (
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	authEventBus "github.com/IDTS-LAB/go-codebase/internal/authentication/infrastructure/eventbus"
)

app := fx.New(
	fx.Supply(cfg),

	// Shared infrastructure
	events.Module,  // <-- provides EventBus interface
	logger.Module,
	// ... other modules ...

	// Applications modules
	authentication.Module,
	authorization.Module,
	todo.Module,
	user.Module,

	// Email handler registration
	fx.Invoke(func(bus events.EventBus, eh *authEventBus.EmailHandler) {
		eh.Register(bus)
	}),

	// ... rest of main.go ...
)
```

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

Expected: Clean build.

- [ ] **Step 5: Run tests**

```bash
go test ./... 2>&1
```

Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/todo/module.go internal/authentication/module.go cmd/api/main.go
git commit -m "chore: update Fx wiring for shared EventBus and email handler"
```

---

### Task 8: Documentation

**Files:**
- Modify: `docs/Architecture.md`
- Modify: `docs/FolderStructure.md`
- Modify: `docs/API.md`

- [ ] **Step 1: Update `docs/Architecture.md`**

Add a section for event-driven email:

```markdown
### Event-Driven Email

Services publish domain events to an `EventBus` interface (in-memory synchronous by default, swappable for RabbitMQ/Kafka). An `EmailHandler` subscribes to auth domain events (`auth.user.registered`, `auth.user.email_verified`, `auth.user.password_reset_requested`) and calls the appropriate mailer method.

Flow: `Service → EventBus.Publish() → EmailHandler → domain.Emailer.Send*()`
```

Add a section for unified response envelope:

```markdown
### API Response Format

All HTTP responses use a unified envelope:

```json
// Success (single)
{"success": true, "data": {...}, "meta": null}

// Success (paginated list)
{"success": true, "data": [...], "meta": {"page": 1, "per_page": 20, "total": 100, "total_pages": 5}}

// Error
{"success": false, "data": null, "error": {"code": "VALIDATION_ERROR", "message": "..."}}
```

Errors are mapped to HTTP status codes via `utils.MapError`, which translates `domain.ErrNotFound`, `domain.ErrConflict`, etc.
```
```

- [ ] **Step 2: Update `docs/FolderStructure.md`**

Add entries:

```
internal/authentication/domain/event/       # Domain events (UserRegistered, EmailVerified, PasswordResetRequested)
internal/authentication/infrastructure/eventbus/  # Event handler implementations (EmailHandler)
internal/shared/events/                     # EventBus interface + InMemoryEventBus + Fx module
```

- [ ] **Step 3: Update `docs/API.md`**

Add a section at the top:

```markdown
## Response Format

All API responses follow this structure:

| Field | Type | Description |
|-------|------|-------------|
| `success` | bool | Always present. `true` for success, `false` for error. |
| `data` | any | Response payload. `null` on errors. |
| `meta` | object or null | Pagination metadata. `null` for single-resource responses. |
| `error` | object or omitted | Error details. Present only on errors. |

### Pagination

Paginated list endpoints return `meta`:

| Field | Type | Description |
|-------|------|-------------|
| `page` | int | Current page number |
| `per_page` | int | Items per page |
| `total` | int | Total items across all pages |
| `total_pages` | int | Total number of pages |

### Error Codes

| HTTP | Code | Description |
|------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid request body or parameters |
| 401 | `UNAUTHORIZED` | Missing, invalid, or expired token |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 403 | `ACCOUNT_LOCKED` | Account temporarily locked |
| 403 | `EMAIL_NOT_VERIFIED` | Email not verified |
| 404 | `NOT_FOUND` | Resource not found |
| 409 | `CONFLICT` | Duplicate or state conflict |
| 429 | `RATE_LIMITED` | Rate limit exceeded |
| 500 | `INTERNAL_ERROR` | Unexpected server error |
```

- [ ] **Step 4: Commit**

```bash
git add docs/
git commit -m "docs: update architecture, folder structure, and API docs for event-driven email, unified response, error handling"
```

---

## Verification Summary

| Check | Expected |
|-------|----------|
| `go build ./...` | No compilation errors |
| `go test ./...` | All tests pass |
| `go vet ./...` | No vet issues |
| HTTP 200 | `{"success": true, "data": {...}, "meta": null}` |
| HTTP 201 | `{"success": true, "data": {...}, "meta": null}` |
| HTTP 400 | `{"success": false, "data": null, "error": {"code": "VALIDATION_ERROR", "message": "..."}}` |
| HTTP 401 | `{"success": false, "data": null, "error": {"code": "UNAUTHORIZED", "message": "..."}}` |
| HTTP 403 | `{"success": false, "data": null, "error": {"code": "FORBIDDEN", "message": "..."}}` |
| HTTP 404 | `{"success": false, "data": null, "error": {"code": "NOT_FOUND", "message": "..."}}` |
| HTTP 429 | `{"success": false, "data": null, "error": {"code": "RATE_LIMITED", "message": "..."}}` |
| Paginated list | `meta` includes `page`, `per_page`, `total`, `total_pages` |
| Email on register | `auth.user.registered` event → `EmailHandler` → `SendVerification()` |
| Email on verify | `auth.user.email_verified` event → `EmailHandler` → `SendWelcome()` |
| Email on forgot password | `auth.user.password_reset_requested` → `EmailHandler` → `SendPasswordReset()` |
| Todo events published | `todo.created`, `todo.updated`, `todo.completed`, `todo.deleted` emitted on actions |
