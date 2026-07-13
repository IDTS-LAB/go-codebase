package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	commandBus cqrs.CommandBus
	queryBus   cqrs.QueryBus
	validator  *validator.Validator
}

func NewHandler(commandBus cqrs.CommandBus, queryBus cqrs.QueryBus, v *validator.Validator) *Handler {
	return &Handler{commandBus: commandBus, queryBus: queryBus, validator: v}
}

// CreateRole godoc
// @Summary Create a role
// @Description Create a new role
// @Tags authorization
// @Accept json
// @Produce json
// @Param request body dto.CreateRoleRequest true "Role to create"
// @Success 201 {object} utils.APIResponse{data=dto.RoleResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 409 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/roles [post]
func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	resp, err := h.commandBus.Dispatch(r.Context(), command.CreateRoleCommand{
		Name:        req.Name,
		Description: req.Description,
	})
	utils.HandleCreated(w, r, resp, err)
}

// ListRoles godoc
// @Summary List roles
// @Description Get all roles with pagination
// @Tags authorization
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} utils.APIResponse{data=dto.ListResponse}
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/roles [get]
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	cursorStr := r.URL.Query().Get("cursor")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	var cursor *string
	if cursorStr != "" {
		cursor = &cursorStr
	}

	resp, err := h.queryBus.Ask(r.Context(), query.ListRolesQuery{Cursor: cursor, Limit: limit})
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	result := resp.(query.ListRolesResult)
	utils.RespondCursorPaginated(w, result.Roles, result.NextCursor, result.PrevCursor, result.HasNext, result.HasPrev, limit)
}

// GetRole godoc
// @Summary Get a role
// @Description Get a role by ID
// @Tags authorization
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} utils.APIResponse{data=dto.RoleResponse}
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/roles/{id} [get]
func (h *Handler) GetRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid role ID")
		return
	}
	resp, err := h.queryBus.Ask(r.Context(), query.GetRoleQuery{ID: id})
	utils.Handle(w, r, resp, err)
}

// UpdateRole godoc
// @Summary Update a role
// @Description Update an existing role
// @Tags authorization
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Param request body dto.UpdateRoleRequest true "Fields to update"
// @Success 200 {object} utils.APIResponse{data=dto.RoleResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/roles/{id} [put]
func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid role ID")
		return
	}
	var req dto.UpdateRoleRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	resp, err := h.commandBus.Dispatch(r.Context(), command.UpdateRoleCommand{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	})
	utils.Handle(w, r, resp, err)
}

// DeleteRole godoc
// @Summary Delete a role
// @Description Delete a role by ID
// @Tags authorization
// @Param id path string true "Role ID"
// @Success 200 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/roles/{id} [delete]
func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid role ID")
		return
	}
	_, err = h.commandBus.Dispatch(r.Context(), command.DeleteRoleCommand{ID: id})
	utils.HandleNoContent(w, r, err)
}

// CreatePermission godoc
// @Summary Create a permission
// @Description Create a new permission
// @Tags authorization
// @Accept json
// @Produce json
// @Param request body dto.CreatePermissionRequest true "Permission to create"
// @Success 201 {object} utils.APIResponse{data=dto.PermissionResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 409 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/permissions [post]
func (h *Handler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	resp, err := h.commandBus.Dispatch(r.Context(), command.CreatePermissionCommand{
		Name:        req.Name,
		Description: req.Description,
		Resource:    req.Resource,
		Action:      req.Action,
	})
	utils.HandleCreated(w, r, resp, err)
}

// ListPermissions godoc
// @Summary List permissions
// @Description Get all permissions with pagination
// @Tags authorization
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} utils.APIResponse{data=dto.ListResponse}
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/permissions [get]
func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	cursorStr := r.URL.Query().Get("cursor")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	var cursor *string
	if cursorStr != "" {
		cursor = &cursorStr
	}

	resp, err := h.queryBus.Ask(r.Context(), query.ListPermissionsQuery{Cursor: cursor, Limit: limit})
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	result := resp.(query.ListPermissionsResult)
	utils.RespondCursorPaginated(w, result.Permissions, result.NextCursor, result.PrevCursor, result.HasNext, result.HasPrev, limit)
}

// GetPermission godoc
// @Summary Get a permission
// @Description Get a permission by ID
// @Tags authorization
// @Produce json
// @Param id path string true "Permission ID"
// @Success 200 {object} utils.APIResponse{data=dto.PermissionResponse}
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/permissions/{id} [get]
func (h *Handler) GetPermission(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid permission ID")
		return
	}
	resp, err := h.queryBus.Ask(r.Context(), query.GetPermissionQuery{ID: id})
	utils.Handle(w, r, resp, err)
}

// UpdatePermission godoc
// @Summary Update a permission
// @Description Update an existing permission
// @Tags authorization
// @Accept json
// @Produce json
// @Param id path string true "Permission ID"
// @Param request body dto.UpdatePermissionRequest true "Fields to update"
// @Success 200 {object} utils.APIResponse{data=dto.PermissionResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/permissions/{id} [put]
func (h *Handler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid permission ID")
		return
	}
	var req dto.UpdatePermissionRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	resp, err := h.commandBus.Dispatch(r.Context(), command.UpdatePermissionCommand{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Resource:    req.Resource,
		Action:      req.Action,
	})
	utils.Handle(w, r, resp, err)
}

// DeletePermission godoc
// @Summary Delete a permission
// @Description Delete a permission by ID
// @Tags authorization
// @Param id path string true "Permission ID"
// @Success 200 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/permissions/{id} [delete]
func (h *Handler) DeletePermission(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid permission ID")
		return
	}
	_, err = h.commandBus.Dispatch(r.Context(), command.DeletePermissionCommand{ID: id})
	utils.HandleNoContent(w, r, err)
}

// AssignRole godoc
// @Summary Assign role to user
// @Description Assign a role to a user
// @Tags authorization
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param request body dto.AssignRoleRequest true "Role to assign"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/users/{userId}/roles [post]
func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	var req dto.AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	_, err := h.commandBus.Dispatch(r.Context(), command.AssignRoleCommand{
		UserID: req.UserID,
		RoleID: req.RoleID,
	})
	utils.HandleNoContent(w, r, err)
}

// RemoveRole godoc
// @Summary Remove role from user
// @Description Remove a role from a user
// @Tags authorization
// @Param userId path string true "User ID"
// @Param roleId path string true "Role ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/users/{userId}/roles/{roleId} [delete]
func (h *Handler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid user ID")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleId"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid role ID")
		return
	}
	_, err = h.commandBus.Dispatch(r.Context(), command.UnassignRoleCommand{
		UserID: userID,
		RoleID: roleID,
	})
	utils.HandleNoContent(w, r, err)
}

// GetUserRoles godoc
// @Summary Get user roles
// @Description Get all roles assigned to a user
// @Tags authorization
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} utils.APIResponse{data=[]dto.RoleResponse}
// @Failure 400 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/users/{userId}/roles [get]
func (h *Handler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid user ID")
		return
	}
	resp, err := h.queryBus.Ask(r.Context(), query.GetUserRolesQuery{UserID: userID})
	utils.Handle(w, r, resp, err)
}

// AssignPermission godoc
// @Summary Assign permission to role
// @Description Assign a permission to a role
// @Tags authorization
// @Accept json
// @Produce json
// @Param roleId path string true "Role ID"
// @Param request body dto.AssignPermissionRequest true "Permission to assign"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/roles/{roleId}/permissions [post]
func (h *Handler) AssignPermission(w http.ResponseWriter, r *http.Request) {
	var req dto.AssignPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	_, err := h.commandBus.Dispatch(r.Context(), command.AssignPermissionCommand{
		RoleID:       req.RoleID,
		PermissionID: req.PermissionID,
	})
	utils.HandleNoContent(w, r, err)
}

// RemovePermission godoc
// @Summary Remove permission from role
// @Description Remove a permission from a role
// @Tags authorization
// @Param roleId path string true "Role ID"
// @Param permissionId path string true "Permission ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/roles/{roleId}/permissions/{permissionId} [delete]
func (h *Handler) RemovePermission(w http.ResponseWriter, r *http.Request) {
	roleID, err := uuid.Parse(chi.URLParam(r, "roleId"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid role ID")
		return
	}
	permID, err := uuid.Parse(chi.URLParam(r, "permissionId"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid permission ID")
		return
	}
	_, err = h.commandBus.Dispatch(r.Context(), command.UnassignPermissionCommand{
		RoleID:       roleID,
		PermissionID: permID,
	})
	utils.HandleNoContent(w, r, err)
}

// GetRolePermissions godoc
// @Summary Get role permissions
// @Description Get all permissions assigned to a role
// @Tags authorization
// @Produce json
// @Param roleId path string true "Role ID"
// @Success 200 {object} utils.APIResponse{data=[]dto.PermissionResponse}
// @Failure 400 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/roles/{roleId}/permissions [get]
func (h *Handler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID, err := uuid.Parse(chi.URLParam(r, "roleId"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid role ID")
		return
	}
	resp, err := h.queryBus.Ask(r.Context(), query.GetRolePermissionsQuery{RoleID: roleID})
	utils.Handle(w, r, resp, err)
}

// CheckPermission godoc
// @Summary Check user permission
// @Description Check if the current user has a specific permission
// @Tags authorization
// @Accept json
// @Produce json
// @Param request body dto.CheckPermissionRequest true "Permission to check"
// @Success 200 {object} utils.APIResponse{data=dto.CheckPermissionResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/sessions/check-permission [post]
func (h *Handler) CheckPermission(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		utils.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		utils.RespondBadRequest(w, "invalid user ID")
		return
	}
	var req dto.CheckPermissionRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err = h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	resp, err := h.queryBus.Ask(r.Context(), query.CheckPermissionQuery{
		UserID:   uid,
		Resource: req.Resource,
		Action:   req.Action,
	})
	allowed, _ := resp.(bool)
	utils.Handle(w, r, dto.CheckPermissionResponse{Allowed: allowed}, err)
}
