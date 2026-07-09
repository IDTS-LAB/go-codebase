package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	useTLS   bool
	from     string
	fromName string
}

func NewSMTPMailer(host string, port int, username, password string, useTLS bool, from, fromName string) *SMTPMailer {
	return &SMTPMailer{
		host: host, port: port, username: username,
		password: password, useTLS: useTLS, from: from, fromName: fromName,
	}
}

func (m *SMTPMailer) SendVerification(to, name, token string) error {
	subject := "Verify your email address"
	body := fmt.Sprintf("Hello %s,\n\nPlease verify your email by clicking the link below:\n\n%s/verify-email?token=%s\n\nIf you didn't create an account, please ignore this email.", name, token, token)
	return m.send(to, subject, body)
}

func (m *SMTPMailer) SendPasswordReset(to, name, token string) error {
	subject := "Reset your password"
	body := fmt.Sprintf("Hello %s,\n\nYou requested a password reset. Click the link below to reset your password:\n\n%s/reset-password?token=%s\n\nThis link expires in 1 hour. If you didn't request this, please ignore this email.", name, token, token)
	return m.send(to, subject, body)
}

func (m *SMTPMailer) SendWelcome(to, name string) error {
	subject := fmt.Sprintf("Welcome %s!", name)
	body := fmt.Sprintf("Hello %s,\n\nWelcome to our platform! Your account is now active.", name)
	return m.send(to, subject, body)
}

func (m *SMTPMailer) SendInvite(to, name, inviterName string) error {
	subject := fmt.Sprintf("%s invited you to join", inviterName)
	body := fmt.Sprintf("Hello %s,\n\n%s has invited you to join our platform.\n\nClick the link below to get started.", name, inviterName)
	return m.send(to, subject, body)
}

func (m *SMTPMailer) send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	auth := smtp.PlainAuth("", m.username, m.password, m.host)

	msg := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		m.fromName, m.from, to, subject, body)

	if m.useTLS {
		return m.sendWithTLS(addr, auth, to, msg)
	}
	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}

func (m *SMTPMailer) sendWithTLS(addr string, auth smtp.Auth, to, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.host})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return err
	}
	defer client.Close()
	if err = client.Auth(auth); err != nil {
		return err
	}
	if err = client.Mail(m.from); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write([]byte(msg)); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return client.Quit()
}
