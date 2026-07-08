package messaging

import (
	"context"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/nats-io/nats.go"
	"go.uber.org/fx"
)

var Module = fx.Module("nats", fx.Provide(NewNATSMessenger))

type NATSMessenger struct {
	conn *nats.Conn
}

func NewNATSMessenger(cfg *config.Config) (domain.Messenger, error) {
	if cfg.NATS.URL == "" {
		return &NATSMessenger{}, nil
	}

	conn, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}

	return &NATSMessenger{conn: conn}, nil
}

func (n *NATSMessenger) Publish(ctx context.Context, subject string, data []byte) error {
	if n.conn == nil {
		return nil
	}
	return n.conn.Publish(subject, data)
}

func (n *NATSMessenger) Subscribe(ctx context.Context, subject string, handler func(data []byte)) error {
	if n.conn == nil {
		return nil
	}
	_, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	return err
}

func (n *NATSMessenger) Close() error {
	if n.conn != nil {
		return n.conn.Drain()
	}
	return nil
}
