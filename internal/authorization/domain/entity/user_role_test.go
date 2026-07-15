package entity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUserRole(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	ur := NewUserRole(userID, roleID)

	assert.Equal(t, userID, ur.UserID)
	assert.Equal(t, roleID, ur.RoleID)
}
