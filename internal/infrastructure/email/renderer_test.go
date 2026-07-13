package email

import (
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		data        TemplateData
		wantErr     bool
		contains    []string
		notContains []string
	}{
		{
			name:     "verification template renders name and verify URL",
			template: "verification",
			data: TemplateData{
				Name:      "Test User",
				VerifyURL: "http://localhost:3000/verify-email?token=abc123",
			},
			contains: []string{"Test User", "abc123", "Verify Your Email"},
		},
		{
			name:     "password reset template renders name and reset URL",
			template: "password_reset",
			data: TemplateData{
				Name:     "Test User",
				ResetURL: "http://localhost:3000/reset-password?token=xyz789",
			},
			contains: []string{"Test User", "xyz789", "Reset Your Password"},
		},
		{
			name:     "welcome template renders name",
			template: "welcome",
			data: TemplateData{
				Name: "Test User",
			},
			contains: []string{"Test User", "Welcome"},
		},
		{
			name:     "invite template renders name, inviter, and invite URL",
			template: "invite",
			data: TemplateData{
				Name:        "Test User",
				InviterName: "Admin",
				InviteURL:   "http://localhost:3000/invite?token=def456",
			},
			contains: []string{"Test User", "Admin", "def456", "You've Been Invited"},
		},
		{
			name:     "verification template with empty data still renders",
			template: "verification",
			data:     TemplateData{},
			contains: []string{"Verify Your Email"},
		},
		{
			name:     "nonexistent template returns error",
			template: "nonexistent",
			data:     TemplateData{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := renderTemplate(tt.template, tt.data)
			if tt.wantErr {
				if err == nil {
					t.Errorf("renderTemplate(%q) expected error, got nil", tt.template)
				}
				return
			}
			if err != nil {
				t.Fatalf("renderTemplate(%q) failed: %v", tt.template, err)
			}
			if content == "" {
				t.Fatalf("renderTemplate(%q) returned empty content", tt.template)
			}
			for _, s := range tt.contains {
				if !strings.Contains(content, s) {
					t.Errorf("renderTemplate(%q) missing %q", tt.template, s)
				}
			}
			for _, s := range tt.notContains {
				if strings.Contains(content, s) {
					t.Errorf("renderTemplate(%q) should not contain %q", tt.template, s)
				}
			}
		})
	}
}

func TestRenderTemplateOutputIsValidHTML(t *testing.T) {
	content, err := renderTemplate("verification", TemplateData{
		Name:      "Test",
		VerifyURL: "http://example.com/verify",
	})
	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}
	if !strings.HasPrefix(content, "<!DOCTYPE html>") {
		t.Error("verification template does not start with <!DOCTYPE html>")
	}
	if !strings.HasSuffix(strings.TrimSpace(content), "</html>") {
		t.Error("verification template does not end with </html>")
	}
}
