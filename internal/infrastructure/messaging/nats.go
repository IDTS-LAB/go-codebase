package messaging

import (
	"context"
	"fmt"
	"net/http"

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
		func(m *NATSMessenger) nats.JetStreamContext { return m.JetStream() },
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
	conn     *nats.Conn
	js       nats.JetStreamContext
	debugBuf *debugBuffer
}

func NewNATSMessenger(cfg *config.Config) (*NATSMessenger, error) {
	m := &NATSMessenger{}
	if cfg.NATS.DebugEndpoint {
		m.debugBuf = newDebugBuffer(100)
	}
	if cfg.NATS.URL == "" {
		return m, nil
	}

	conn, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}
	m.conn = conn

	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	m.js = js

	return m, nil
}

func (n *NATSMessenger) JetStream() nats.JetStreamContext {
	return n.js
}

func (n *NATSMessenger) DebugHandler() http.Handler {
	return &debugNATSHandler{buffer: n.debugBuf}
}

func (n *NATSMessenger) Publish(ctx context.Context, subject string, data []byte) error {
	if n.conn == nil {
		return nil
	}
	if n.debugBuf != nil {
		n.debugBuf.append(subject, data)
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
