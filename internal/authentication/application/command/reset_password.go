package command

import (
	"context"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"golang.org/x/crypto/bcrypt"
)

type ResetPasswordCommand struct {
	Token       string
	NewPassword string
}

type ResetPasswordHandler struct {
	userRepo    repository.UserRepository
	refreshRepo repository.RefreshTokenRepository
}

func NewResetPasswordHandler(userRepo repository.UserRepository, refreshRepo repository.RefreshTokenRepository) *ResetPasswordHandler {
	return &ResetPasswordHandler{userRepo: userRepo, refreshRepo: refreshRepo}
}

func (h *ResetPasswordHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(ResetPasswordCommand)
	hashed := hashToken(c.Token)
	user, err := h.userRepo.GetByResetToken(ctx, hashed)
	if err != nil {
		return nil, ErrInvalidResetToken
	}
	if user.PasswordResetExpires != nil && time.Now().After(*user.PasswordResetExpires) {
		return nil, ErrResetTokenExpired
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user.Password = string(hashedPassword)
	user.PasswordResetToken = nil
	user.PasswordResetExpires = nil
	_ = h.refreshRepo.RevokeAllByUserID(ctx, user.ID)
	return nil, h.userRepo.Update(ctx, user)
}
