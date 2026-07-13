package command

import (
	"context"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/google/uuid"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type GenerateTokensCommand struct {
	User     *entity.User
	TokenTTL time.Duration
}

type GenerateTokensHandler struct {
	refreshRepo  repository.RefreshTokenRepository
	tokenService domain.TokenService
}

func NewGenerateTokensHandler(refreshRepo repository.RefreshTokenRepository, tokenService domain.TokenService) *GenerateTokensHandler {
	return &GenerateTokensHandler{refreshRepo: refreshRepo, tokenService: tokenService}
}

func (h *GenerateTokensHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(GenerateTokensCommand)
	refreshTokenTTL := 7 * 24 * time.Hour
	if c.TokenTTL > 0 {
		refreshTokenTTL = c.TokenTTL
	}
	accessTokenTTL := 15 * time.Minute

	tc := &domain.TokenClaims{
		UserID:   c.User.ID.String(),
		Email:    c.User.Email,
		Role:     "user",
		JTI:      uuid.New().String(),
		TenantID: middleware.GetTenantID(ctx),
	}
	accessToken, err := h.tokenService.GenerateToken(tc)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshTokenStr, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshToken := entity.NewRefreshToken(c.User.ID, refreshTokenStr, time.Now().Add(refreshTokenTTL))
	if err = h.refreshRepo.Create(ctx, refreshToken); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}
