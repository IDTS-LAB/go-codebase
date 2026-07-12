package command

import (
	"context"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
)

type ResendVerificationCommand struct {
	Email string
}

type ResendVerificationHandler struct {
	userRepo repository.UserRepository
	bus      events.EventBus
}

func NewResendVerificationHandler(userRepo repository.UserRepository, bus events.EventBus) *ResendVerificationHandler {
	return &ResendVerificationHandler{userRepo: userRepo, bus: bus}
}

func (h *ResendVerificationHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(ResendVerificationCommand)
	user, err := h.userRepo.GetByEmail(ctx, c.Email)
	if err != nil {
		return nil, nil
	}
	if user.EmailVerified {
		return nil, nil
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	expires := time.Now().Add(24 * time.Hour)
	hashed := hashToken(token)
	user.EmailVerifyToken = &hashed
	user.EmailVerifyExpires = &expires
	if err := h.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	_ = h.bus.Publish(ctx, events.Event{
		Type: event.UserRegisteredEvent,
		Payload: event.UserRegistered{
			Email:             user.Email,
			Name:              user.Name,
			VerificationToken: token,
		},
	})
	return nil, nil
}
