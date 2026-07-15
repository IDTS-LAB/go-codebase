package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type testEvent struct {
	BaseEvent
}

func TestNewAggregateRoot(t *testing.T) {
	a := NewAggregateRoot()

	assert.NotEqual(t, uuid.Nil, a.ID)
	assert.False(t, a.CreatedAt.IsZero())
	assert.False(t, a.HasEvents())
}

func TestRecordEvent(t *testing.T) {
	a := NewAggregateRoot()
	original := a.UpdatedAt

	time.Sleep(time.Millisecond)
	a.RecordEvent(testEvent{})

	assert.True(t, a.HasEvents())
	assert.Len(t, a.Events(), 1)
	assert.True(t, a.UpdatedAt.After(original))
}

func TestPullEvents(t *testing.T) {
	a := NewAggregateRoot()
	a.RecordEvent(testEvent{})

	events := a.PullEvents()

	assert.Len(t, events, 1)
	assert.False(t, a.HasEvents())
	assert.Empty(t, a.Events())
}

func TestClearEvents(t *testing.T) {
	a := NewAggregateRoot()
	a.RecordEvent(testEvent{})

	a.ClearEvents()

	assert.False(t, a.HasEvents())
	assert.Empty(t, a.Events())
}

func TestHasEvents(t *testing.T) {
	t.Run("no events", func(t *testing.T) {
		a := NewAggregateRoot()
		assert.False(t, a.HasEvents())
	})

	t.Run("with events", func(t *testing.T) {
		a := NewAggregateRoot()
		a.RecordEvent(testEvent{})
		assert.True(t, a.HasEvents())
	})

	t.Run("after pull", func(t *testing.T) {
		a := NewAggregateRoot()
		a.RecordEvent(testEvent{})
		a.PullEvents()
		assert.False(t, a.HasEvents())
	})
}

func TestTouchWith(t *testing.T) {
	a := NewAggregateRoot()
	expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	a.TouchWith(expected)

	assert.Equal(t, expected, a.UpdatedAt)
}
