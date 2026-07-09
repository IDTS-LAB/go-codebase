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
