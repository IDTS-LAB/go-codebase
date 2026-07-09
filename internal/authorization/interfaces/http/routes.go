package http

import (
	"net/http"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler, authMiddleware func(http.Handler) http.Handler, authorizer middleware.Authorizer) *chi.Mux {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.Timeout(30 * time.Second))

		r.Route("/roles", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "role", "create"))
				r.Post("/", handler.CreateRole)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "role", "list"))
				r.Get("/", handler.ListRoles)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "role", "read"))
				r.Get("/{id}", handler.GetRole)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "role", "update"))
				r.Put("/{id}", handler.UpdateRole)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "role", "delete"))
				r.Delete("/{id}", handler.DeleteRole)
			})

			r.Route("/{roleId}/permissions", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(middleware.Authorization(authorizer, "role_permission", "create"))
					r.Post("/", handler.AssignPermission)
				})
				r.Group(func(r chi.Router) {
					r.Use(middleware.Authorization(authorizer, "role_permission", "delete"))
					r.Delete("/{permissionId}", handler.RemovePermission)
				})
				r.Group(func(r chi.Router) {
					r.Use(middleware.Authorization(authorizer, "role_permission", "list"))
					r.Get("/", handler.GetRolePermissions)
				})
			})
		})

		r.Route("/permissions", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "permission", "create"))
				r.Post("/", handler.CreatePermission)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "permission", "list"))
				r.Get("/", handler.ListPermissions)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "permission", "read"))
				r.Get("/{id}", handler.GetPermission)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "permission", "update"))
				r.Put("/{id}", handler.UpdatePermission)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "permission", "delete"))
				r.Delete("/{id}", handler.DeletePermission)
			})
		})

		r.Route("/users/{userId}/roles", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "user_role", "create"))
				r.Post("/", handler.AssignRole)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "user_role", "delete"))
				r.Delete("/{roleId}", handler.RemoveRole)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authorization(authorizer, "user_role", "list"))
				r.Get("/", handler.GetUserRoles)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.Authorization(authorizer, "permission", "read"))
			r.Post("/check-permission", handler.CheckPermission)
		})
	})

	return r
}
