package email

import (
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type SendGridMailer struct {
	client      *sendgrid.Client
	from        string
	fromName    string
	frontendURL string
}

func NewSendGridMailer(apiKey, from, fromName, frontendURL string) *SendGridMailer {
	return &SendGridMailer{
		client:      sendgrid.NewSendClient(apiKey),
		from:        from,
		fromName:    fromName,
		frontendURL: frontendURL,
	}
}

func (m *SendGridMailer) SendVerification(to, name, token string) error {
	subject := "Verify your email address"
	verifyURL := m.frontendURL + "/verify-email?token=" + token
	htmlContent, err := renderTemplate("verification", TemplateData{Name: name, VerifyURL: verifyURL})
	if err != nil {
		return err
	}
	return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) SendPasswordReset(to, name, token string) error {
	subject := "Reset your password"
	resetURL := m.frontendURL + "/reset-password?token=" + token
	htmlContent, err := renderTemplate("password_reset", TemplateData{Name: name, ResetURL: resetURL})
	if err != nil {
		return err
	}
	return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) SendWelcome(to, name string) error {
	subject := fmt.Sprintf("Welcome %s!", name)
	htmlContent, err := renderTemplate("welcome", TemplateData{Name: name})
	if err != nil {
		return err
	}
	return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) SendInvite(to, name, inviterName string) error {
	subject := fmt.Sprintf("%s invited you to join", inviterName)
	htmlContent, err := renderTemplate("invite", TemplateData{Name: name, InviterName: inviterName})
	if err != nil {
		return err
	}
	return m.send(to, subject, htmlContent)
}

func (m *SendGridMailer) send(to, subject, htmlContent string) error {
	from := mail.NewEmail(m.fromName, m.from)
	message := mail.NewSingleEmail(from, subject, mail.NewEmail(to, to), "", htmlContent)
	_, err := m.client.Send(message)
	if err != nil {
		return fmt.Errorf("sendgrid: send email: %w", err)
	}
	return nil
}
