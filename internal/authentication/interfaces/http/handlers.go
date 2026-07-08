package http

import (
	"encoding/json"
	"net/http"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/service"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/google/uuid"
)

type Handler struct {
	svc       *service.AuthenticationService
	validator *validator.Validator
}

func NewHandler(svc *service.AuthenticationService, v *validator.Validator) *Handler {
	return &Handler{svc: svc, validator: v}
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account and return tokens
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Registration details"
// @Success 201 {object} utils.SuccessResponse{data=dto.TokenResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	user, err := h.svc.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		switch err {
		case service.ErrEmailAlreadyExists:
			utils.RespondConflict(w, "email already registered")
		default:
			utils.RespondInternalError(w, "failed to register user")
		}
		return
	}
	tokens, err := h.svc.GenerateTokens(r.Context(), user)
	if err != nil {
		utils.RespondInternalError(w, "failed to generate tokens")
		return
	}
	utils.RespondCreated(w, dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
	})
}

// Login godoc
// @Summary Login
// @Description Authenticate user and return tokens
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login credentials"
// @Success 200 {object} utils.SuccessResponse{data=dto.TokenResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Router /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	user, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			utils.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
		case service.ErrAccountDisabled:
			utils.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "account is disabled")
		default:
			utils.RespondInternalError(w, "failed to login")
		}
		return
	}
	tokens, err := h.svc.GenerateTokens(r.Context(), user)
	if err != nil {
		utils.RespondInternalError(w, "failed to generate tokens")
		return
	}
	utils.RespondSuccess(w, dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
	})
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Get new token pair using refresh token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.RefreshRequest true "Refresh token"
// @Success 200 {object} utils.SuccessResponse{data=dto.TokenResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Router /auth/refresh [post]
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	tokens, err := h.svc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		switch err {
		case service.ErrInvalidRefreshToken:
			utils.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired refresh token")
		default:
			utils.RespondInternalError(w, "failed to refresh token")
		}
		return
	}
	utils.RespondSuccess(w, dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
	})
}

// Logout godoc
// @Summary Logout
// @Description Revoke refresh token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.RefreshRequest true "Refresh token to revoke"
// @Success 200 {object} utils.SuccessResponse{data=dto.MessageResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Router /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		utils.RespondInternalError(w, "failed to logout")
		return
	}
	utils.RespondSuccess(w, dto.MessageResponse{Message: "logged out successfully"})
}

// LogoutAllSessions godoc
// @Summary Logout all sessions
// @Description Revoke all refresh tokens for the current user
// @Tags authentication
// @Produce json
// @Success 200 {object} utils.SuccessResponse{data=dto.MessageResponse}
// @Failure 401 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /auth/sessions/logout-all [post]
func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
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
	if err := h.svc.LogoutAll(r.Context(), uid); err != nil {
		utils.RespondInternalError(w, "failed to logout all sessions")
		return
	}
	utils.RespondSuccess(w, dto.MessageResponse{Message: "all sessions terminated"})
}

// Me godoc
// @Summary Get current user
// @Description Get the authenticated user's profile
// @Tags authentication
// @Produce json
// @Success 200 {object} utils.SuccessResponse{data=dto.UserResponse}
// @Failure 401 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /auth/sessions/me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		utils.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}
	utils.RespondSuccess(w, dto.UserResponse{
		ID:       userID,
		Email:    middleware.GetUserEmail(r.Context()),
		Name:     "",
		IsActive: true,
	})
}