package email

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
)

type ConsoleMailer struct {
	from        string
	fromName    string
	frontendURL string
	log         domain.Logger
}

func NewConsoleMailer(from, fromName, frontendURL string, log domain.Logger) *ConsoleMailer {
	return &ConsoleMailer{from: from, fromName: fromName, frontendURL: frontendURL, log: log}
}

func (m *ConsoleMailer) SendVerification(to, name, token string) error {
	verifyURL := m.frontendURL + "/verify-email?token=" + token
	content, err := renderTemplate("verification", TemplateData{Name: name, VerifyURL: verifyURL})
	if err != nil {
		return err
	}
	m.log.Info(context.Background(), "[EMAIL] verification",
		domain.String("to", to),
		domain.String("subject", "Verify your email"),
		domain.String("content", content),
	)
	return nil
}

func (m *ConsoleMailer) SendPasswordReset(to, name, token string) error {
	resetURL := m.frontendURL + "/reset-password?token=" + token
	content, err := renderTemplate("password_reset", TemplateData{Name: name, ResetURL: resetURL})
	if err != nil {
		return err
	}
	m.log.Info(context.Background(), "[EMAIL] password reset",
		domain.String("to", to),
		domain.String("subject", "Reset your password"),
		domain.String("content", content),
	)
	return nil
}

func (m *ConsoleMailer) SendWelcome(to, name string) error {
	content, err := renderTemplate("welcome", TemplateData{Name: name})
	if err != nil {
		return err
	}
	m.log.Info(context.Background(), "[EMAIL] welcome",
		domain.String("to", to),
		domain.String("subject", "Welcome "+name+"!"),
		domain.String("content", content),
	)
	return nil
}

func (m *ConsoleMailer) SendInvite(to, name, inviterName string) error {
	content, err := renderTemplate("invite", TemplateData{Name: name, InviterName: inviterName})
	if err != nil {
		return err
	}
	m.log.Info(context.Background(), "[EMAIL] invite",
		domain.String("to", to),
		domain.String("subject", inviterName+" invited you"),
		domain.String("content", content),
	)
	return nil
}
