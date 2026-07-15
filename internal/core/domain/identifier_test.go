package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewIdentifier(t *testing.T) {
	id := NewIdentifier()

	assert.False(t, id.IsZero())
	assert.NotEqual(t, uuid.Nil, id.UUID())
}

func TestIdentifierFromUUID(t *testing.T) {
	uid := uuid.New()
	id := IdentifierFromUUID(uid)

	assert.Equal(t, uid, id.UUID())
}

func TestIdentifierFromString(t *testing.T) {
	t.Run("valid string", func(t *testing.T) {
		uid := uuid.New()
		id, err := IdentifierFromString(uid.String())

		assert.NoError(t, err)
		assert.Equal(t, uid, id.UUID())
	})

	t.Run("invalid string", func(t *testing.T) {
		_, err := IdentifierFromString("not-a-uuid")

		assert.Error(t, err)
	})
}

func TestIdentifierUUID(t *testing.T) {
	uid := uuid.New()
	id := IdentifierFromUUID(uid)

	assert.Equal(t, uid, id.UUID())
}

func TestIdentifierString(t *testing.T) {
	uid := uuid.New()
	id := IdentifierFromUUID(uid)

	assert.Equal(t, uid.String(), id.String())
}

func TestIdentifierIsZero(t *testing.T) {
	t.Run("nil UUID", func(t *testing.T) {
		id := Identifier{}
		assert.True(t, id.IsZero())
	})

	t.Run("non-nil UUID", func(t *testing.T) {
		id := NewIdentifier()
		assert.False(t, id.IsZero())
	})
}

func TestIdentifierEquals(t *testing.T) {
	uid := uuid.New()

	t.Run("equal", func(t *testing.T) {
		a := IdentifierFromUUID(uid)
		b := IdentifierFromUUID(uid)
		assert.True(t, a.Equals(b))
	})

	t.Run("not equal", func(t *testing.T) {
		a := IdentifierFromUUID(uuid.New())
		b := IdentifierFromUUID(uuid.New())
		assert.False(t, a.Equals(b))
	})
}

func TestIdentifierMarshalText(t *testing.T) {
	uid := uuid.New()
	id := IdentifierFromUUID(uid)

	data, err := id.MarshalText()

	assert.NoError(t, err)
	assert.Equal(t, []byte(uid.String()), data)
}

func TestIdentifierUnmarshalText(t *testing.T) {
	t.Run("invalid text", func(t *testing.T) {
		var id Identifier
		err := id.UnmarshalText([]byte("bad"))
		assert.Error(t, err)
	})
}

func TestIdentifierValue(t *testing.T) {
	uid := uuid.New()
	id := IdentifierFromUUID(uid)

	v, err := id.Value()

	assert.NoError(t, err)
	s, ok := v.(string)
	assert.True(t, ok)
	assert.Equal(t, uid.String(), s)
}

func TestIdentifierValueAndScan(t *testing.T) {
	uid := uuid.New()
	id := IdentifierFromUUID(uid)

	v, err := id.Value()
	assert.NoError(t, err)
	_, ok := v.(string)
	assert.True(t, ok)
}
