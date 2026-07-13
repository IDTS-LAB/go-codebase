package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	authEntity "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/user/public"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	commandBus  cqrs.CommandBus
	queryBus    cqrs.QueryBus
	profileProv public.UserProfileProvider
}

func NewHandler(commandBus cqrs.CommandBus, queryBus cqrs.QueryBus) *Handler {
	return &Handler{commandBus: commandBus, queryBus: queryBus}
}

type UserResponse struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	IsActive  bool     `json:"is_active"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
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

func userToResponse(user *authEntity.User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// List godoc
// @Summary List users
// @Description Get a paginated list of users
// @Tags users
// @Produce json
// @Param limit query int false "Page size" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} utils.APIResponse{data=ListResponse}
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /users [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.queryBus.Ask(r.Context(), query.ListUsersQuery{Cursor: cursor, Limit: limit})
	if err != nil {
		utils.MapErrorFromRequest(w, r, err)
		return
	}

	result := resp.(query.ListUsersResult)
	usersResp := make([]UserResponse, len(result.Users))
	for i, u := range result.Users {
		usersResp[i] = userToResponse(u)
	}

	utils.RespondCursorPaginated(w, usersResp, result.NextCursor, result.PrevCursor, result.HasNext, result.HasPrev, result.Limit)
}

// Get godoc
// @Summary Get user by ID
// @Description Get a user by their ID
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} utils.APIResponse{data=UserResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /users/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid user ID")
		return
	}

	resp, err := h.queryBus.Ask(r.Context(), query.GetUserQuery{ID: id})
	if err != nil {
		utils.Handle(w, r, nil, err)
		return
	}

	utils.RespondSuccess(w, userToResponse(resp.(*authEntity.User)))
}

// Me godoc
// @Summary Get current user
// @Description Get the authenticated user's profile
// @Tags users
// @Produce json
// @Success 200 {object} utils.APIResponse{data=UserResponse}
// @Failure 401 {object} utils.APIResponse
// @Security BearerAuth
// @Router /users/me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		utils.RespondUnauthorized(w, "user not authenticated")
		return
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		utils.RespondBadRequest(w, "invalid user ID")
		return
	}

	resp, err := h.profileProv.GetProfile(r.Context(), id)
	if err != nil {
		utils.Handle(w, r, nil, err)
		return
	}

	utils.RespondSuccess(w, resp)
}

// Update godoc
// @Summary Update user
// @Description Update a user's profile
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body UpdateUserRequest true "Update details"
// @Success 200 {object} utils.APIResponse{data=UserResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /users/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid user ID")
		return
	}

	var req UpdateUserRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	resp, err := h.commandBus.Dispatch(r.Context(), command.UpdateUserCommand{
		ID:       id,
		Name:     req.Name,
		Email:    req.Email,
		IsActive: isActive,
	})
	if err != nil {
		utils.Handle(w, r, nil, err)
		return
	}

	utils.RespondSuccess(w, userToResponse(resp.(*authEntity.User)))
}

// Delete godoc
// @Summary Delete user
// @Description Soft-delete a user
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /users/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid user ID")
		return
	}

	_, err = h.commandBus.Dispatch(r.Context(), command.DeleteUserCommand{ID: id})
	utils.Handle(w, r, map[string]string{"message": "user deleted"}, err)
}
