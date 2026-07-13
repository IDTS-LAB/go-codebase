package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

type SMTPMailer struct {
	host        string
	port        int
	username    string
	password    string
	useTLS      bool
	from        string
	fromName    string
	frontendURL string
}

func NewSMTPMailer(host string, port int, username, password string, useTLS bool, from, fromName, frontendURL string) *SMTPMailer {
	return &SMTPMailer{
		host: host, port: port, username: username,
		password: password, useTLS: useTLS, from: from, fromName: fromName,
		frontendURL: frontendURL,
	}
}

func (m *SMTPMailer) SendVerification(to, name, token string) error {
	subject := "Verify your email address"
	verifyURL := m.frontendURL + "/verify-email?token=" + token
	content, err := renderTemplate("verification", TemplateData{Name: name, VerifyURL: verifyURL})
	if err != nil {
		return err
	}
	return m.send(to, subject, content)
}

func (m *SMTPMailer) SendPasswordReset(to, name, token string) error {
	subject := "Reset your password"
	resetURL := m.frontendURL + "/reset-password?token=" + token
	content, err := renderTemplate("password_reset", TemplateData{Name: name, ResetURL: resetURL})
	if err != nil {
		return err
	}
	return m.send(to, subject, content)
}

func (m *SMTPMailer) SendWelcome(to, name string) error {
	subject := fmt.Sprintf("Welcome %s!", name)
	content, err := renderTemplate("welcome", TemplateData{Name: name})
	if err != nil {
		return err
	}
	return m.send(to, subject, content)
}

func (m *SMTPMailer) SendInvite(to, name, inviterName string) error {
	subject := fmt.Sprintf("%s invited you to join", inviterName)
	content, err := renderTemplate("invite", TemplateData{Name: name, InviterName: inviterName})
	if err != nil {
		return err
	}
	return m.send(to, subject, content)
}

func (m *SMTPMailer) send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	auth := smtp.PlainAuth("", m.username, m.password, m.host)

	msg := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		m.fromName, m.from, to, subject, body)

	if !m.useTLS {
		return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
	}

	if m.port == 465 {
		return m.sendWithImplicitTLS(addr, auth, to, msg)
	}
	return m.sendWithSTARTTLS(addr, auth, to, msg)
}

func (m *SMTPMailer) sendWithImplicitTLS(addr string, auth smtp.Auth, to, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.host})
	if err != nil {
		return fmt.Errorf("smtp implicit TLS dial: %w", err)
	}
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()
	return m.deliver(client, auth, to, msg)
}

func (m *SMTPMailer) sendWithSTARTTLS(addr string, auth smtp.Auth, to, msg string) error {
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()
	if err := conn.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
		return fmt.Errorf("smtp STARTTLS: %w", err)
	}
	return m.deliver(conn, auth, to, msg)
}

func (m *SMTPMailer) deliver(client *smtp.Client, auth smtp.Auth, to, msg string) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}
