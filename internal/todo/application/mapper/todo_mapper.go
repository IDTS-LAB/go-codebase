package mapper

import (
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
)

func ToTodoResponse(todo *entity.Todo) dto.TodoResponse {
	return dto.TodoResponse{
		ID:          todo.ID,
		Title:       todo.Title,
		Description: todo.Description,
		Completed:   todo.Completed,
		CreatedAt:   todo.CreatedAt,
		UpdatedAt:   todo.UpdatedAt,
	}
}

func ToTodoListResponse(todos []*entity.Todo, total, page, limit int) dto.TodoListResponse {
	responses := make([]dto.TodoResponse, len(todos))
	for i, todo := range todos {
		responses[i] = ToTodoResponse(todo)
	}
	return dto.TodoListResponse{
		Todos: responses,
		Total: total,
		Page:  page,
		Limit: limit,
	}
}
