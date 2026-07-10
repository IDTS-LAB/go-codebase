package events

import "go.uber.org/fx"

var Module = fx.Module("events",
	fx.Provide(
		NewInMemoryEventBus,
		fx.Annotate(NewLoggingEventBus, fx.As(new(EventBus))),
	),
)
