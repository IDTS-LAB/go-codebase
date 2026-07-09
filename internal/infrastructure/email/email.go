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
		return NewSMTPMailer(
			cfg.Email.SMTP.Host,
			cfg.Email.SMTP.Port,
			cfg.Email.SMTP.Username,
			cfg.Email.SMTP.Password,
			cfg.Email.SMTP.UseTLS,
			cfg.Email.From,
			cfg.Email.FromName,
			cfg.Email.FrontendURL,
		)
	case "sendgrid":
		return NewSendGridMailer(
			cfg.Email.SendGrid.APIKey,
			cfg.Email.From,
			cfg.Email.FromName,
			cfg.Email.FrontendURL,
		)
	default:
		return NewConsoleMailer(cfg.Email.From, cfg.Email.FromName, cfg.Email.FrontendURL)
	}
}
