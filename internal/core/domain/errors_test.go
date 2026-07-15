package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainError(t *testing.T) {
	err := NewDomainError(ErrNotFound, "NOT_FOUND", "entity was not found")

	assert.Equal(t, "entity was not found", err.Error())
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, "NOT_FOUND", err.Code)
}

func TestDomainErrorMessage(t *testing.T) {
	t.Run("with message", func(t *testing.T) {
		err := NewDomainError(ErrNotFound, "NOT_FOUND", "custom message")
		assert.Equal(t, "custom message", err.Error())
	})

	t.Run("without message", func(t *testing.T) {
		err := &DomainError{Err: ErrNotFound, Code: "NOT_FOUND"}
		assert.Equal(t, ErrNotFound.Error(), err.Error())
	})
}

func TestIsNotFound(t *testing.T) {
	t.Run("not found error", func(t *testing.T) {
		assert.True(t, IsNotFound(ErrNotFound))
	})

	t.Run("wrapped not found", func(t *testing.T) {
		err := NewDomainError(ErrNotFound, "NOT_FOUND", "")
		assert.True(t, IsNotFound(err))
	})

	t.Run("other error", func(t *testing.T) {
		assert.False(t, IsNotFound(errors.New("something else")))
	})
}

func TestIsConflict(t *testing.T) {
	t.Run("conflict error", func(t *testing.T) {
		assert.True(t, IsConflict(ErrConflict))
	})

	t.Run("wrapped conflict", func(t *testing.T) {
		err := NewDomainError(ErrConflict, "CONFLICT", "")
		assert.True(t, IsConflict(err))
	})

	t.Run("other error", func(t *testing.T) {
		assert.False(t, IsConflict(errors.New("something else")))
	})
}
