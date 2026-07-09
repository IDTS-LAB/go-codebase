package entity

import (
	"testing"
	"time"
)

func TestUserEmailVerificationFields(t *testing.T) {
	user := NewUser("test@example.com", "hashed_password", "Test User")

	// New fields should default to zero values
	if user.EmailVerified {
		t.Error("new user should not be email verified")
	}
	if user.EmailVerifyToken != nil {
		t.Error("new user should not have verify token")
	}
	if user.EmailVerifyExpires != nil {
		t.Error("new user should not have verify expires")
	}
	if user.PasswordResetToken != nil {
		t.Error("new user should not have reset token")
	}
	if user.PasswordResetExpires != nil {
		t.Error("new user should not have reset expires")
	}

	// Test setting verification fields
	token := "verify-token-123"
	expires := time.Now().Add(24 * time.Hour)
	user.EmailVerifyToken = &token
	user.EmailVerifyExpires = &expires
	user.EmailVerified = true

	if user.EmailVerifyToken == nil || *user.EmailVerifyToken != "verify-token-123" {
		t.Error("verify token not set correctly")
	}
	if user.EmailVerifyExpires == nil || !user.EmailVerifyExpires.Equal(expires) {
		t.Error("verify expires not set correctly")
	}
	if !user.EmailVerified {
		t.Error("email_verified not set correctly")
	}

	// Test clearing verification fields
	user.EmailVerifyToken = nil
	user.EmailVerifyExpires = nil
	if user.EmailVerifyToken != nil {
		t.Error("verify token should be nil after clearing")
	}
}
