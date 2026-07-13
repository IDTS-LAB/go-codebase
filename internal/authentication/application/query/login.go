package query

import (
	"context"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"golang.org/x/crypto/bcrypt"
)

type LoginQuery struct {
	Email    string
	Password string
}

type LoginHandler struct {
	userRepo         repository.UserRepository
	maxLoginAttempts int
	lockoutDuration  time.Duration
}

func NewLoginHandler(userRepo repository.UserRepository) *LoginHandler {
	return &LoginHandler{
		userRepo:         userRepo,
		maxLoginAttempts: 5,
		lockoutDuration:  15 * time.Minute,
	}
}

func (h *LoginHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(LoginQuery)
	user, err := h.userRepo.GetByEmail(ctx, q.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	if user.IsLocked() {
		return nil, ErrAccountLocked
	}

	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(q.Password)); err != nil {
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= h.maxLoginAttempts {
			user.Lock(h.lockoutDuration)
		}
		_ = h.userRepo.Update(ctx, user)
		return nil, ErrInvalidCredentials
	}

	user.Unlock()
	_ = h.userRepo.Update(ctx, user)

	return user, nil
}
