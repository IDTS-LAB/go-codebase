package eventbus

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
)

type EmailHandler struct {
	mailer domain.Emailer
	log    domain.Logger
}

func NewEmailHandler(mailer domain.Emailer, log domain.Logger) *EmailHandler {
	return &EmailHandler{mailer: mailer, log: log}
}

func (h *EmailHandler) Register(bus events.EventBus) {
	bus.Subscribe(event.UserRegisteredEvent, h.onUserRegistered)
	bus.Subscribe(event.EmailVerifiedEvent, h.onEmailVerified)
	bus.Subscribe(event.PasswordResetRequestedEvent, h.onPasswordResetRequested)
}

func (h *EmailHandler) onUserRegistered(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(event.UserRegistered)
	if !ok {
		return nil
	}
	if err := h.mailer.SendVerification(payload.Email, payload.Name, payload.VerificationToken); err != nil {
		h.log.Error(ctx, "failed to send verification email", domain.Error(err))
	}
	return nil
}

func (h *EmailHandler) onEmailVerified(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(event.EmailVerified)
	if !ok {
		return nil
	}
	if err := h.mailer.SendWelcome(payload.Email, payload.Name); err != nil {
		h.log.Error(ctx, "failed to send welcome email", domain.Error(err))
	}
	return nil
}

func (h *EmailHandler) onPasswordResetRequested(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(event.PasswordResetRequested)
	if !ok {
		return nil
	}
	if err := h.mailer.SendPasswordReset(payload.Email, payload.Name, payload.ResetToken); err != nil {
		h.log.Error(ctx, "failed to send password reset email", domain.Error(err))
	}
	return nil
}
