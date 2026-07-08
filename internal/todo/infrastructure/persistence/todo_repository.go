package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/repository"
	"github.com/google/uuid"
)

type todoRepository struct {
	db *sql.DB
}

func NewTodoRepository(db *sql.DB) repository.TodoRepository {
	return &todoRepository{db: db}
}

func (r *todoRepository) Create(ctx context.Context, todo *entity.Todo) error {
	query := `
		INSERT INTO todos (id, title, description, completed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		todo.ID,
		todo.Title,
		todo.Description,
		todo.Completed,
		todo.CreatedAt,
		todo.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert todo: %w", err)
	}
	return nil
}

func (r *todoRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Todo, error) {
	query := `
		SELECT id, title, description, completed, created_at, updated_at, deleted_at
		FROM todos
		WHERE id = $1 AND deleted_at IS NULL`

	todo := &entity.Todo{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&todo.ID,
		&todo.Title,
		&todo.Description,
		&todo.Completed,
		&todo.CreatedAt,
		&todo.UpdatedAt,
		&todo.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("todo not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get todo: %w", err)
	}
	return todo, nil
}

func (r *todoRepository) GetAll(ctx context.Context, offset, limit int) ([]*entity.Todo, int, error) {
	countQuery := `SELECT COUNT(*) FROM todos WHERE deleted_at IS NULL`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count todos: %w", err)
	}

	query := `
		SELECT id, title, description, completed, created_at, updated_at, deleted_at
		FROM todos
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query todos: %w", err)
	}
	defer rows.Close()

	var todos []*entity.Todo
	for rows.Next() {
		todo := &entity.Todo{}
		if err := rows.Scan(
			&todo.ID,
			&todo.Title,
			&todo.Description,
			&todo.Completed,
			&todo.CreatedAt,
			&todo.UpdatedAt,
			&todo.DeletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan todo: %w", err)
		}
		todos = append(todos, todo)
	}
	return todos, total, nil
}

func (r *todoRepository) Update(ctx context.Context, todo *entity.Todo) error {
	query := `
		UPDATE todos
		SET title = $2, description = $3, completed = $4, updated_at = $5
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query,
		todo.ID,
		todo.Title,
		todo.Description,
		todo.Completed,
		todo.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update todo: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("todo not found")
	}
	return nil
}

func (r *todoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE todos SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("todo not found")
	}
	return nil
}

func (r *todoRepository) Search(ctx context.Context, query string, offset, limit int) ([]*entity.Todo, int, error) {
	searchPattern := "%" + query + "%"

	countQuery := `SELECT COUNT(*) FROM todos WHERE deleted_at IS NULL AND (title ILIKE $1 OR description ILIKE $1)`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, searchPattern).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count search results: %w", err)
	}

	sqlQuery := `
		SELECT id, title, description, completed, created_at, updated_at, deleted_at
		FROM todos
		WHERE deleted_at IS NULL AND (title ILIKE $1 OR description ILIKE $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, sqlQuery, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("search todos: %w", err)
	}
	defer rows.Close()

	var todos []*entity.Todo
	for rows.Next() {
		todo := &entity.Todo{}
		if err := rows.Scan(
			&todo.ID,
			&todo.Title,
			&todo.Description,
			&todo.Completed,
			&todo.CreatedAt,
			&todo.UpdatedAt,
			&todo.DeletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan todo: %w", err)
		}
		todos = append(todos, todo)
	}
	return todos, total, nil
}
