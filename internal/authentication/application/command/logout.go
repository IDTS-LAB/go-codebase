package command

import (
	"context"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
)

type LogoutCommand struct {
	RefreshToken   string
	AccessTokenJTI string
	AccessTokenTTL time.Duration
	Denylist       func(ctx context.Context, jti string, ttl time.Duration) error
}

type LogoutHandler struct {
	refreshRepo repository.RefreshTokenRepository
}

func NewLogoutHandler(refreshRepo repository.RefreshTokenRepository) *LogoutHandler {
	return &LogoutHandler{refreshRepo: refreshRepo}
}

func (h *LogoutHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(LogoutCommand)
	if c.Denylist != nil && c.AccessTokenJTI != "" {
		_ = c.Denylist(ctx, c.AccessTokenJTI, c.AccessTokenTTL)
	}
	return nil, h.refreshRepo.Revoke(ctx, c.RefreshToken)
}
