package entity

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTenant_Creation(t *testing.T) {
	settings := json.RawMessage(`{"theme": "dark"}`)
	now := time.Now().UTC().Truncate(time.Second)

	tenant := Tenant{
		ID:        uuid.New(),
		Name:      "Acme Corp",
		Slug:      "acme-corp",
		Settings:  settings,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.NotEqual(t, uuid.Nil, tenant.ID)
	assert.Equal(t, "Acme Corp", tenant.Name)
	assert.Equal(t, "acme-corp", tenant.Slug)
	assert.Nil(t, tenant.Domain)
	assert.Equal(t, settings, tenant.Settings)
	assert.True(t, tenant.IsActive)
	assert.Equal(t, now, tenant.CreatedAt)
	assert.Equal(t, now, tenant.UpdatedAt)
}

func TestTenant_WithDomain(t *testing.T) {
	domain := "acme.com"
	tenant := Tenant{
		ID:        uuid.New(),
		Name:      "Acme Corp",
		Slug:      "acme-corp",
		Domain:    &domain,
		Settings:  json.RawMessage(`{}`),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	assert.NotNil(t, tenant.Domain)
	assert.Equal(t, "acme.com", *tenant.Domain)
}

func TestTenant_Inactive(t *testing.T) {
	tenant := Tenant{
		ID:        uuid.New(),
		Name:      "Inactive Corp",
		Slug:      "inactive-corp",
		Settings:  json.RawMessage(`{}`),
		IsActive:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	assert.False(t, tenant.IsActive)
}
