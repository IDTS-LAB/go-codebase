package entity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewTodo(t *testing.T) {
	todo := NewTodo("Buy milk", "2% milk")

	assert.NotEqual(t, uuid.Nil, todo.ID)
	assert.Equal(t, "Buy milk", todo.Title)
	assert.Equal(t, "2% milk", todo.Description)
	assert.False(t, todo.Completed)
}

func TestNewTodo_EmptyDescription(t *testing.T) {
	todo := NewTodo("Buy milk", "")

	assert.Equal(t, "Buy milk", todo.Title)
	assert.Empty(t, todo.Description)
}

func TestComplete(t *testing.T) {
	todo := NewTodo("Task", "")
	todo.Complete()

	assert.True(t, todo.Completed)
	assert.True(t, todo.UpdatedAt.After(todo.CreatedAt))
}

func TestUpdate(t *testing.T) {
	todo := NewTodo("Old Title", "Old Desc")

	todo.Update("New Title", "New Desc")

	assert.Equal(t, "New Title", todo.Title)
	assert.Equal(t, "New Desc", todo.Description)
}

func TestUpdate_Partial(t *testing.T) {
	t.Run("empty title keeps old", func(t *testing.T) {
		todo := NewTodo("Old Title", "Old Desc")
		todo.Update("", "New Desc")

		assert.Equal(t, "Old Title", todo.Title)
		assert.Equal(t, "New Desc", todo.Description)
	})

	t.Run("empty description keeps old", func(t *testing.T) {
		todo := NewTodo("Old Title", "Old Desc")
		todo.Update("New Title", "")

		assert.Equal(t, "New Title", todo.Title)
		assert.Equal(t, "Old Desc", todo.Description)
	})
}

func TestIDString(t *testing.T) {
	todo := NewTodo("Task", "")

	assert.Equal(t, todo.ID.String(), todo.IDString())
}

func TestTodoIDFromString(t *testing.T) {
	t.Run("valid string", func(t *testing.T) {
		id := uuid.New()
		parsed, err := TodoIDFromString(id.String())

		assert.NoError(t, err)
		assert.Equal(t, id, parsed)
	})

	t.Run("invalid string", func(t *testing.T) {
		_, err := TodoIDFromString("invalid")

		assert.Error(t, err)
	})
}
