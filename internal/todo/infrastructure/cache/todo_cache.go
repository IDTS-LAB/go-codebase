package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/google/uuid"
)

const todoCachePrefix = "todo:"
const todoCacheTTL = 10 * time.Minute

type TodoCache struct {
	cache domain.Cache
}

func NewTodoCache(cache domain.Cache) *TodoCache {
	return &TodoCache{cache: cache}
}

func (c *TodoCache) Get(ctx context.Context, id uuid.UUID) (*entity.Todo, error) {
	key := todoCachePrefix + id.String()
	val, err := c.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var todo entity.Todo
	if err := json.Unmarshal([]byte(val), &todo); err != nil {
		return nil, fmt.Errorf("unmarshal todo: %w", err)
	}
	return &todo, nil
}

func (c *TodoCache) Set(ctx context.Context, todo *entity.Todo) error {
	key := todoCachePrefix + todo.ID.String()
	data, err := json.Marshal(todo)
	if err != nil {
		return fmt.Errorf("marshal todo: %w", err)
	}
	return c.cache.Set(ctx, key, string(data), todoCacheTTL)
}

func (c *TodoCache) Delete(ctx context.Context, id uuid.UUID) error {
	key := todoCachePrefix + id.String()
	return c.cache.Del(ctx, key)
}
