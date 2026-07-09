package email

import (
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
)

func TestConsoleMailer(t *testing.T) {
	mailer := NewConsoleMailer("test@example.com", "Test App", "http://localhost:3000")

	if err := mailer.SendVerification("user@test.com", "Test User", "abc123"); err != nil {
		t.Errorf("SendVerification failed: %v", err)
	}
	if err := mailer.SendPasswordReset("user@test.com", "Test User", "xyz789"); err != nil {
		t.Errorf("SendPasswordReset failed: %v", err)
	}
	if err := mailer.SendWelcome("user@test.com", "Test User"); err != nil {
		t.Errorf("SendWelcome failed: %v", err)
	}
	if err := mailer.SendInvite("user@test.com", "Test User", "Admin"); err != nil {
		t.Errorf("SendInvite failed: %v", err)
	}
}

func TestNewEmailer(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{name: "console provider (default)", provider: "console"},
		{name: "smtp provider", provider: "smtp"},
		{name: "sendgrid provider", provider: "sendgrid"},
		{name: "unknown provider falls back to console", provider: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Email.Provider = tt.provider
			cfg.Email.From = "test@example.com"
			cfg.Email.FromName = "Test"
			cfg.Email.FrontendURL = "http://localhost:3000"
			cfg.Email.SMTP.Host = "localhost"
			cfg.Email.SMTP.Port = 587
			cfg.Email.SendGrid.APIKey = "test-api-key"

			mailer := NewEmailer(cfg)
			if mailer == nil {
				t.Fatalf("NewEmailer(%q) returned nil", tt.provider)
			}
		})
	}
}
