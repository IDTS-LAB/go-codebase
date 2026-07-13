# CQRS Standardization Across All Domain Modules

## Purpose

Standardize every domain module (authentication, user, authorization, tenant) to use the CQRS pattern already established in the todo module, but upgraded with a central CommandBus and QueryBus for dispatch.

## Architecture

```
HTTP Handler → bus.Dispatch(cmd) / bus.Ask(query)
                   ↓
          InMemoryCommandBus / InMemoryQueryBus
                   ↓
            (type-keyed handler lookup)
                   ↓
          CommandHandler / QueryHandler
                   ↓
          Domain Service / Repository
                   ↓
          EventBus.Publish (commands only)
```

### Bus Layer (`internal/shared/cqrs/`)

Two interfaces + in-memory implementations:

**CommandBus**
```go
type CommandHandler interface {
    Handle(ctx context.Context, cmd any) (any, error)
}

type CommandBus interface {
    Dispatch(ctx context.Context, cmd any) (any, error)
    Register(cmd any, handler CommandHandler)
}
```

**QueryBus**
```go
type QueryHandler interface {
    Handle(ctx context.Context, query any) (any, error)
}

type QueryBus interface {
    Ask(ctx context.Context, query any) (any, error)
    Register(query any, handler QueryHandler)
}
```

**Routing:** `reflect.TypeOf(cmd/query).String()` as map key, computed at registration, looked up on dispatch. No per-field reflection at runtime.

**Cross-cutting:** Bus wraps dispatch with telemetry span + panic recovery + error wrapping.

**Fx Module:** Provides `CommandBus` and `QueryBus` as singletons. Registration happens via `fx.Invoke` in each module.

### Per-Module Structure

Each module gains `application/command/` and `application/query/` packages with one file per operation. Each file exports:
- A **command/query struct** (the input)
- A **handler struct** implementing `CommandHandler`/`QueryHandler`
- A `Handle(ctx, struct) (response, error)` method

The old `application/service/` file is either replaced by the handlers (auth, tenant, user) or kept as-is if it provides non-CQRS functionality.

### Module Migrations

#### Authentication Module

Service methods → Commands/Queries:

| Method | Becomes | Type | Events |
|--------|---------|------|--------|
| Register | RegisterUserCommand | Command | UserRegistered |
| Login | LoginQuery | Query | — |
| GenerateTokens | GenerateTokensCommand | Command | — |
| RefreshToken | RefreshTokenCommand | Command | — |
| Logout | LogoutCommand | Command | — |
| LogoutAll | LogoutAllCommand | Command | — |
| VerifyEmail | VerifyEmailCommand | Command | EmailVerified |
| ForgotPassword | ForgotPasswordCommand | Command | PasswordResetRequested |
| ResetPassword | ResetPasswordCommand | Command | — |
| ResendVerification | ResendVerificationCommand | Command | UserRegistered (reuses event) |

#### User Module

| Method | Becomes | Type |
|--------|---------|------|
| List | ListUsersQuery | Query |
| GetByID | GetUserQuery | Query |
| Update | UpdateUserCommand | Command |
| Delete | DeleteUserCommand | Command |

#### Authorization Module

| Method | Becomes | Type |
|--------|---------|------|
| CreateRole | CreateRoleCommand | Command |
| GetRole | GetRoleQuery | Query |
| ListRoles | ListRolesQuery | Query |
| UpdateRole | UpdateRoleCommand | Command |
| DeleteRole | DeleteRoleCommand | Command |
| CreatePermission | CreatePermissionCommand | Command |
| GetPermission | GetPermissionQuery | Query |
| ListPermissions | ListPermissionsQuery | Query |
| UpdatePermission | UpdatePermissionCommand | Command |
| DeletePermission | DeletePermissionCommand | Command |
| AssignRoleToUser | AssignRoleCommand | Command |
| RemoveRoleFromUser | UnassignRoleCommand | Command |
| GetUserRoles | GetUserRolesQuery | Query |
| AssignPermissionToRole | AssignPermissionCommand | Command |
| RemovePermissionFromRole | UnassignPermissionCommand | Command |
| GetRolePermissions | GetRolePermissionsQuery | Query |
| CheckPermission | CheckPermissionQuery | Query |

#### Tenant Module

| Method | Becomes | Type |
|--------|---------|------|
| Create | CreateTenantCommand | Command |
| GetByID | GetTenantQuery | Query |
| List | ListTenantsQuery | Query |
| Update | UpdateTenantCommand | Command |
| Delete | DeleteTenantCommand | Command |

### HTTP Handler Changes

Every handler method changes from:
```go
// before
resp, err := h.svc.SomeMethod(ctx, args)
```

to:
```go
// after
resp, err := h.bus.Dispatch(ctx, command.CreateSomeCommand{...})
// or
resp, err := h.bus.Ask(ctx, query.SomeQuery{...})
```

Each handler gets `CommandBus` and `QueryBus` injected via constructor (Fx provides them).

### Existing Todo Module Alignment

The todo module's existing `application/service/todo_app_service.go` (facade) is replaced by direct bus dispatch in handlers. The existing command/query handlers remain, but are registered with the bus in the module's `fx.Invoke` instead of being called through the facade.

### Event Publishing

Events stay where they are — published in command handlers (todo already does this, auth will adopt it). No changes to `EventBus` or event types. User, authorization, tenant commands don't publish events initially (can be added per command later).

### Testing Strategy

- Bus unit tests: dispatch to registered handler returns expected response, unregistered command returns error
- Each command/query handler: unit test with mocked repository (same as current service tests, just moved)
- HTTP handlers: integration test with real bus + mocked repositories
- No changes needed to existing repository mocks

### Files Changed

| Area | Files |
|------|-------|
| **New: cqrs package** | `internal/shared/cqrs/bus.go`, `internal/shared/cqrs/module.go` |
| **Auth module** | Replace `application/service/` with 10 command/query files, update handlers, update module.go |
| **User module** | Replace `application/service/` with 4 command/query files, update handlers, update module.go |
| **Authorization module** | Replace `application/service/` with 17 command/query files, update handlers, update module.go |
| **Tenant module** | Replace `application/service/` with 5 command/query files, update handlers, update module.go |
| **Todo module** | Remove `application/service/todo_app_service.go`, update handlers to use bus, update module.go |
| **cmd/api/main.go** | Wire cqrs module, update handler constructors |
