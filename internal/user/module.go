package user

import (
	roleProvider "github.com/IDTS-LAB/go-codebase/internal/authorization/public"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/user/application"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/user/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/user/infrastructure/persistence"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/user/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/user/public"
	"go.uber.org/fx"
)

var Module = fx.Module("user",
	fx.Provide(
		persistence.NewUserRepository,
		httpHandler.NewHandler,
		fx.Annotate(application.NewUserProfileProvider, fx.As(new(public.UserProfileProvider))),
	),

	fx.Invoke(registerHandlers),
)

func registerHandlers(
	commandBus cqrs.CommandBus,
	queryBus cqrs.QueryBus,
	repo repository.UserRepository,
	roleProvider roleProvider.AuthorizationProvider,
) {
	commandBus.Register(command.CreateUserCommand{}, command.NewCreateUserHandler(repo))
	commandBus.Register(command.UpdateUserCommand{}, command.NewUpdateUserHandler(repo))
	commandBus.Register(command.DeleteUserCommand{}, command.NewDeleteUserHandler(repo))

	queryBus.Register(query.ListUsersQuery{}, query.NewListUsersHandler(repo))
	queryBus.Register(query.GetUserQuery{}, query.NewGetUserHandler(repo, roleProvider))
}
