package domain

import "context"

type Messenger interface {
	Publish(ctx context.Context, subject string, data []byte) error
	Subscribe(ctx context.Context, subject string, handler func(data []byte)) error
	Close() error
}
