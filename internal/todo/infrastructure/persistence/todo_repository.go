package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cursor"
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

func (r *todoRepository) GetAll(ctx context.Context, cursorArg *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error) {
	args := []interface{}{}
	whereClause := "WHERE deleted_at IS NULL"

	if r.tenantConfig != nil && r.tenantConfig.Enabled {
		tenantID := middleware.GetTenantID(ctx)
		if tenantID != "" {
			whereClause += fmt.Sprintf(" AND tenant_id = $%d", len(args)+1)
			args = append(args, tenantID)
		}
	}

	nextPos := len(args) + 1
	if cursorArg != nil {
		c, err := cursor.Decode(*cursorArg)
		if err != nil {
			return nil, nil, nil, false, false, fmt.Errorf("invalid cursor: %w", err)
		}
		whereClause += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", nextPos, nextPos+1)
		args = append(args, c.Timestamp, c.ID)
		nextPos += 2
	}

	dataQuery := fmt.Sprintf("SELECT id, title, description, completed, created_at, updated_at, deleted_at FROM todos %s ORDER BY created_at DESC, id DESC LIMIT $%d", whereClause, nextPos)
	queryArgs := append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, dataQuery, queryArgs...)
	if err != nil {
		return nil, nil, nil, false, false, fmt.Errorf("query todos: %w", err)
	}
	defer rows.Close()

	var todos []*entity.Todo
	for rows.Next() {
		var t entity.Todo
		var deletedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt, &deletedAt); err != nil {
			return nil, nil, nil, false, false, fmt.Errorf("scan todo: %w", err)
		}
		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.Time
		}
		todos = append(todos, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, false, false, fmt.Errorf("rows iteration: %w", err)
	}

	hasNext := len(todos) > limit
	if hasNext {
		todos = todos[:limit]
	}

	var nextCursor *string
	var prevCursor *string
	if len(todos) > 0 {
		last := todos[len(todos)-1]
		nc := cursor.Encode(last.CreatedAt, last.ID)
		nextCursor = &nc

		first := todos[0]
		pc := cursor.Encode(first.CreatedAt, first.ID)
		prevCursor = &pc
	}

	hasPrev := cursorArg != nil
	if hasPrev && len(todos) == 0 {
		hasPrev = false
	}

	return todos, nextCursor, prevCursor, hasNext, hasPrev, nil
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

func (r *todoRepository) Search(ctx context.Context, query string, cursorArg *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error) {
	searchPattern := "%" + query + "%"

	args := []interface{}{searchPattern}
	whereClause := "WHERE deleted_at IS NULL AND (title ILIKE $1 OR description ILIKE $1)"
	nextPos := 2

	if r.tenantConfig != nil && r.tenantConfig.Enabled {
		tenantID := middleware.GetTenantID(ctx)
		if tenantID != "" {
			whereClause += fmt.Sprintf(" AND tenant_id = $%d", nextPos)
			args = append(args, tenantID)
			nextPos++
		}
	}

	if cursorArg != nil {
		c, err := cursor.Decode(*cursorArg)
		if err != nil {
			return nil, nil, nil, false, false, fmt.Errorf("invalid cursor: %w", err)
		}
		whereClause += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", nextPos, nextPos+1)
		args = append(args, c.Timestamp, c.ID)
		nextPos += 2
	}

	dataQuery := fmt.Sprintf("SELECT id, title, description, completed, created_at, updated_at, deleted_at FROM todos %s ORDER BY created_at DESC, id DESC LIMIT $%d", whereClause, nextPos)
	dataArgs := append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, nil, nil, false, false, fmt.Errorf("search todos: %w", err)
	}
	defer rows.Close()

	var todos []*entity.Todo
	for rows.Next() {
		var t entity.Todo
		var deletedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt, &deletedAt); err != nil {
			return nil, nil, nil, false, false, fmt.Errorf("scan todo: %w", err)
		}
		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.Time
		}
		todos = append(todos, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, false, false, fmt.Errorf("rows iteration: %w", err)
	}

	hasNext := len(todos) > limit
	if hasNext {
		todos = todos[:limit]
	}

	var nextCursor *string
	var prevCursor *string
	if len(todos) > 0 {
		last := todos[len(todos)-1]
		nc := cursor.Encode(last.CreatedAt, last.ID)
		nextCursor = &nc

		first := todos[0]
		pc := cursor.Encode(first.CreatedAt, first.ID)
		prevCursor = &pc
	}

	hasPrev := cursorArg != nil
	if hasPrev && len(todos) == 0 {
		hasPrev = false
	}

	return todos, nextCursor, prevCursor, hasNext, hasPrev, nil
}
