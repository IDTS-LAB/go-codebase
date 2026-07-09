package email

import (
	"log"
)

type ConsoleMailer struct {
	from        string
	fromName    string
	frontendURL string
}

func NewConsoleMailer(from, fromName, frontendURL string) *ConsoleMailer {
	return &ConsoleMailer{from: from, fromName: fromName, frontendURL: frontendURL}
}

func (m *ConsoleMailer) SendVerification(to, name, token string) error {
	verifyURL := m.frontendURL + "/verify-email?token=" + token
	content, err := renderTemplate("verification", TemplateData{Name: name, VerifyURL: verifyURL})
	if err != nil {
		return err
	}
	log.Printf("[EMAIL] To: %s | Subject: Verify your email\n%s", to, content)
	return nil
}

func (m *ConsoleMailer) SendPasswordReset(to, name, token string) error {
	resetURL := m.frontendURL + "/reset-password?token=" + token
	content, err := renderTemplate("password_reset", TemplateData{Name: name, ResetURL: resetURL})
	if err != nil {
		return err
	}
	log.Printf("[EMAIL] To: %s | Subject: Reset your password\n%s", to, content)
	return nil
}

func (m *ConsoleMailer) SendWelcome(to, name string) error {
	content, err := renderTemplate("welcome", TemplateData{Name: name})
	if err != nil {
		return err
	}
	log.Printf("[EMAIL] To: %s | Subject: Welcome %s!\n%s", to, name, content)
	return nil
}

func (m *ConsoleMailer) SendInvite(to, name, inviterName string) error {
	content, err := renderTemplate("invite", TemplateData{Name: name, InviterName: inviterName})
	if err != nil {
		return err
	}
	log.Printf("[EMAIL] To: %s | Subject: %s invited you\n%s", to, inviterName, content)
	return nil
}
