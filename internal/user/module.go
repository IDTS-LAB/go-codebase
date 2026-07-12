package user

import (
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/user/infrastructure/persistence"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/user/interfaces/http"
	"go.uber.org/fx"
)

var Module = fx.Module("user",
	fx.Provide(
		persistence.NewUserRepository,
		httpHandler.NewHandler,
	),

	fx.Invoke(registerHandlers),
)

func registerHandlers(
	commandBus cqrs.CommandBus,
	queryBus cqrs.QueryBus,
	repo repository.UserRepository,
) {
	commandBus.Register(command.UpdateUserCommand{}, command.NewUpdateUserHandler(repo))
	commandBus.Register(command.DeleteUserCommand{}, command.NewDeleteUserHandler(repo))

	queryBus.Register(query.ListUsersQuery{}, query.NewListUsersHandler(repo))
	queryBus.Register(query.GetUserQuery{}, query.NewGetUserHandler(repo))
}
