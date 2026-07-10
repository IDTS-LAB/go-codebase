package events

import "go.uber.org/fx"

var Module = fx.Module("events",
	fx.Provide(
		fx.Annotate(NewInMemoryEventBus, fx.As(new(EventBus))),
	),
)
