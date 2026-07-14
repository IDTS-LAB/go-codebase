package authentication

import (
	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/query"
	authEvent "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/infrastructure/eventbus"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/infrastructure/persistence"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/authentication/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"go.uber.org/fx"
)

var Module = fx.Module("authentication",
	fx.Provide(
		persistence.NewUserRepository,
		persistence.NewRefreshTokenRepository,
		eventbus.NewEmailHandler,
		httpHandler.NewHandler,
	),

	fx.Invoke(
		registerHandlers,
		func() {
			events.Register(authEvent.UserRegisteredEvent, func() interface{} { return &authEvent.UserRegistered{} })
			events.Register(authEvent.EmailVerifiedEvent, func() interface{} { return &authEvent.EmailVerified{} })
			events.Register(authEvent.PasswordResetRequestedEvent, func() interface{} { return &authEvent.PasswordResetRequested{} })
		},
	),
)

func registerHandlers(
	commandBus cqrs.CommandBus,
	queryBus cqrs.QueryBus,
	userRepo repository.UserRepository,
	refreshRepo repository.RefreshTokenRepository,
	tokenService domain.TokenService,
	bus events.EventBus,
) {
	generateTokensHandler := command.NewGenerateTokensHandler(refreshRepo, tokenService)

	commandBus.Register(command.RegisterUserCommand{}, command.NewRegisterUserHandler(userRepo, bus))
	commandBus.Register(command.GenerateTokensCommand{}, generateTokensHandler)
	commandBus.Register(command.RefreshTokenCommand{}, command.NewRefreshTokenHandler(refreshRepo, userRepo, generateTokensHandler))
	commandBus.Register(command.LogoutCommand{}, command.NewLogoutHandler(refreshRepo))
	commandBus.Register(command.LogoutAllCommand{}, command.NewLogoutAllHandler(refreshRepo))
	commandBus.Register(command.VerifyEmailCommand{}, command.NewVerifyEmailHandler(userRepo, bus))
	commandBus.Register(command.ForgotPasswordCommand{}, command.NewForgotPasswordHandler(userRepo, bus))
	commandBus.Register(command.ResetPasswordCommand{}, command.NewResetPasswordHandler(userRepo, refreshRepo))
	commandBus.Register(command.ResendVerificationCommand{}, command.NewResendVerificationHandler(userRepo, bus))

	queryBus.Register(query.LoginQuery{}, query.NewLoginHandler(userRepo))
}
