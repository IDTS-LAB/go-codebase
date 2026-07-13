package command

import (
	"context"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
)

type ForgotPasswordCommand struct {
	Email string
}

type ForgotPasswordHandler struct {
	userRepo repository.UserRepository
	bus      events.EventBus
}

func NewForgotPasswordHandler(userRepo repository.UserRepository, bus events.EventBus) *ForgotPasswordHandler {
	return &ForgotPasswordHandler{userRepo: userRepo, bus: bus}
}

func (h *ForgotPasswordHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(ForgotPasswordCommand)
	user, err := h.userRepo.GetByEmail(ctx, c.Email)
	if err != nil {
		return nil, nil
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	expires := time.Now().Add(1 * time.Hour)
	hashed := hashToken(token)
	user.PasswordResetToken = &hashed
	user.PasswordResetExpires = &expires
	if err := h.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	_ = h.bus.Publish(ctx, events.Event{
		Type: event.PasswordResetRequestedEvent,
		Payload: event.PasswordResetRequested{
			Email:      user.Email,
			Name:       user.Name,
			ResetToken: token,
		},
	})
	return nil, nil
}
