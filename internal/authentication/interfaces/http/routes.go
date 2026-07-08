package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler, authMiddleware func(http.Handler) http.Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Post("/register", handler.Register)
	r.Post("/login", handler.Login)
	r.Post("/refresh", handler.RefreshToken)
	r.Post("/logout", handler.Logout)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/logout-all", handler.LogoutAll)
		r.Get("/me", handler.Me)
	})

	return r
}
