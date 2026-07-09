package email

import (
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type SendGridMailer struct {
	apiKey      string
	from        string
	fromName    string
	frontendURL string
}

func NewSendGridMailer(apiKey, from, fromName, frontendURL string) *SendGridMailer {
	return &SendGridMailer{apiKey: apiKey, from: from, fromName: fromName, frontendURL: frontendURL}
}

func (m *SendGridMailer) SendVerification(to, name, token string) error {
	subject := "Verify your email address"
	verifyURL := m.frontendURL + "/verify-email?token=" + token
	htmlContent := fmt.Sprintf(`<p>Hello %s,</p><p>Please verify your email by clicking the link below:</p><p><a href="%s">Verify Email</a></p><p>If you didn't create an account, please ignore this email.</p>`, name, verifyURL)
	return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) SendPasswordReset(to, name, token string) error {
	subject := "Reset your password"
	resetURL := m.frontendURL + "/reset-password?token=" + token
	htmlContent := fmt.Sprintf(`<p>Hello %s,</p><p>You requested a password reset. Click the link below:</p><p><a href="%s">Reset Password</a></p><p>This link expires in 1 hour.</p>`, name, resetURL)
	return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) SendWelcome(to, name string) error {
	subject := fmt.Sprintf("Welcome %s!", name)
	htmlContent := fmt.Sprintf(`<p>Hello %s,</p><p>Welcome to our platform! Your account is now active.</p>`, name)
	return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) SendInvite(to, name, inviterName string) error {
	subject := fmt.Sprintf("%s invited you to join", inviterName)
	htmlContent := fmt.Sprintf(`<p>Hello %s,</p><p>%s has invited you to join our platform.</p>`, name, inviterName)
	return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) send(to, subject, htmlContent string) error {
	from := mail.NewEmail(m.fromName, m.from)
	message := mail.NewSingleEmail(from, subject, mail.NewEmail(to, to), "", htmlContent)
	client := sendgrid.NewSendClient(m.apiKey)
	_, err := client.Send(message)
	return err
}
