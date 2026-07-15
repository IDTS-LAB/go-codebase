package specification

import (
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestNotCompletedSpec(t *testing.T) {
	spec := NotCompletedSpec{}

	t.Run("not completed returns true", func(t *testing.T) {
		todo := entity.NewTodo("Task", "")
		assert.True(t, spec.IsSatisfiedBy(todo))
	})

	t.Run("completed returns false", func(t *testing.T) {
		todo := entity.NewTodo("Task", "")
		todo.Complete()
		assert.False(t, spec.IsSatisfiedBy(todo))
	})
}

func TestCompletedSpec(t *testing.T) {
	spec := CompletedSpec{}

	t.Run("completed returns true", func(t *testing.T) {
		todo := entity.NewTodo("Task", "")
		todo.Complete()
		assert.True(t, spec.IsSatisfiedBy(todo))
	})

	t.Run("not completed returns false", func(t *testing.T) {
		todo := entity.NewTodo("Task", "")
		assert.False(t, spec.IsSatisfiedBy(todo))
	})
}

func TestTitleContainsSpec(t *testing.T) {
	t.Run("empty substring returns true", func(t *testing.T) {
		spec := TitleContainsSpec{Substring: ""}
		todo := entity.NewTodo("Task", "")
		assert.True(t, spec.IsSatisfiedBy(todo))
	})

	t.Run("non-empty substring", func(t *testing.T) {
		spec := TitleContainsSpec{Substring: "Task"}
		todo := entity.NewTodo("Task", "")
		assert.True(t, spec.IsSatisfiedBy(todo))
	})
}
