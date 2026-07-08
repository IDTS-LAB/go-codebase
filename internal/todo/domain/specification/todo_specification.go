package specification

import (
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
)

type Specification interface {
	IsSatisfiedBy(todo *entity.Todo) bool
}

type NotCompletedSpec struct{}

func (s NotCompletedSpec) IsSatisfiedBy(todo *entity.Todo) bool {
	return !todo.Completed
}

type CompletedSpec struct{}

func (s CompletedSpec) IsSatisfiedBy(todo *entity.Todo) bool {
	return todo.Completed
}

type TitleContainsSpec struct {
	Substring string
}

func (s TitleContainsSpec) IsSatisfiedBy(todo *entity.Todo) bool {
	if s.Substring == "" {
		return true
	}
	return len(todo.Title) > 0
}
