package http

import (
	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Post("/", handler.CreateTodo)
	r.Get("/", handler.ListTodos)
	r.Get("/search", handler.SearchTodos)
	r.Get("/{id}", handler.GetTodo)
	r.Put("/{id}", handler.UpdateTodo)
	r.Delete("/{id}", handler.DeleteTodo)
	r.Patch("/{id}/complete", handler.CompleteTodo)

	return r
}
