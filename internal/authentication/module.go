package authentication

import (
	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/service"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/infrastructure/persistence"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/authentication/interfaces/http"
	"go.uber.org/fx"
)

var Module = fx.Module("authentication",
	fx.Provide(
		persistence.NewUserRepository,
		persistence.NewRefreshTokenRepository,
		service.NewAuthenticationService,
		httpHandler.NewHandler,
	),
)
