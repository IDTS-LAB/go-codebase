package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewEntity(t *testing.T) {
	e := NewEntity()

	assert.NotEqual(t, uuid.Nil, e.ID)
	assert.False(t, e.CreatedAt.IsZero())
	assert.False(t, e.UpdatedAt.IsZero())
	assert.True(t, e.CreatedAt.Equal(e.UpdatedAt))
	assert.Nil(t, e.DeletedAt)
}

func TestTouch(t *testing.T) {
	e := NewEntity()
	original := e.UpdatedAt

	time.Sleep(time.Millisecond)
	e.Touch()

	assert.True(t, e.UpdatedAt.After(original))
}

func TestSoftDelete(t *testing.T) {
	e := NewEntity()
	assert.Nil(t, e.DeletedAt)
	assert.False(t, e.IsDeleted())

	e.SoftDelete()

	assert.NotNil(t, e.DeletedAt)
	assert.True(t, e.IsDeleted())
	assert.True(t, e.UpdatedAt.Equal(*e.DeletedAt))
}

func TestIsDeleted(t *testing.T) {
	t.Run("not deleted", func(t *testing.T) {
		e := NewEntity()
		assert.False(t, e.IsDeleted())
	})

	t.Run("deleted", func(t *testing.T) {
		e := NewEntity()
		e.SoftDelete()
		assert.True(t, e.IsDeleted())
	})
}

func TestEquals(t *testing.T) {
	id := uuid.New()

	t.Run("same ID", func(t *testing.T) {
		a := Entity{ID: id}
		b := Entity{ID: id}
		assert.True(t, a.Equals(&b))
	})

	t.Run("different ID", func(t *testing.T) {
		a := Entity{ID: uuid.New()}
		b := Entity{ID: uuid.New()}
		assert.False(t, a.Equals(&b))
	})

	t.Run("nil receiver", func(t *testing.T) {
		var a *Entity
		b := Entity{ID: uuid.New()}
		assert.False(t, a.Equals(&b))
	})

	t.Run("nil other", func(t *testing.T) {
		a := Entity{ID: uuid.New()}
		assert.False(t, a.Equals(nil))
	})

	t.Run("both nil", func(t *testing.T) {
		var a *Entity
		assert.False(t, a.Equals(nil))
	})
}
