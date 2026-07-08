package http

import (
	"net/http"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler, authMiddleware func(http.Handler) http.Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(middleware.Timeout(30 * time.Second))

			r.Route("/roles", func(r chi.Router) {
				r.Post("/", handler.CreateRole)
				r.Get("/", handler.ListRoles)
				r.Get("/{id}", handler.GetRole)
				r.Put("/{id}", handler.UpdateRole)
				r.Delete("/{id}", handler.DeleteRole)

				r.Route("/{roleId}/permissions", func(r chi.Router) {
					r.Post("/", handler.AssignPermission)
					r.Delete("/{permissionId}", handler.RemovePermission)
					r.Get("/", handler.GetRolePermissions)
				})
			})

			r.Route("/permissions", func(r chi.Router) {
				r.Post("/", handler.CreatePermission)
				r.Get("/", handler.ListPermissions)
				r.Get("/{id}", handler.GetPermission)
				r.Put("/{id}", handler.UpdatePermission)
				r.Delete("/{id}", handler.DeletePermission)
			})

			r.Route("/users/{userId}/roles", func(r chi.Router) {
				r.Post("/", handler.AssignRole)
				r.Delete("/{roleId}", handler.RemoveRole)
				r.Get("/", handler.GetUserRoles)
			})

			r.Post("/check-permission", handler.CheckPermission)
		})
	})

	return r
}
