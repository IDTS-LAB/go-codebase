# CQRS Standardization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add CommandBus/QueryBus infrastructure and standardize all 5 domain modules (auth, user, authorization, tenant, todo) to use CQRS with central dispatch.

**Architecture:** Shared `CommandBus` and `QueryBus` in `internal/shared/cqrs/` route commands/queries by type to registered handlers. Each module's HTTP handlers dispatch to the bus instead of calling services directly. Event publishing moves into command handlers.

**Tech Stack:** Go, fx, sqlc, in-memory bus with reflect-based routing

## Global Constraints

- All bus routing uses `reflect.TypeOf(cmd/query).String()` as map key
- Command/Query handlers follow the exact same pattern: one file per operation, `Handle(ctx, struct) (response, error)`
- No new event types or EventBus changes
- Existing repository interfaces and domain entities remain unchanged
- Tests must pass after each task

---

### Task 1: Create Shared CQRS Bus Layer

**Files:**
- Create: `internal/shared/cqrs/bus.go`
- Create: `internal/shared/cqrs/bus_test.go`
- Create: `internal/shared/cqrs/module.go`

**Interfaces:**
- Produces: `CommandBus`, `QueryBus`, `CommandHandler`, `QueryHandler` interfaces
- Produces: `InMemoryCommandBus`, `InMemoryQueryBus` implementations
- Produces: Fx module providing both buses as singletons

- [ ] **Step 1: Create `internal/shared/cqrs/bus.go`**

```go
package cqrs

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type CommandHandler interface {
	Handle(ctx context.Context, cmd any) (any, error)
}

type CommandBus interface {
	Dispatch(ctx context.Context, cmd any) (any, error)
	Register(cmd any, handler CommandHandler)
}

type QueryHandler interface {
	Handle(ctx context.Context, query any) (any, error)
}

type QueryBus interface {
	Ask(ctx context.Context, query any) (any, error)
	Register(query any, handler QueryHandler)
}

type inMemoryCommandBus struct {
	mu       sync.RWMutex
	handlers map[string]CommandHandler
}

func NewInMemoryCommandBus() *inMemoryCommandBus {
	return &inMemoryCommandBus{handlers: make(map[string]CommandHandler)}
}

func (b *inMemoryCommandBus) Register(cmd any, handler CommandHandler) {
	key := reflect.TypeOf(cmd).String()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[key] = handler
}

func (b *inMemoryCommandBus) Dispatch(ctx context.Context, cmd any) (any, error) {
	key := reflect.TypeOf(cmd).String()
	b.mu.RLock()
	handler, ok := b.handlers[key]
	b.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no handler registered for command: %s", key)
	}
	return handler.Handle(ctx, cmd)
}

type inMemoryQueryBus struct {
	mu       sync.RWMutex
	handlers map[string]QueryHandler
}

func NewInMemoryQueryBus() *inMemoryQueryBus {
	return &inMemoryQueryBus{handlers: make(map[string]QueryHandler)}
}

func (b *inMemoryQueryBus) Register(query any, handler QueryHandler) {
	key := reflect.TypeOf(query).String()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[key] = handler
}

func (b *inMemoryQueryBus) Ask(ctx context.Context, query any) (any, error) {
	key := reflect.TypeOf(query).String()
	b.mu.RLock()
	handler, ok := b.handlers[key]
	b.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no handler registered for query: %s", key)
	}
	return handler.Handle(ctx, query)
}
```

- [ ] **Step 2: Create `internal/shared/cqrs/bus_test.go`**

```go
package cqrs

import (
	"context"
	"errors"
	"testing"
)

type testCommand struct {
	Value string
}

type testCommandHandler struct{}

func (h *testCommandHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(testCommand)
	if c.Value == "error" {
		return nil, errors.New("test error")
	}
	return "handled:" + c.Value, nil
}

type testQuery struct {
	ID string
}

type testQueryHandler struct{}

func (h *testQueryHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(testQuery)
	return "result:" + q.ID, nil
}

func TestCommandBus_Dispatch(t *testing.T) {
	bus := NewInMemoryCommandBus()
	bus.Register(testCommand{}, &testCommandHandler{})

	resp, err := bus.Dispatch(context.Background(), testCommand{Value: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.(string) != "handled:hello" {
		t.Fatalf("expected 'handled:hello', got '%s'", resp)
	}
}

func TestCommandBus_Unregistered(t *testing.T) {
	bus := NewInMemoryCommandBus()
	_, err := bus.Dispatch(context.Background(), testCommand{Value: "x"})
	if err == nil {
		t.Fatal("expected error for unregistered command")
	}
}

func TestCommandBus_HandlerError(t *testing.T) {
	bus := NewInMemoryCommandBus()
	bus.Register(testCommand{}, &testCommandHandler{})

	_, err := bus.Dispatch(context.Background(), testCommand{Value: "error"})
	if err == nil || err.Error() != "test error" {
		t.Fatalf("expected 'test error', got '%v'", err)
	}
}

func TestQueryBus_Ask(t *testing.T) {
	bus := NewInMemoryQueryBus()
	bus.Register(testQuery{}, &testQueryHandler{})

	resp, err := bus.Ask(context.Background(), testQuery{ID: "123"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.(string) != "result:123" {
		t.Fatalf("expected 'result:123', got '%s'", resp)
	}
}

func TestQueryBus_Unregistered(t *testing.T) {
	bus := NewInMemoryQueryBus()
	_, err := bus.Ask(context.Background(), testQuery{ID: "x"})
	if err == nil {
		t.Fatal("expected error for unregistered query")
	}
}
```

- [ ] **Step 3: Create `internal/shared/cqrs/module.go`**

```go
package cqrs

import "go.uber.org/fx"

var Module = fx.Module("cqrs",
	fx.Provide(
		NewInMemoryCommandBus,
		NewInMemoryQueryBus,
	),
)
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/shared/cqrs/... -v`
Expected: all 5 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/shared/cqrs/ && git commit -m "feat(cqrs): add CommandBus and QueryBus with in-memory implementations"
```

---

### Task 2: Migrate Authorization Module to CQRS

**Files:**
- Create: `internal/authorization/application/command/create_role.go`
- Create: `internal/authorization/application/command/update_role.go`
- Create: `internal/authorization/application/command/delete_role.go`
- Create: `internal/authorization/application/command/create_permission.go`
- Create: `internal/authorization/application/command/update_permission.go`
- Create: `internal/authorization/application/command/delete_permission.go`
- Create: `internal/authorization/application/command/assign_role.go`
- Create: `internal/authorization/application/command/unassign_role.go`
- Create: `internal/authorization/application/command/assign_permission.go`
- Create: `internal/authorization/application/command/unassign_permission.go`
- Create: `internal/authorization/application/query/get_role.go`
- Create: `internal/authorization/application/query/list_roles.go`
- Create: `internal/authorization/application/query/get_permission.go`
- Create: `internal/authorization/application/query/list_permissions.go`
- Create: `internal/authorization/application/query/get_user_roles.go`
- Create: `internal/authorization/application/query/get_role_permissions.go`
- Create: `internal/authorization/application/query/check_permission.go`
- Modify: `internal/authorization/application/service/authorization_service.go` — delete file entirely (replaced by handlers)
- Modify: `internal/authorization/module.go` — register handlers with bus, remove service provider
- Modify: `internal/authorization/interfaces/http/handlers.go` — use bus instead of service
- Modify: `internal/authorization/interfaces/http/handler_test.go` — update constructor call

**Interfaces:**
- Consumes: `CommandBus`, `QueryBus` from cqrs package
- Consumes: `repository.RoleRepository`, `repository.PermissionRepository`, `repository.UserRoleRepository`, `repository.RolePermissionRepository`, `casbin.Enforcer`
- Produces: command/query handler types registered with bus

Each command/query file follows this pattern:

```go
// create_role.go
package command

import (
	"context"
	"errors"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	coredomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type CreateRoleCommand struct {
	Name        string
	Description string
}

type CreateRoleHandler struct {
	roleRepo repository.RoleRepository
	enforcer Enforcer
}

type Enforcer interface {
	ReloadPolicies(ctx context.Context) error
}

func NewCreateRoleHandler(roleRepo repository.RoleRepository, enforcer Enforcer) *CreateRoleHandler {
	return &CreateRoleHandler{roleRepo: roleRepo, enforcer: enforcer}
}

func (h *CreateRoleHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(CreateRoleCommand)
	existing, _ := h.roleRepo.GetByName(ctx, c.Name)
	if existing != nil {
		return nil, coredomain.ErrConflict
	}
	role := entity.NewRole(c.Name, c.Description)
	if err := h.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}
```

```go
// get_role.go
package query

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/google/uuid"
)

type GetRoleQuery struct {
	ID uuid.UUID
}

type GetRoleHandler struct {
	roleRepo repository.RoleRepository
}

func NewGetRoleHandler(roleRepo repository.RoleRepository) *GetRoleHandler {
	return &GetRoleHandler{roleRepo: roleRepo}
}

func (h *GetRoleHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(GetRoleQuery)
	return h.roleRepo.GetByID(ctx, q.ID)
}
```

- [ ] **Step 1: Create all 17 command/query files**

Each file follows the pattern above with:
- Unique command/query struct
- Handler struct with repository/enforcer dependencies
- Constructor function
- `Handle(ctx, any) (any, error)` method

Create commands:
1. `create_role.go` — creates role, checks existing name, returns `*entity.Role`
2. `update_role.go` — gets role by ID, updates fields, calls repo.Update, returns `*entity.Role`
3. `delete_role.go` — calls repo.Delete
4. `create_permission.go` — same pattern as create_role but for Permission
5. `update_permission.go` — same pattern as update_role but for Permission
6. `delete_permission.go` — calls repo.Delete
7. `assign_role.go` — calls repo.GetByID for role validation, then userRoleRepo.Assign, then enforcer.ReloadUserPolicies
8. `unassign_role.go` — calls userRoleRepo.Remove, then enforcer.ReloadUserPolicies
9. `assign_permission.go` — calls roleRepo.GetByID + permRepo.GetByID for validation, then rolePermRepo.Assign, then enforcer.ReloadPolicies
10. `unassign_permission.go` — calls rolePermRepo.Remove, then enforcer.ReloadPolicies

Create queries:
11. `get_role.go` — calls roleRepo.GetByID, returns `*entity.Role`
12. `list_roles.go` — calls roleRepo.GetAll, returns `([]*entity.Role, int)`
13. `get_permission.go` — calls permRepo.GetByID, returns `*entity.Permission`
14. `list_permissions.go` — calls permRepo.GetAll, returns `([]*entity.Permission, int)`
15. `get_user_roles.go` — calls userRoleRepo.GetRolesByUserID, returns `[]*entity.Role`
16. `get_role_permissions.go` — calls rolePermRepo.GetPermissionsByRoleID, returns `[]*entity.Permission`
17. `check_permission.go` — calls enforcer.Enforce, returns `bool`

- [ ] **Step 2: Update `internal/authorization/interfaces/http/handlers.go`**

Add `cqrs.CommandBus` and `cqrs.QueryBus` to constructor and all handler methods.

Current pattern:
```go
type Handler struct {
	svc *service.AuthorizationService
}
```

New pattern:
```go
type Handler struct {
	commandBus cqrs.CommandBus
	queryBus   cqrs.QueryBus
}
```

Each handler method changes from `h.svc.CreateRole(...)` to dispatching a command/query:
```go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    // ... parse request ...
    resp, err := h.commandBus.Dispatch(r.Context(), command.CreateRoleCommand{
        Name:        req.Name,
        Description: req.Description,
    })
    // ... handle response ...
}
```

For methods that return entity types, cast to the expected type:
```go
role := resp.(*entity.Role)
// map to DTO response...
```

Remove dependency on `service.AuthorizationService`. Commands import from `command` package, queries from `query` package.

- [ ] **Step 3: Update `internal/authorization/module.go`**

Remove `fx.Provide` for `service.NewAuthorizationService`. Add `fx.Invoke` to register handlers with the bus:

```go
package authorization

import (
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/casbin"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"go.uber.org/fx"
)

var Module = fx.Module("authorization",
	fx.Provide(
		persistence.NewRoleRepository,
		persistence.NewPermissionRepository,
		persistence.NewUserRoleRepository,
		persistence.NewRolePermissionRepository,
	),
	fx.Provide(NewHandler),
	fx.Invoke(registerHandlers),
)

func registerHandlers(
	commandBus cqrs.CommandBus,
	queryBus cqrs.QueryBus,
	roleRepo persistence.RoleRepository,
	permRepo persistence.PermissionRepository,
	userRoleRepo persistence.UserRoleRepository,
	rolePermRepo persistence.RolePermissionRepository,
	enforcer *casbin.Enforcer,
) {
	// Commands
	commandBus.Register(command.CreateRoleCommand{}, command.NewCreateRoleHandler(roleRepo, enforcer))
	commandBus.Register(command.UpdateRoleCommand{}, command.NewUpdateRoleHandler(roleRepo))
	commandBus.Register(command.DeleteRoleCommand{}, command.NewDeleteRoleHandler(roleRepo))
	commandBus.Register(command.CreatePermissionCommand{}, command.NewCreatePermissionHandler(permRepo))
	commandBus.Register(command.UpdatePermissionCommand{}, command.NewUpdatePermissionHandler(permRepo))
	commandBus.Register(command.DeletePermissionCommand{}, command.NewDeletePermissionHandler(permRepo))
	commandBus.Register(command.AssignRoleCommand{}, command.NewAssignRoleHandler(roleRepo, userRoleRepo, enforcer))
	commandBus.Register(command.UnassignRoleCommand{}, command.NewUnassignRoleHandler(userRoleRepo, enforcer))
	commandBus.Register(command.AssignPermissionCommand{}, command.NewAssignPermissionHandler(roleRepo, permRepo, rolePermRepo, enforcer))
	commandBus.Register(command.UnassignPermissionCommand{}, command.NewUnassignPermissionHandler(rolePermRepo, enforcer))

	// Queries
	queryBus.Register(query.GetRoleQuery{}, query.NewGetRoleHandler(roleRepo))
	queryBus.Register(query.ListRolesQuery{}, query.NewListRolesHandler(roleRepo))
	queryBus.Register(query.GetPermissionQuery{}, query.NewGetPermissionHandler(permRepo))
	queryBus.Register(query.ListPermissionsQuery{}, query.NewListPermissionsHandler(permRepo))
	queryBus.Register(query.GetUserRolesQuery{}, query.NewGetUserRolesHandler(userRoleRepo))
	queryBus.Register(query.GetRolePermissionsQuery{}, query.NewGetRolePermissionsHandler(rolePermRepo))
	queryBus.Register(query.CheckPermissionQuery{}, query.NewCheckPermissionHandler(enforcer))
}
```

Note: `persistence.RoleRepository` is the concrete type from persistence package, but `registerHandlers` needs the interface type `repository.RoleRepository`. Since Go uses implicit interface satisfaction, use the concrete type in fx and cast in the function:

```go
type RoleRepository interface {
    Create(ctx context.Context, role *entity.Role) error
    // ... etc
}
```

Actually, since the handler constructors accept interface types, we need to use the interface. But since Fx resolves by type, we need to either provide the interface or use the concrete type and cast.

The cleanest approach: define aliases for the repository interfaces at the module level, or provide the interfaces directly. Let's keep it simple — the handler constructors accept the repository interfaces directly, and Fx will resolve them since the concrete types satisfy the interfaces:

```go
func registerHandlers(
	commandBus cqrs.CommandBus,
	queryBus cqrs.QueryBus,
	roleRepo repository.RoleRepository,
	permRepo repository.PermissionRepository,
	userRoleRepo repository.UserRoleRepository,
	rolePermRepo repository.RolePermissionRepository,
	enforcer Enforcer,
) {
```

- [ ] **Step 4: Update the Handler struct to inject commandBus/queryBus**

Read the current `internal/authorization/interfaces/http/handlers.go` and update the constructor and all methods.

The Handler currently:
```go
type Handler struct {
	svc *service.AuthorizationService
}

func NewHandler(svc *service.AuthorizationService, v *validator.Validator) *Handler {
```

Update to:
```go
type Handler struct {
	commandBus cqrs.CommandBus
	queryBus   cqrs.QueryBus
	v          *validator.Validator
}

func NewHandler(commandBus cqrs.CommandBus, queryBus cqrs.QueryBus, v *validator.Validator) *Handler {
```

Update module.go's `fx.Provide(NewHandler)` — Fx auto-resolves the new dependencies.

- [ ] **Step 5: Update `internal/authorization/interfaces/http/handler_test.go`**

The test creates a Handler with `handlers.NewHandler(svc, validator)`. Update to:
```go
cmdBus := cqrs.NewInMemoryCommandBus()
qBus := cqrs.NewInMemoryQueryBus()
h := handlers.NewHandler(cmdBus, qBus, validator)

// Register mock handlers for tests
cmdBus.Register(command.CreateRoleCommand{}, ...)
```

Since the tests need to register handlers that return mock data, create a simple test handler that returns predefined values:

```go
type mockCreateRoleHandler struct {
	result *entity.Role
	err    error
}

func (h *mockCreateRoleHandler) Handle(ctx context.Context, cmd any) (any, error) {
	return h.result, h.err
}
```

Register these in test setup.

- [ ] **Step 6: Run tests**

Run: `go test ./internal/authorization/... -v -count=1`
Expected: all tests PASS (previously 57 tests)

- [ ] **Step 7: Commit**

```bash
git add internal/authorization/ && git commit -m "refactor(authorization): migrate to CQRS with CommandBus/QueryBus"
```

---

### Task 3: Migrate User Module to CQRS

**Files:**
- Create: `internal/user/application/command/update_user.go`
- Create: `internal/user/application/command/delete_user.go`
- Create: `internal/user/application/query/list_users.go`
- Create: `internal/user/application/query/get_user.go`
- Delete: `internal/user/application/service/user_service.go`
- Delete: `internal/user/application/service/service_test.go`
- Modify: `internal/user/interfaces/http/handler.go` — use bus instead of service
- Modify: `internal/user/interfaces/http/handler_test.go` — update constructor
- Modify: `internal/user/module.go` — register handlers with bus

- [ ] **Step 1: Create 4 command/query files**

Commands:
1. `update_user.go` — `UpdateUserCommand{ID uuid.UUID, Name string, Email string, IsActive bool}` — handler updates user via repository
2. `delete_user.go` — `DeleteUserCommand{ID uuid.UUID}` — handler soft-deletes user

Queries:
3. `list_users.go` — `ListUsersQuery{Offset, Limit int}` — returns `([]*entity.User, int)`
4. `get_user.go` — `GetUserQuery{ID uuid.UUID}` — returns `*entity.User`

Each handler depends on `repository.UserRepository` (from `internal/user/domain/repository`).

- [ ] **Step 2: Update `internal/user/interfaces/http/handler.go`**

Replace `*service.UserService` with `cqrs.CommandBus` + `cqrs.QueryBus`. Update all handler methods.

- [ ] **Step 3: Update `internal/user/module.go`**

Remove `fx.Provide(service.NewUserService)`. Add `fx.Invoke(registerHandlers)`.

Handler registration:
```go
commandBus.Register(command.UpdateUserCommand{}, command.NewUpdateUserHandler(userRepo))
commandBus.Register(command.DeleteUserCommand{}, command.NewDeleteUserHandler(userRepo))
queryBus.Register(query.ListUsersQuery{}, query.NewListUsersHandler(userRepo))
queryBus.Register(query.GetUserQuery{}, query.NewGetUserHandler(userRepo))
```

- [ ] **Step 4: Update `internal/user/interfaces/http/handler_test.go`**

Replace `service.NewUserService` with bus-based constructor using mock handlers.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/user/... -v -count=1`
Expected: all 26 tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/user/ && git commit -m "refactor(user): migrate to CQRS with CommandBus/QueryBus"
```

---

### Task 4: Migrate Tenant Module to CQRS

**Files:**
- Create: `internal/tenant/application/command/create_tenant.go`
- Create: `internal/tenant/application/command/update_tenant.go`
- Create: `internal/tenant/application/command/delete_tenant.go`
- Create: `internal/tenant/application/query/get_tenant.go`
- Create: `internal/tenant/application/query/list_tenants.go`
- Delete: `internal/tenant/application/service/tenant.go`
- Modify: `internal/tenant/interfaces/http/handlers.go` — use bus instead of service
- Modify: `internal/tenant/module.go` — register handlers with bus

- [ ] **Step 1: Create 5 command/query files**

Commands:
1. `create_tenant.go` — `CreateTenantCommand{dto.CreateTenantRequest}` — creates tenant via repository, returns `dto.TenantResponse`
2. `update_tenant.go` — `UpdateTenantCommand{ID, Name, Domain, Settings, IsActive}` — updates via repository
3. `delete_tenant.go` — `DeleteTenantCommand{ID}` — deletes via repository

Queries:
4. `get_tenant.go` — `GetTenantQuery{ID}` — returns `dto.TenantResponse`
5. `list_tenants.go` — `ListTenantsQuery{Page, PerPage}` — returns `dto.TenantListResponse`

- [ ] **Step 2: Update `internal/tenant/interfaces/http/handlers.go`**

Replace `*appService.TenantService` with `cqrs.CommandBus` + `cqrs.QueryBus`. Update all handler methods.

- [ ] **Step 3: Update `internal/tenant/module.go`**

Remove service provider. Add handler registration via `fx.Invoke`.

- [ ] **Step 4: Run tests**

Run: `go build ./internal/tenant/...`
Expected: builds clean

- [ ] **Step 5: Commit**

```bash
git add internal/tenant/ && git commit -m "refactor(tenant): migrate to CQRS with CommandBus/QueryBus"
```

---

### Task 5: Migrate Authentication Module to CQRS

**Files:**
- Create: `internal/authentication/application/command/register_user.go`
- Create: `internal/authentication/application/command/generate_tokens.go`
- Create: `internal/authentication/application/command/refresh_token.go`
- Create: `internal/authentication/application/command/logout.go`
- Create: `internal/authentication/application/command/logout_all.go`
- Create: `internal/authentication/application/command/verify_email.go`
- Create: `internal/authentication/application/command/forgot_password.go`
- Create: `internal/authentication/application/command/reset_password.go`
- Create: `internal/authentication/application/command/resend_verification.go`
- Create: `internal/authentication/application/query/login.go`
- Modify: `internal/authentication/module.go` — register handlers, remove service
- Modify: `internal/authentication/interfaces/http/handlers.go` — use bus instead of service
- Modify: `internal/authentication/application/service/authentication_service.go` — delete (replaced by handlers)
- Modify: `internal/authentication/application/service/authentication_service_test.go` — delete (move tests or rewrite)

- [ ] **Step 1: Create 10 command/query files**

Commands (9):
1. `register_user.go` — hashes password, creates user, generates verify token, publishes `UserRegisteredEvent`
2. `generate_tokens.go` — generates JWT + refresh token pair, returns `service.TokenPair`
3. `refresh_token.go` — validates refresh token, generates new pair
4. `logout.go` — revokes refresh token + optionally denylists JWT
5. `logout_all.go` — revokes all refresh tokens for user
6. `verify_email.go` — validates verification token, marks email verified, publishes `EmailVerifiedEvent`
7. `forgot_password.go` — generates reset token, publishes `PasswordResetRequestedEvent`
8. `reset_password.go` — validates reset token, updates password, revokes all sessions
9. `resend_verification.go` — generates new verify token, publishes `UserRegisteredEvent`

Queries (1):
10. `login.go` — authenticates user, checks lockout/verification, returns `*entity.User`

Handler dependencies:
- `repository.UserRepository`
- `repository.RefreshTokenRepository`  
- `domain.TokenService`
- `events.EventBus`
- Optional: `denylist func(ctx, jti string, ttl time.Duration) error`

- [ ] **Step 2: Update `internal/authentication/interfaces/http/handlers.go`**

Replace `*service.AuthenticationService` with `cqrs.CommandBus` + `cqrs.QueryBus`.

Handlers like `Register`:
```go
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
    var req dto.RegisterRequest
    // ...
    resp, err := h.commandBus.Dispatch(r.Context(), command.RegisterUserCommand{
        Email:    req.Email,
        Password: req.Password,
        Name:     req.Name,
    })
    // ... handle response ...
}
```

- [ ] **Step 3: Update module.go**

Register all 10 handlers with the bus via `fx.Invoke`.

- [ ] **Step 4: Update tests**

The existing `authentication_service_test.go` tests the service methods directly. Rewrite these as command/query handler tests. Use the same mocked repositories approach.

For the HTTP handler tests (`authentication/interfaces/http/handlers_test.go`), update to use bus with mock handlers.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/authentication/... -v -count=1`
Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/authentication/ && git commit -m "refactor(authentication): migrate to CQRS with CommandBus/QueryBus"
```

---

### Task 6: Migrate Todo Module to Use Bus

**Files:**
- Delete: `internal/todo/application/service/todo_app_service.go`
- Modify: `internal/todo/interfaces/http/handlers.go` — use bus instead of service facade
- Modify: `internal/todo/module.go` — register handlers with bus, remove service provider
- Modify: `internal/todo/interfaces/http/handlers_test.go` — update constructor

- [ ] **Step 1: Update `internal/todo/interfaces/http/handlers.go`**

Replace `*appService.TodoAppService` with `cqrs.CommandBus` + `cqrs.QueryBus`. Wire each method to dispatch the corresponding command/query:

```go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    var req dto.CreateTodoRequest
    // ... parse ...
    resp, err := h.commandBus.Dispatch(r.Context(), command.CreateTodoCommand{
        Title:       req.Title,
        Description: req.Description,
    })
    // ... handle ...
}
```

- [ ] **Step 2: Update `internal/todo/module.go`**

Remove `fx.Provide(appService.NewTodoAppService)`. Add `fx.Invoke` to register command/query handlers:

```go
func registerHandlers(
    commandBus cqrs.CommandBus,
    queryBus cqrs.QueryBus,
    createHandler *command.CreateTodoHandler,
    updateHandler *command.UpdateTodoHandler,
    completeHandler *command.CompleteTodoHandler,
    deleteHandler *command.DeleteTodoHandler,
    getHandler *query.GetTodoHandler,
    listHandler *query.ListTodosHandler,
    searchHandler *query.SearchTodosHandler,
) {
    commandBus.Register(command.CreateTodoCommand{}, createHandler)
    commandBus.Register(command.UpdateTodoCommand{}, updateHandler)
    commandBus.Register(command.CompleteTodoCommand{}, completeHandler)
    commandBus.Register(command.DeleteTodoCommand{}, deleteHandler)
    queryBus.Register(query.GetTodoQuery{}, getHandler)
    queryBus.Register(query.ListTodosQuery{}, listHandler)
    queryBus.Register(query.SearchTodosQuery{}, searchHandler)
}
```

- [ ] **Step 3: Update test**

Update handler tests to use bus with registered mock handlers.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/todo/... -v -count=1`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/todo/ && git commit -m "refactor(todo): migrate AppService facade to direct bus dispatch"
```

---

### Task 7: Update Main Wiring

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Add cqrs module to fx.New**

```go
app := fx.New(
    // ... existing ...
    cqrs.Module,  // Add before domain modules
    // ... domain modules ...
)
```

- [ ] **Step 2: Update handler constructors**

The handler constructors changed — they now accept `CommandBus` and `QueryBus` instead of service instances. Fx auto-resolves these, so the `fx.Populate` and router wiring stay the same if the constructors are updated.

The router `NewRouter` calls need their handler constructors updated to match new signatures. Since Fx handles injection, the main.go `fx.Populate` should still work as long as the new constructors are correctly set up.

- [ ] **Step 3: Run tests**

Run: `go build ./... && go test ./... -count=1`
Expected: all builds and tests PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/api/main.go && git commit -m "feat: wire CQRS CommandBus/QueryBus into application"
```

---

### Task 8: Remove Dead Code

- [ ] **Step 1: Remove deleted service files from git**

Some service files were deleted in previous tasks but may still be in the working tree. Clean up:

```bash
git rm internal/user/application/service/user_service.go internal/user/application/service/service_test.go 2>/dev/null || true
git rm internal/authorization/application/service/authorization_service.go 2>/dev/null || true
git rm internal/tenant/application/service/tenant.go 2>/dev/null || true
git rm internal/authentication/application/service/authentication_service.go 2>/dev/null || true
git rm internal/todo/application/service/todo_app_service.go 2>/dev/null || true
```

- [ ] **Step 2: Final test pass**

Run: `go build ./... && go test ./... -count=1`
Expected: clean build, all tests pass

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "chore: remove deprecated service facades after CQRS migration"
```
