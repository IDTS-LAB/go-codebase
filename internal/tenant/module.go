package tenant

import (
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/tenant/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/infrastructure/persistence"
	"go.uber.org/fx"
)

var Module = fx.Module("tenant",
	fx.Provide(
		persistence.NewTenantRepository,
		func(commandBus cqrs.CommandBus, queryBus cqrs.QueryBus, v *validator.Validator) *httpHandler.Handler {
			return httpHandler.NewHandler(commandBus, queryBus, v)
		},
	),

	fx.Invoke(registerHandlers),
)

func registerHandlers(
	commandBus cqrs.CommandBus,
	queryBus cqrs.QueryBus,
	repo repository.TenantRepository,
) {
	commandBus.Register(command.CreateTenantCommand{}, command.NewCreateTenantHandler(repo))
	commandBus.Register(command.UpdateTenantCommand{}, command.NewUpdateTenantHandler(repo))
	commandBus.Register(command.DeleteTenantCommand{}, command.NewDeleteTenantHandler(repo))

	queryBus.Register(query.GetTenantQuery{}, query.NewGetTenantHandler(repo))
	queryBus.Register(query.ListTenantsQuery{}, query.NewListTenantsHandler(repo))
}
