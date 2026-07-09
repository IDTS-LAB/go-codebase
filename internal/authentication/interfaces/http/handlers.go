package http

import (
	"encoding/json"
	"net/http"
	"time"

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
	_, err := h.svc.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		switch err {
		case service.ErrEmailAlreadyExists:
			utils.RespondConflict(w, "email already registered")
		default:
			utils.RespondInternalError(w, "failed to register user")
		}
		return
	}
	utils.RespondCreated(w, dto.MessageResponse{Message: "user registered successfully. Check your email for verification."})
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
		case service.ErrAccountLocked:
			utils.RespondError(w, http.StatusForbidden, "ACCOUNT_LOCKED", "account is temporarily locked due to too many failed attempts")
		case service.ErrEmailNotVerified:
			utils.RespondError(w, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "email is not verified")
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
	accessTokenJTI := ""
	if jti := r.Context().Value("access_token_jti"); jti != nil {
		accessTokenJTI, _ = jti.(string)
	}
	if err := h.svc.Logout(r.Context(), req.RefreshToken, accessTokenJTI, 15*time.Minute); err != nil {
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
// @Router /auth/logout-all [post]
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
// @Router /auth/me [get]
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

// VerifyEmail godoc
// @Summary Verify email address
// @Description Verify user email with token from email
// @Tags authentication
// @Param token query string true "Verification token"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Router /auth/verify-email [get]
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		utils.RespondBadRequest(w, "token is required")
		return
	}
	if err := h.svc.VerifyEmail(r.Context(), token); err != nil {
		switch err {
		case service.ErrInvalidVerifyToken, service.ErrVerifyTokenExpired:
			utils.RespondBadRequest(w, err.Error())
		default:
			utils.RespondInternalError(w, "failed to verify email")
		}
		return
	}
	utils.RespondSuccess(w, map[string]string{"message": "email verified successfully"})
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Send password reset email
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Email address"
// @Success 200 {object} utils.SuccessResponse
// @Router /auth/forgot-password [post]
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	if err := h.svc.ForgotPassword(r.Context(), req.Email); err != nil {
		utils.RespondInternalError(w, "failed to process request")
		return
	}
	utils.RespondSuccess(w, map[string]string{"message": "if the email exists, a reset link has been sent"})
}

// ResetPassword godoc
// @Summary Reset password
// @Description Reset password with token from email
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.ResetPasswordRequest true "Token and new password"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Router /auth/reset-password [post]
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		switch err {
		case service.ErrInvalidResetToken, service.ErrResetTokenExpired:
			utils.RespondBadRequest(w, err.Error())
		default:
			utils.RespondInternalError(w, "failed to reset password")
		}
		return
	}
	utils.RespondSuccess(w, map[string]string{"message": "password reset successfully"})
}

// ResendVerification godoc
// @Summary Resend verification email
// @Description Resend email verification link
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.ResendVerificationRequest true "Email address"
// @Success 200 {object} utils.SuccessResponse
// @Router /auth/resend-verification [post]
func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req dto.ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	if err := h.svc.ResendVerification(r.Context(), req.Email); err != nil {
		utils.RespondInternalError(w, "failed to resend verification")
		return
	}
	utils.RespondSuccess(w, map[string]string{"message": "if the email exists, a verification link has been sent"})
}