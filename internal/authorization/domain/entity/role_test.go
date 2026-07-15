package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRole(t *testing.T) {
	role := NewRole("admin", "Administrator role")

	assert.NotEqual(t, "", role.ID.String())
	assert.Equal(t, "admin", role.Name)
	assert.Equal(t, "Administrator role", role.Description)
}

func TestNewRole_Defaults(t *testing.T) {
	role := NewRole("user", "")

	assert.Equal(t, "user", role.Name)
	assert.Empty(t, role.Description)
}
