package email

import (
	"log"
)

type ConsoleMailer struct {
	from     string
	fromName string
}

func NewConsoleMailer(from, fromName string) *ConsoleMailer {
	return &ConsoleMailer{from: from, fromName: fromName}
}

func (m *ConsoleMailer) SendVerification(to, name, token string) error {
	log.Printf("[EMAIL] To: %s | Subject: Verify your email | Link: %s/verify-email?token=%s", to, name, token)
	return nil
}

func (m *ConsoleMailer) SendPasswordReset(to, name, token string) error {
	log.Printf("[EMAIL] To: %s | Subject: Reset your password | Link: %s/reset-password?token=%s", to, name, token)
	return nil
}

func (m *ConsoleMailer) SendWelcome(to, name string) error {
	log.Printf("[EMAIL] To: %s | Subject: Welcome %s!", to, name)
	return nil
}

func (m *ConsoleMailer) SendInvite(to, name, inviterName string) error {
	log.Printf("[EMAIL] To: %s | Subject: %s invited you to join", to, inviterName)
	return nil
}
