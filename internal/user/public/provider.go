package public

import (
	"context"

	"github.com/google/uuid"
)

type UserProfile struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	IsActive  bool     `json:"is_active"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type UserProfileProvider interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)
}
