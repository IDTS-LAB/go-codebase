package email

import (
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"go.uber.org/fx"
)

var Module = fx.Module("email", fx.Provide(NewEmailer))

func NewEmailer(cfg *config.Config) domain.Emailer {
	switch cfg.Email.Provider {
	case "smtp":
		return NewSMTPMailer(cfg.Email.SMTP.Host, cfg.Email.SMTP.Port,
			cfg.Email.SMTP.Username, cfg.Email.SMTP.Password,
			cfg.Email.SMTP.UseTLS, cfg.Email.From, cfg.Email.FromName)
	case "sendgrid":
		return NewSendGridMailer(cfg.Email.SendGrid.APIKey,
			cfg.Email.From, cfg.Email.FromName)
	default:
		return NewConsoleMailer(cfg.Email.From, cfg.Email.FromName)
	}
}

// NewSMTPMailer is a placeholder for SMTP implementation (Task 3)
func NewSMTPMailer(host string, port int, username, password string, useTLS bool, from, fromName string) domain.Emailer {
	return NewConsoleMailer(from, fromName)
}

// NewSendGridMailer is a placeholder for SendGrid implementation (Task 4)
func NewSendGridMailer(apiKey, from, fromName string) domain.Emailer {
	return NewConsoleMailer(from, fromName)
}