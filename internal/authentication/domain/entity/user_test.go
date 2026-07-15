package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewUser(t *testing.T) {
	user := NewUser("test@example.com", "hashed_password", "Test User")

	assert.NotEqual(t, "", user.ID.String())
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "hashed_password", user.Password)
	assert.Equal(t, "Test User", user.Name)
	assert.True(t, user.IsActive)
	assert.Equal(t, 0, user.FailedLoginAttempts)
	assert.Nil(t, user.LockedUntil)
	assert.False(t, user.EmailVerified)
}

func TestUserIsLocked(t *testing.T) {
	t.Run("not locked", func(t *testing.T) {
		user := NewUser("test@example.com", "pwd", "User")
		assert.False(t, user.IsLocked())
	})

	t.Run("locked until future", func(t *testing.T) {
		user := NewUser("test@example.com", "pwd", "User")
		user.Lock(time.Hour)
		assert.True(t, user.IsLocked())
	})

	t.Run("lock expired", func(t *testing.T) {
		user := NewUser("test@example.com", "pwd", "User")
		user.Lock(-time.Hour)
		assert.False(t, user.IsLocked())
	})
}

func TestUserLock(t *testing.T) {
	user := NewUser("test@example.com", "pwd", "User")

	user.Lock(30 * time.Minute)

	assert.Equal(t, 1, user.FailedLoginAttempts)
	assert.NotNil(t, user.LockedUntil)
	assert.True(t, time.Now().Before(*user.LockedUntil))

	user.Lock(30 * time.Minute)
	assert.Equal(t, 2, user.FailedLoginAttempts)
}

func TestUserUnlock(t *testing.T) {
	user := NewUser("test@example.com", "pwd", "User")
	user.Lock(time.Hour)
	assert.True(t, user.IsLocked())

	user.Unlock()

	assert.Equal(t, 0, user.FailedLoginAttempts)
	assert.Nil(t, user.LockedUntil)
	assert.False(t, user.IsLocked())
}

func TestUserEmailVerificationFields(t *testing.T) {
	user := NewUser("test@example.com", "hashed_password", "Test User")

	assert.False(t, user.EmailVerified)
	assert.Nil(t, user.EmailVerifyToken)
	assert.Nil(t, user.EmailVerifyExpires)
	assert.Nil(t, user.PasswordResetToken)
	assert.Nil(t, user.PasswordResetExpires)

	token := "verify-token-123"
	expires := time.Now().Add(24 * time.Hour)
	user.EmailVerifyToken = &token
	user.EmailVerifyExpires = &expires
	user.EmailVerified = true

	assert.Equal(t, "verify-token-123", *user.EmailVerifyToken)
	assert.Equal(t, expires, *user.EmailVerifyExpires)
	assert.True(t, user.EmailVerified)

	user.EmailVerifyToken = nil
	user.EmailVerifyExpires = nil
	assert.Nil(t, user.EmailVerifyToken)
	assert.Nil(t, user.EmailVerifyExpires)
}
