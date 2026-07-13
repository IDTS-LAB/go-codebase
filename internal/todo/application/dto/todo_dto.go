package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateTodoRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=1000"`
}

type UpdateTodoRequest struct {
	Title       string `json:"title" validate:"omitempty,min=1,max=255"`
	Description string `json:"description" validate:"omitempty,max=1000"`
}

type TodoResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TodoListResponse struct {
	Todos []TodoResponse `json:"todos"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}
