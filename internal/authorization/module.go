package authorization

import (
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/service"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/casbin"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/authorization/interfaces/http"
	"go.uber.org/fx"
)

var Module = fx.Module("authorization",
	casbin.Module,

	fx.Provide(
		persistence.NewRoleRepository,
		persistence.NewPermissionRepository,
		persistence.NewUserRoleRepository,
		persistence.NewRolePermissionRepository,
		casbin.NewPolicyLoader,
		service.NewAuthorizationService,
		httpHandler.NewHandler,
		httpHandler.NewRouter,
	),
)
