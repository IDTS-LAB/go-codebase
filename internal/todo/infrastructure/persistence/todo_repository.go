package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/todo/infrastructure/persistence/sqlc"
	"github.com/google/uuid"
)

type todoRepository struct {
	db *sql.DB
}

func NewTodoRepository(db *sql.DB) repository.TodoRepository {
	return &todoRepository{db: db}
}

func (r *todoRepository) Create(ctx context.Context, todo *entity.Todo) error {
	q := sqlc.New(r.db)
	err := q.CreateTodo(ctx, sqlc.CreateTodoParams{
		ID:          todo.ID,
		Title:       todo.Title,
		Description: todo.Description,
		Completed:   todo.Completed,
		CreatedAt:   todo.CreatedAt,
		UpdatedAt:   todo.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("insert todo: %w", err)
	}
	return nil
}

func (r *todoRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Todo, error) {
	q := sqlc.New(r.db)
	row, err := q.GetTodoByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("todo not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get todo: %w", err)
	}
	return mapSqlcTodoToEntity(row), nil
}

func (r *todoRepository) GetAll(ctx context.Context, offset, limit int) ([]*entity.Todo, int, error) {
	q := sqlc.New(r.db)

	total, err := q.CountTodos(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count todos: %w", err)
	}

	rows, err := q.ListTodos(ctx, sqlc.ListTodosParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("query todos: %w", err)
	}

	todos := make([]*entity.Todo, len(rows))
	for i, row := range rows {
		todos[i] = mapSqlcTodoToEntity(row)
	}
	return todos, int(total), nil
}

func (r *todoRepository) Update(ctx context.Context, todo *entity.Todo) error {
	q := sqlc.New(r.db)
	rows, err := q.UpdateTodo(ctx, sqlc.UpdateTodoParams{
		ID:          todo.ID,
		Title:       todo.Title,
		Description: todo.Description,
		Completed:   todo.Completed,
		UpdatedAt:   todo.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("update todo: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("todo not found")
	}
	return nil
}

func (r *todoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := sqlc.New(r.db)
	rows, err := q.SoftDeleteTodo(ctx, id)
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("todo not found")
	}
	return nil
}

func (r *todoRepository) Search(ctx context.Context, query string, offset, limit int) ([]*entity.Todo, int, error) {
	q := sqlc.New(r.db)
	searchPattern := sql.NullString{String: "%" + query + "%", Valid: true}

	total, err := q.CountSearchTodos(ctx, searchPattern)
	if err != nil {
		return nil, 0, fmt.Errorf("count search results: %w", err)
	}

	rows, err := q.SearchTodos(ctx, sqlc.SearchTodosParams{
		Column1: searchPattern,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("search todos: %w", err)
	}

	todos := make([]*entity.Todo, len(rows))
	for i, row := range rows {
		todos[i] = mapSqlcTodoToEntity(row)
	}
	return todos, int(total), nil
}

func mapSqlcTodoToEntity(row sqlc.Todo) *entity.Todo {
	return &entity.Todo{
		Entity: domain.Entity{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: nullTimeToPtr(row.DeletedAt),
		},
		Title:       row.Title,
		Description: row.Description,
		Completed:   row.Completed,
	}
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}
