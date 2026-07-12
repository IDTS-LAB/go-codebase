package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/repository"
	"github.com/google/uuid"
)

type todoRepository struct {
	db           *sql.DB
	tenantConfig *tenantfilter.Config
}

func NewTodoRepository(db *sql.DB, tenantConfig *tenantfilter.Config) repository.TodoRepository {
	return &todoRepository{db: db, tenantConfig: tenantConfig}
}

func (r *todoRepository) Create(ctx context.Context, todo *entity.Todo) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO todos (id, title, description, completed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, todo.ID, todo.Title, todo.Description, todo.Completed, todo.CreatedAt, todo.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert todo: %w", err)
	}
	return nil
}

func (r *todoRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Todo, error) {
	var t entity.Todo
	var deletedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, description, completed, created_at, updated_at, deleted_at
		FROM todos WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt, &deletedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("todo not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get todo: %w", err)
	}
	if deletedAt.Valid {
		t.DeletedAt = &deletedAt.Time
	}
	return &t, nil
}

func (r *todoRepository) GetAll(ctx context.Context, offset, limit int) ([]*entity.Todo, int, error) {
	var args []interface{}
	countQuery := "SELECT COUNT(*) FROM todos WHERE deleted_at IS NULL"
	dataQuery := "SELECT id, title, description, completed, created_at, updated_at, deleted_at FROM todos WHERE deleted_at IS NULL"

	if r.tenantConfig != nil && r.tenantConfig.Enabled {
		tenantID := middleware.GetTenantID(ctx)
		if tenantID != "" {
			countQuery += " AND tenant_id = $1"
			dataQuery += " AND tenant_id = $1"
			args = append(args, tenantID)
		}
	}

	var total int64
	var err error
	if len(args) > 0 {
		err = r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	} else {
		err = r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("count todos: %w", err)
	}

	if len(args) > 0 {
		dataQuery += " ORDER BY created_at DESC LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	} else {
		dataQuery += " ORDER BY created_at DESC LIMIT $1 OFFSET $2"
		args = append(args, limit, offset)
	}

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query todos: %w", err)
	}
	defer rows.Close()

	var todos []*entity.Todo
	for rows.Next() {
		var t entity.Todo
		var deletedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt, &deletedAt); err != nil {
			return nil, 0, fmt.Errorf("scan todo: %w", err)
		}
		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.Time
		}
		todos = append(todos, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	return todos, int(total), nil
}

func (r *todoRepository) Update(ctx context.Context, todo *entity.Todo) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE todos SET title = $2, description = $3, completed = $4, updated_at = $5 WHERE id = $1 AND deleted_at IS NULL
	`, todo.ID, todo.Title, todo.Description, todo.Completed, todo.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update todo: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update todo rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("todo not found")
	}
	return nil
}

func (r *todoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE todos SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete todo rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("todo not found")
	}
	return nil
}

func (r *todoRepository) Search(ctx context.Context, query string, offset, limit int) ([]*entity.Todo, int, error) {
	searchPattern := "%" + query + "%"

	fromWhere := "FROM todos WHERE deleted_at IS NULL AND (title ILIKE $1 OR description ILIKE $1)"
	countQuery := "SELECT COUNT(*) " + fromWhere
	dataQuery := "SELECT id, title, description, completed, created_at, updated_at, deleted_at " + fromWhere

	args := []interface{}{searchPattern}
	nextPos := 2

	if r.tenantConfig != nil && r.tenantConfig.Enabled {
		tenantID := middleware.GetTenantID(ctx)
		if tenantID != "" {
			tenantClause := fmt.Sprintf(" AND tenant_id = $%d", nextPos)
			countQuery += tenantClause
			dataQuery += tenantClause
			args = append(args, tenantID)
			nextPos++
		}
	}

	dataQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", nextPos, nextPos+1)

	var total int64
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count search results: %w", err)
	}

	dataArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("search todos: %w", err)
	}
	defer rows.Close()

	var todos []*entity.Todo
	for rows.Next() {
		var t entity.Todo
		var deletedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt, &deletedAt); err != nil {
			return nil, 0, fmt.Errorf("scan todo: %w", err)
		}
		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.Time
		}
		todos = append(todos, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	return todos, int(total), nil
}
