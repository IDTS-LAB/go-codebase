package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/service"
	"github.com/google/uuid"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *service.UserService
}

func NewHandler(svc *service.UserService) *Handler {
	return &Handler{svc: svc}
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type UpdateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	IsActive *bool  `json:"is_active"`
}

type ListResponse struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total"`
}

// List godoc
// @Summary List users
// @Description Get a paginated list of users
// @Tags users
// @Produce json
// @Param limit query int false "Page size" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} utils.SuccessResponse{data=ListResponse}
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /users [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	users, total, err := h.svc.List(r.Context(), offset, limit)
	if err != nil {
		utils.RespondInternalError(w, "failed to list users")
		return
	}

	resp := make([]UserResponse, len(users))
	for i, u := range users {
		resp[i] = UserResponse{
			ID:        u.ID.String(),
			Email:     u.Email,
			Name:      u.Name,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	utils.RespondSuccess(w, ListResponse{Users: resp, Total: total})
}

// Get godoc
// @Summary Get user by ID
// @Description Get a user by their ID
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} utils.SuccessResponse{data=UserResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid user ID")
		return
	}

	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		utils.RespondNotFound(w, "user not found")
		return
	}

	utils.RespondSuccess(w, UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// Me godoc
// @Summary Get current user
// @Description Get the authenticated user's profile
// @Tags users
// @Produce json
// @Success 200 {object} utils.SuccessResponse{data=UserResponse}
// @Failure 401 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /users/me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		utils.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		utils.RespondBadRequest(w, "invalid user ID")
		return
	}

	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		utils.RespondNotFound(w, "user not found")
		return
	}

	utils.RespondSuccess(w, UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// Update godoc
// @Summary Update user
// @Description Update a user's profile
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body UpdateUserRequest true "Update details"
// @Success 200 {object} utils.SuccessResponse{data=UserResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid user ID")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	user, err := h.svc.Update(r.Context(), id, req.Name, req.Email, isActive)
	if err != nil {
		utils.RespondNotFound(w, "user not found")
		return
	}

	utils.RespondSuccess(w, UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// Delete godoc
// @Summary Delete user
// @Description Soft-delete a user
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid user ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		utils.RespondNotFound(w, "user not found")
		return
	}

	utils.RespondSuccess(w, map[string]string{"message": "user deleted"})
}
