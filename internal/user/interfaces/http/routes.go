package http

import (
	"net/http"

	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler, authMiddleware func(http.Handler) http.Handler, authorizer middleware.Authorizer) *chi.Mux {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "user", "read"))
			r.Get("/me", handler.Me)
			r.Get("/", handler.List)
			r.Get("/{id}", handler.Get)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "user", "update"))
			r.Put("/{id}", handler.Update)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "user", "delete"))
			r.Delete("/{id}", handler.Delete)
		})
	})

	return r
}
