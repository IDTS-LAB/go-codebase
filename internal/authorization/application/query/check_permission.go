package query

import (
	"context"

	"github.com/google/uuid"
)

type CheckPermissionQuery struct {
	UserID   uuid.UUID
	Resource string
	Action   string
}

type CheckPermissionHandler struct {
	enforcer Enforcer
}

type Enforcer interface {
	Enforce(userID uuid.UUID, resource, action string) (bool, error)
}

func NewCheckPermissionHandler(enforcer Enforcer) *CheckPermissionHandler {
	return &CheckPermissionHandler{enforcer: enforcer}
}

func (h *CheckPermissionHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(CheckPermissionQuery)
	return h.enforcer.Enforce(q.UserID, q.Resource, q.Action)
}
