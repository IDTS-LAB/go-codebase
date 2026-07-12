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
			r.Use(middleware.Authorization(authorizer, "tenant", "create"))
			r.Post("/", handler.Create)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "tenant", "list"))
			r.Get("/", handler.List)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "tenant", "read"))
			r.Get("/{id}", handler.GetByID)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "tenant", "update"))
			r.Put("/{id}", handler.Update)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "tenant", "delete"))
			r.Delete("/{id}", handler.Delete)
		})
	})

	return r
}
