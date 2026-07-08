package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

var Module = fx.Module("redis",
	fx.Provide(NewRedisCache),
	fx.Provide(NewRedisClient),
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisClient(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
}

func NewRedisCache(client *redis.Client) (domain.Cache, error) {
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	return &RedisCache{client: client}, nil
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("redis get: %w", err)
	}
	return val, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl interface{}) error {
	duration, ok := ttl.(time.Duration)
	if !ok {
		duration = 0
	}
	if err := r.client.Set(ctx, key, value, duration).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

func (r *RedisCache) Del(ctx context.Context, keys ...string) error {
	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}
