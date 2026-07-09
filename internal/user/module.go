package user

import (
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/user/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/service"
	"github.com/IDTS-LAB/go-codebase/internal/user/infrastructure/persistence"
	"go.uber.org/fx"
)

var Module = fx.Module("user",
	fx.Provide(
		persistence.NewUserRepository,
		service.NewUserService,
		httpHandler.NewHandler,
	),
)
