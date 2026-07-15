package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewRefreshToken(t *testing.T) {
	userID := uuid.New()
	expiresAt := time.Now().Add(time.Hour)

	token := NewRefreshToken(userID, "refresh-token-value", expiresAt)

	assert.NotEqual(t, uuid.Nil, token.ID)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, "refresh-token-value", token.Token)
	assert.Equal(t, expiresAt, token.ExpiresAt)
	assert.Nil(t, token.RevokedAt)
}

func TestRefreshTokenIsExpired(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		token := NewRefreshToken(uuid.New(), "token", time.Now().Add(time.Hour))
		assert.False(t, token.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		token := NewRefreshToken(uuid.New(), "token", time.Now().Add(-time.Hour))
		assert.True(t, token.IsExpired())
	})
}

func TestRefreshTokenIsRevoked(t *testing.T) {
	token := NewRefreshToken(uuid.New(), "token", time.Now().Add(time.Hour))

	assert.False(t, token.IsRevoked())

	token.Revoke()
	assert.True(t, token.IsRevoked())
}

func TestRefreshTokenRevoke(t *testing.T) {
	token := NewRefreshToken(uuid.New(), "token", time.Now().Add(time.Hour))

	token.Revoke()

	assert.NotNil(t, token.RevokedAt)
	assert.True(t, token.UpdatedAt.Equal(*token.RevokedAt) || token.UpdatedAt.After(*token.RevokedAt))
}
