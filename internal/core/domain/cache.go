package domain

import "context"

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl interface{}) error
	Del(ctx context.Context, keys ...string) error
}
