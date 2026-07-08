package domain

import "context"

type Repository[T any] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id interface{}) (*T, error)
	GetAll(ctx context.Context, offset, limit int) ([]*T, int, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id interface{}) error
}

type SearchableRepository[T any] interface {
	Repository[T]
	Search(ctx context.Context, query string, offset, limit int) ([]*T, int, error)
}
