package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Slug      string          `json:"slug"`
	Domain    *string         `json:"domain,omitempty"`
	Settings  json.RawMessage `json:"settings"`
	IsActive  bool            `json:"is_active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
