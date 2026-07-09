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
			r.Use(middleware.Authorization(authorizer, "todo", "create"))
			r.Post("/", handler.CreateTodo)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "todo", "list"))
			r.Get("/", handler.ListTodos)
			r.Get("/search", handler.SearchTodos)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "todo", "read"))
			r.Get("/{id}", handler.GetTodo)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "todo", "update"))
			r.Put("/{id}", handler.UpdateTodo)
			r.Patch("/{id}/complete", handler.CompleteTodo)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "todo", "delete"))
			r.Delete("/{id}", handler.DeleteTodo)
		})
	})

	return r
}
