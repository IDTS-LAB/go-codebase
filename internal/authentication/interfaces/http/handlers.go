package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
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

// Register godoc
// @Summary Register a new user
// @Description Create a new user account and return tokens
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Registration details"
// @Success 201 {object} utils.APIResponse{data=dto.TokenResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 409 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
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
	_, err := h.commandBus.Dispatch(r.Context(), command.RegisterUserCommand{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil && errors.Is(err, command.ErrEmailAlreadyExists) {
		utils.RespondConflict(w, "email already registered")
		return
	}
	utils.HandleCreated(w, r, dto.MessageResponse{Message: "user registered successfully. Check your email for verification."}, err)
}

// Login godoc
// @Summary Login
// @Description Authenticate user and return tokens
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login credentials"
// @Success 200 {object} utils.APIResponse{data=dto.TokenResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse
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
	userResp, err := h.queryBus.Ask(r.Context(), query.LoginQuery{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, query.ErrInvalidCredentials):
			utils.RespondUnauthorized(w, "invalid email or password")
		case errors.Is(err, query.ErrAccountDisabled):
			utils.RespondUnauthorized(w, "account is disabled")
		case errors.Is(err, query.ErrAccountLocked):
			utils.RespondForbidden(w, "ACCOUNT_LOCKED", "account is temporarily locked due to too many failed attempts")
		case errors.Is(err, query.ErrEmailNotVerified):
			utils.RespondForbidden(w, "EMAIL_NOT_VERIFIED", "email is not verified")
		default:
			utils.RespondInternalError(w, "failed to login")
		}
		return
	}
	tokenResp, err := h.commandBus.Dispatch(r.Context(), command.GenerateTokensCommand{
		User: userResp.(*entity.User),
	})
	if err != nil {
		utils.RespondInternalError(w, "failed to generate tokens")
		return
	}
	tokens := tokenResp.(*command.TokenPair)
	utils.Handle(w, r, dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
	}, nil)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Get new token pair using refresh token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.RefreshRequest true "Refresh token"
// @Success 200 {object} utils.APIResponse{data=dto.TokenResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse
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
	resp, err := h.commandBus.Dispatch(r.Context(), command.RefreshTokenCommand{
		RefreshToken: req.RefreshToken,
	})
	if err != nil && errors.Is(err, command.ErrInvalidRefreshToken) {
		utils.RespondUnauthorized(w, "invalid or expired refresh token")
		return
	}
	if err != nil {
		utils.RespondInternalError(w, "failed to refresh token")
		return
	}
	tokens := resp.(*command.TokenPair)
	utils.Handle(w, r, dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
	}, nil)
}

// Logout godoc
// @Summary Logout
// @Description Revoke refresh token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.RefreshRequest true "Refresh token to revoke"
// @Success 200 {object} utils.APIResponse{data=dto.MessageResponse}
// @Failure 400 {object} utils.APIResponse
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
	_, err := h.commandBus.Dispatch(r.Context(), command.LogoutCommand{
		RefreshToken:   req.RefreshToken,
		AccessTokenJTI: accessTokenJTI,
		AccessTokenTTL: 15 * time.Minute,
	})
	utils.Handle(w, r, dto.MessageResponse{Message: "logged out successfully"}, err)
}

// LogoutAllSessions godoc
// @Summary Logout all sessions
// @Description Revoke all refresh tokens for the current user
// @Tags authentication
// @Produce json
// @Success 200 {object} utils.APIResponse{data=dto.MessageResponse}
// @Failure 401 {object} utils.APIResponse
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
	_, err = h.commandBus.Dispatch(r.Context(), command.LogoutAllCommand{UserID: uid})
	utils.Handle(w, r, dto.MessageResponse{Message: "all sessions terminated"}, err)
}

// Me godoc
// @Summary Get current user
// @Description Get the authenticated user's profile
// @Tags authentication
// @Produce json
// @Success 200 {object} utils.APIResponse{data=dto.UserResponse}
// @Failure 401 {object} utils.APIResponse
// @Security BearerAuth
// @Router /auth/me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		utils.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}
	utils.Handle(w, r, dto.UserResponse{
		ID:       userID,
		Email:    middleware.GetUserEmail(r.Context()),
		Name:     "",
		IsActive: true,
	}, nil)
}

// VerifyEmail godoc
// @Summary Verify email address
// @Description Verify user email with token from email
// @Tags authentication
// @Param token query string true "Verification token"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Router /auth/verify-email [get]
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		utils.RespondBadRequest(w, "token is required")
		return
	}
	_, err := h.commandBus.Dispatch(r.Context(), command.VerifyEmailCommand{Token: token})
	if err != nil && (errors.Is(err, command.ErrInvalidVerifyToken) || errors.Is(err, command.ErrVerifyTokenExpired)) {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	utils.Handle(w, r, map[string]string{"message": "email verified successfully"}, err)
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Send password reset email
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Email address"
// @Success 200 {object} utils.APIResponse
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
	_, _ = h.commandBus.Dispatch(r.Context(), command.ForgotPasswordCommand{Email: req.Email})
	utils.RespondSuccess(w, map[string]string{"message": "if the email exists, a reset link has been sent"})
}

// ResetPassword godoc
// @Summary Reset password
// @Description Reset password with token from email
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.ResetPasswordRequest true "Token and new password"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
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
	_, err := h.commandBus.Dispatch(r.Context(), command.ResetPasswordCommand{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})
	if err != nil && (errors.Is(err, command.ErrInvalidResetToken) || errors.Is(err, command.ErrResetTokenExpired)) {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	utils.Handle(w, r, map[string]string{"message": "password reset successfully"}, err)
}

// ResendVerification godoc
// @Summary Resend verification email
// @Description Resend email verification link
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.ResendVerificationRequest true "Email address"
// @Success 200 {object} utils.APIResponse
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
	_, _ = h.commandBus.Dispatch(r.Context(), command.ResendVerificationCommand{Email: req.Email})
	utils.RespondSuccess(w, map[string]string{"message": "if the email exists, a verification link has been sent"})
}
