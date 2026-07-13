package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/todo/infrastructure/persistence/sqlc"
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
	todo := &entity.Todo{
		Entity:      domain.Entity{ID: row.ID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt},
		Title:       row.Title,
		Description: row.Description,
		Completed:   row.Completed,
	}
	if row.DeletedAt.Valid {
		todo.DeletedAt = &row.DeletedAt.Time
	}
	return todo, nil
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
