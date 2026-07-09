package email

import (
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"go.uber.org/fx"
)

var Module = fx.Module("email", fx.Provide(NewEmailer))

func NewEmailer(cfg *config.Config) domain.Emailer {
	switch cfg.Email.Provider {
	default:
		return NewConsoleMailer(cfg.Email.From, cfg.Email.FromName)
	}
}
