package domain

import (
	"time"

	"github.com/google/uuid"
)

type Entity struct {
	ID        uuid.UUID  `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func NewEntity() Entity {
	now := time.Now().UTC()
	return Entity{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (e *Entity) Touch() {
	e.UpdatedAt = time.Now().UTC()
}

func (e *Entity) SoftDelete() {
	now := time.Now().UTC()
	e.DeletedAt = &now
	e.UpdatedAt = now
}

func (e *Entity) IsDeleted() bool {
	return e.DeletedAt != nil
}

func (e *Entity) Equals(other *Entity) bool {
	if e == nil || other == nil {
		return false
	}
	return e.ID == other.ID
}
