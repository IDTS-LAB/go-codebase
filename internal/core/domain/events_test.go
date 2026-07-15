package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseEventEventType(t *testing.T) {
	e := BaseEvent{Type: "test.event"}

	assert.Equal(t, "test.event", e.EventType())
}

func TestBaseEventOccurredAt(t *testing.T) {
	e := BaseEvent{Timestamp: "2024-01-01T00:00:00Z"}

	assert.Equal(t, "2024-01-01T00:00:00Z", e.OccurredAt())
}
