package events

import (
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/nats-io/nats.go"
	"go.uber.org/fx"
)

func ensureStream(js nats.JetStreamContext, cfg config.StreamConfig) {
	_, err := js.AddStream(&nats.StreamConfig{
		Name:      cfg.Name,
		Subjects:  cfg.Subjects,
		Storage:   nats.FileStorage,
		Retention: nats.InterestPolicy,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		// log but don't fatal
	}
}

func provideEventBus(cfg *config.Config, js nats.JetStreamContext, log domain.Logger) EventBus {
	var bus EventBus
	if cfg.Events.Driver == "nats" {
		ensureStream(js, cfg.NATS.Stream)
		bus = NewNATSEventBus(&jsContextAdapter{js: js})
	} else {
		bus = NewInMemoryEventBus()
	}
	return NewLoggingEventBus(bus, log)
}

var Module = fx.Module("events",
	fx.Provide(provideEventBus),
)
