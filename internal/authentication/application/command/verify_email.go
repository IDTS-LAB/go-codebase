package command

import (
	"context"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
)

type VerifyEmailCommand struct {
	Token string
}

type VerifyEmailHandler struct {
	userRepo repository.UserRepository
	bus      events.EventBus
}

func NewVerifyEmailHandler(userRepo repository.UserRepository, bus events.EventBus) *VerifyEmailHandler {
	return &VerifyEmailHandler{userRepo: userRepo, bus: bus}
}

func (h *VerifyEmailHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(VerifyEmailCommand)
	hashed := hashToken(c.Token)
	user, err := h.userRepo.GetByVerifyToken(ctx, hashed)
	if err != nil {
		return nil, ErrInvalidVerifyToken
	}
	if user.EmailVerifyExpires != nil && time.Now().After(*user.EmailVerifyExpires) {
		return nil, ErrVerifyTokenExpired
	}

	user.EmailVerified = true
	user.EmailVerifyToken = nil
	user.EmailVerifyExpires = nil
	if err := h.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	_ = h.bus.Publish(ctx, events.Event{
		Type: event.EmailVerifiedEvent,
		Payload: event.EmailVerified{
			UserID: user.ID.String(),
			Email:  user.Email,
			Name:   user.Name,
		},
	})
	return map[string]string{"message": "email verified successfully"}, nil
}
