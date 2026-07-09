package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler, authMiddleware func(http.Handler) http.Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/me", handler.Me)
		r.Get("/", handler.List)
		r.Get("/{id}", handler.Get)
		r.Put("/{id}", handler.Update)
		r.Delete("/{id}", handler.Delete)
	})

	return r
}
