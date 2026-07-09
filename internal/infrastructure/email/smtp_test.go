package email

import "testing"

func TestSMTPMailer(t *testing.T) {
	mailer := NewSMTPMailer("localhost", 587, "", "", false, "test@example.com", "Test", "http://localhost:3000")
	if mailer == nil {
		t.Fatal("NewSMTPMailer returned nil")
	}
	// Note: We can't actually send emails in tests without a real SMTP server
	// but we can verify the struct is properly initialized
	if mailer.host != "localhost" {
		t.Errorf("expected host 'localhost', got '%s'", mailer.host)
	}
	if mailer.port != 587 {
		t.Errorf("expected port 587, got %d", mailer.port)
	}
	if mailer.frontendURL != "http://localhost:3000" {
		t.Errorf("expected frontendURL 'http://localhost:3000', got '%s'", mailer.frontendURL)
	}
}
