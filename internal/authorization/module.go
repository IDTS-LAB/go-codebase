package authorization

import (
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/casbin"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/public"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/authorization/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"go.uber.org/fx"
)

var Module = fx.Module("authorization",
	casbin.Module,

	fx.Provide(
		persistence.NewRoleRepository,
		persistence.NewPermissionRepository,
		persistence.NewUserRoleRepository,
		persistence.NewRolePermissionRepository,
		casbin.NewAdapter,
		httpHandler.NewHandler,
		fx.Annotate(application.NewAuthorizationProvider, fx.As(new(public.AuthorizationProvider))),
	),

	fx.Invoke(registerHandlers),
)

func registerHandlers(
	commandBus cqrs.CommandBus,
	queryBus cqrs.QueryBus,
	roleRepo repository.RoleRepository,
	permRepo repository.PermissionRepository,
	userRoleRepo repository.UserRoleRepository,
	rolePermRepo repository.RolePermissionRepository,
	enforcer *casbin.Enforcer,
) {
	// Commands
	commandBus.Register(command.CreateRoleCommand{}, command.NewCreateRoleHandler(roleRepo))
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
