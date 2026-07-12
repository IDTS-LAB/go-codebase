package cqrs

import "go.uber.org/fx"

var Module = fx.Module("cqrs",
	fx.Provide(
		fx.Annotate(
			NewInMemoryCommandBus,
			fx.As(new(CommandBus)),
		),

		fx.Annotate(
			NewInMemoryQueryBus,
			fx.As(new(QueryBus)),
		),
	),
)
