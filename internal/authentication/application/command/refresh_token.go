package command

import (
	"context"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
)

type RefreshTokenCommand struct {
	RefreshToken string
	TokenTTL     time.Duration
}

type RefreshTokenHandler struct {
	refreshRepo    repository.RefreshTokenRepository
	userRepo       repository.UserRepository
	generateTokens *GenerateTokensHandler
}

func NewRefreshTokenHandler(
	refreshRepo repository.RefreshTokenRepository,
	userRepo repository.UserRepository,
	generateTokens *GenerateTokensHandler,
) *RefreshTokenHandler {
	return &RefreshTokenHandler{refreshRepo: refreshRepo, userRepo: userRepo, generateTokens: generateTokens}
}

func (h *RefreshTokenHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(RefreshTokenCommand)
	refreshToken, err := h.refreshRepo.GetByToken(ctx, c.RefreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if refreshToken.IsExpired() || refreshToken.IsRevoked() {
		return nil, ErrInvalidRefreshToken
	}

	user, err := h.userRepo.GetByID(ctx, refreshToken.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	if err := h.refreshRepo.Revoke(ctx, c.RefreshToken); err != nil {
		return nil, err
	}

	return h.generateTokens.Handle(ctx, GenerateTokensCommand{User: user, TokenTTL: c.TokenTTL})
}
