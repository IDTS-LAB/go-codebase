package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPermission(t *testing.T) {
	perm := NewPermission("read:users", "Can read users", "users", "read")

	assert.NotEqual(t, "", perm.ID.String())
	assert.Equal(t, "read:users", perm.Name)
	assert.Equal(t, "Can read users", perm.Description)
	assert.Equal(t, "users", perm.Resource)
	assert.Equal(t, "read", perm.Action)
}
