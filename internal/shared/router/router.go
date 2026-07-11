package router

import (
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/go-chi/chi/v5"
)

const APIPrefix = "/api/v1"

type Handlers struct {
	Auth  *chi.Mux
	Todo  *chi.Mux
	Authz *chi.Mux
	User  *chi.Mux
}

func NewRouter(h Handlers, mw middleware.Registry, log domain.Logger, cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()

	r.Use(mw.Tracing)
	r.Use(middleware.RequestID)
	r.Use(mw.ErrorHandler)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS(&cfg.CORS))
	r.Use(mw.ErrorRecorder)
	r.Use(mw.AuditLog)
	r.Use(mw.RateLimit)
	r.Use(middleware.Logger(log))

	registerWeb(r, cfg)

	r.Route(APIPrefix, func(r chi.Router) {
		r.Use(middleware.ResponseFormatter())

		r.Group(func(r chi.Router) {
			r.Use(mw.MaxBodySize)
			r.Use(mw.Idempotency)
			r.Mount("/auth", h.Auth)
		})

		r.Group(func(r chi.Router) {
			r.Use(mw.Auth)
			r.Use(mw.MaxBodySize)
			r.Mount("/todos", h.Todo)
			r.Mount("/users", h.User)
			r.Mount("/auth/sessions", h.Authz)
		})
	})

	return r
}
