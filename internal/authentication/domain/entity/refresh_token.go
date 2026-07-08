package entity

import (
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type RefreshToken struct {
	domain.Entity
	UserID    uuid.UUID  `json:"user_id"`
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

func NewRefreshToken(userID uuid.UUID, token string, expiresAt time.Time) *RefreshToken {
	return &RefreshToken{
		Entity:    domain.NewEntity(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}
}

func (r *RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

func (r *RefreshToken) IsRevoked() bool {
	return r.RevokedAt != nil
}

func (r *RefreshToken) Revoke() {
	now := time.Now()
	r.RevokedAt = &now
	r.Touch()
}
