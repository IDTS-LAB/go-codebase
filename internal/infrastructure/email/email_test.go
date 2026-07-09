package email

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
)

func TestConsoleMailer(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr) // restore

	mailer := NewConsoleMailer("test@example.com", "Test App", "http://localhost:3000")

	if err := mailer.SendVerification("user@test.com", "Test User", "abc123"); err != nil {
		t.Errorf("SendVerification failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "user@test.com") {
		t.Error("SendVerification should log recipient email")
	}
	if !strings.Contains(output, "abc123") {
		t.Error("SendVerification should log the token")
	}

	buf.Reset()
	if err := mailer.SendPasswordReset("user@test.com", "Test User", "xyz789"); err != nil {
		t.Errorf("SendPasswordReset failed: %v", err)
	}
	output = buf.String()
	if !strings.Contains(output, "user@test.com") {
		t.Error("SendPasswordReset should log recipient email")
	}
	if !strings.Contains(output, "xyz789") {
		t.Error("SendPasswordReset should log the token")
	}

	buf.Reset()
	if err := mailer.SendWelcome("user@test.com", "Test User"); err != nil {
		t.Errorf("SendWelcome failed: %v", err)
	}
	output = buf.String()
	if !strings.Contains(output, "user@test.com") {
		t.Error("SendWelcome should log recipient email")
	}
	if !strings.Contains(output, "Test User") {
		t.Error("SendWelcome should log the name")
	}

	buf.Reset()
	if err := mailer.SendInvite("user@test.com", "Test User", "Admin"); err != nil {
		t.Errorf("SendInvite failed: %v", err)
	}
	output = buf.String()
	if !strings.Contains(output, "user@test.com") {
		t.Error("SendInvite should log recipient email")
	}
	if !strings.Contains(output, "Admin") {
		t.Error("SendInvite should log the inviter name")
	}
}

func TestNewEmailer(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantType string
	}{
		{"console", "console", "*email.ConsoleMailer"},
		{"smtp", "smtp", "*email.SMTPMailer"},
		{"sendgrid", "sendgrid", "*email.SendGridMailer"},
		{"unknown defaults to console", "", "*email.ConsoleMailer"},
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
			cfg.Email.SendGrid.APIKey = "test-key"

			mailer := NewEmailer(cfg)
			if mailer == nil {
				t.Fatal("NewEmailer returned nil")
			}

			gotType := fmt.Sprintf("%T", mailer)
			if gotType != tt.wantType {
				t.Errorf("provider %q: got %s, want %s", tt.provider, gotType, tt.wantType)
			}
		})
	}
}
