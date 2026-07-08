package entity

import (
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type Todo struct {
	domain.Entity
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

func NewTodo(title, description string) *Todo {
	return &Todo{
		Entity:      domain.NewEntity(),
		Title:       title,
		Description: description,
		Completed:   false,
	}
}

func (t *Todo) Complete() {
	t.Completed = true
	t.Touch()
}

func (t *Todo) Update(title, description string) {
	if title != "" {
		t.Title = title
	}
	if description != "" {
		t.Description = description
	}
	t.Touch()
}

func (t *Todo) IDString() string {
	return t.Entity.ID.String()
}

func TodoIDFromString(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
