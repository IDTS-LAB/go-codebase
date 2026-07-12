package cqrs

import "go.uber.org/fx"

var Module = fx.Module("cqrs",
	fx.Provide(
		NewInMemoryCommandBus,
		NewInMemoryQueryBus,
	),
)
