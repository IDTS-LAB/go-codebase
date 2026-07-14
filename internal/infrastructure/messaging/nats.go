package messaging

import (
	"context"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/fx"
)

var Module = fx.Module("nats",
	fx.Provide(
		NewNATSMessenger,
		fx.Annotate(
			func(m *NATSMessenger) domain.Messenger { return m },
			fx.As(new(domain.Messenger)),
		),
	),
)

var (
	natsPublishedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_published_total",
		Help: "Total number of NATS messages published",
	}, []string{"subject"})

	natsReceivedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_received_total",
		Help: "Total number of NATS messages received",
	}, []string{"subject"})

	natsPublishBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_publish_bytes_total",
		Help: "Total bytes published to NATS",
	}, []string{"subject"})

	natsReceivedBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_received_bytes_total",
		Help: "Total bytes received from NATS",
	}, []string{"subject"})
)

type NATSMessenger struct {
	conn *nats.Conn
}

func NewNATSMessenger(cfg *config.Config) (*NATSMessenger, error) {
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
	natsPublishedTotal.WithLabelValues(subject).Inc()
	natsPublishBytesTotal.WithLabelValues(subject).Add(float64(len(data)))
	return n.conn.Publish(subject, data)
}

func (n *NATSMessenger) Subscribe(ctx context.Context, subject string, handler func(data []byte)) error {
	if n.conn == nil {
		return nil
	}
	_, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		natsReceivedTotal.WithLabelValues(subject).Inc()
		natsReceivedBytesTotal.WithLabelValues(subject).Add(float64(len(msg.Data)))
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
