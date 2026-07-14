package events

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/nats-io/nats.go"
	"go.uber.org/fx"
)

func ensureStream(js nats.JetStreamContext, cfg config.StreamConfig, log domain.Logger) {
	_, err := js.AddStream(&nats.StreamConfig{
		Name:      cfg.Name,
		Subjects:  cfg.Subjects,
		Storage:   nats.FileStorage,
		Retention: nats.InterestPolicy,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		log.Error(context.Background(), "failed to ensure JetStream stream", domain.Error(err))
	}
}

func provideEventBus(cfg *config.Config, js nats.JetStreamContext, log domain.Logger) EventBus {
	var bus EventBus
	if cfg.Events.Driver == "nats" {
		ensureStream(js, cfg.NATS.Stream, log)
		bus = NewNATSEventBus(&jsContextAdapter{js: js})
	} else {
		bus = NewInMemoryEventBus()
	}
	return NewLoggingEventBus(bus, log)
}

var Module = fx.Module("events",
	fx.Provide(provideEventBus),
)
