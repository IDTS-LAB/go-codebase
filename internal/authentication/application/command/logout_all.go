package command

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/google/uuid"
)

type LogoutAllCommand struct {
	UserID uuid.UUID
}

type LogoutAllHandler struct {
	refreshRepo repository.RefreshTokenRepository
}

func NewLogoutAllHandler(refreshRepo repository.RefreshTokenRepository) *LogoutAllHandler {
	return &LogoutAllHandler{refreshRepo: refreshRepo}
}

func (h *LogoutAllHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(LogoutAllCommand)
	return nil, h.refreshRepo.RevokeAllByUserID(ctx, c.UserID)
}
