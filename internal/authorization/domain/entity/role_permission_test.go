package entity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewRolePermission(t *testing.T) {
	roleID := uuid.New()
	permID := uuid.New()

	rp := NewRolePermission(roleID, permID)

	assert.Equal(t, roleID, rp.RoleID)
	assert.Equal(t, permID, rp.PermissionID)
}
