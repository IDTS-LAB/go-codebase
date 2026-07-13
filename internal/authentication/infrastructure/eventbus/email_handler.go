package eventbus

import (
	"context"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/event"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
)

type EmailHandler struct {
	mailer domain.Emailer
}

func NewEmailHandler(mailer domain.Emailer) *EmailHandler {
	return &EmailHandler{mailer: mailer}
}

func (h *EmailHandler) Register(bus events.EventBus) {
	bus.Subscribe(event.UserRegisteredEvent, h.onUserRegistered)
	bus.Subscribe(event.EmailVerifiedEvent, h.onEmailVerified)
	bus.Subscribe(event.PasswordResetRequestedEvent, h.onPasswordResetRequested)
}

func (h *EmailHandler) onUserRegistered(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(event.UserRegistered)
	if !ok {
		return fmt.Errorf("invalid payload type for %s", e.Type)
	}
	return h.mailer.SendVerification(payload.Email, payload.Name, payload.VerificationToken)
}

func (h *EmailHandler) onEmailVerified(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(event.EmailVerified)
	if !ok {
		return fmt.Errorf("invalid payload type for %s", e.Type)
	}
	return h.mailer.SendWelcome(payload.Email, payload.Name)
}

func (h *EmailHandler) onPasswordResetRequested(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(event.PasswordResetRequested)
	if !ok {
		return fmt.Errorf("invalid payload type for %s", e.Type)
	}
	return h.mailer.SendPasswordReset(payload.Email, payload.Name, payload.ResetToken)
}
