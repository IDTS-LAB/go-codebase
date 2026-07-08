package event

import (
	"time"

	"github.com/google/uuid"
)

type TodoCreated struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type TodoUpdated struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TodoCompleted struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TodoDeleted struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

const (
	TodoCreatedEvent   = "todo.created"
	TodoUpdatedEvent   = "todo.updated"
	TodoCompletedEvent = "todo.completed"
	TodoDeletedEvent   = "todo.deleted"
)
