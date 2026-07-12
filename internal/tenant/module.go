package tenant

import (
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/service"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/tenant/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/infrastructure/persistence"
	"go.uber.org/fx"
)

var Module = fx.Module("tenant",
	fx.Provide(
		persistence.NewTenantRepository,
		service.NewTenantService,
		func(svc *service.TenantService, v *validator.Validator) *httpHandler.Handler {
			return httpHandler.NewHandler(svc, v)
		},
	),
)
