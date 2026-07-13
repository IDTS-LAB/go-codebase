package public

import (
	"context"

	"github.com/google/uuid"
)

type AuthorizationProvider interface {
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
}
