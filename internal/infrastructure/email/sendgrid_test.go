package email

import "testing"

func TestSendGridMailer(t *testing.T) {
	mailer := NewSendGridMailer("test-api-key", "test@example.com", "Test", "http://localhost:3000")
	if mailer == nil {
		t.Fatal("NewSendGridMailer returned nil")
	}
	if mailer.client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if mailer.from != "test@example.com" {
		t.Errorf("expected from 'test@example.com', got '%s'", mailer.from)
	}
	if mailer.fromName != "Test" {
		t.Errorf("expected fromName 'Test', got '%s'", mailer.fromName)
	}
	if mailer.frontendURL != "http://localhost:3000" {
		t.Errorf("expected frontendURL 'http://localhost:3000', got '%s'", mailer.frontendURL)
	}
}
