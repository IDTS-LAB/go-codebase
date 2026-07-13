package todo

import (
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	"github.com/IDTS-LAB/go-codebase/internal/todo/infrastructure/eventbus"
	"github.com/IDTS-LAB/go-codebase/internal/todo/infrastructure/persistence"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/todo/interfaces/http"
	"go.uber.org/fx"
)

var Module = fx.Module("todo",
	fx.Provide(
		// Infrastructure
		persistence.NewTodoRepository,

		// Domain
		service.NewTodoDomainService,

		// Events
		eventbus.NewTodoEventHandler,

		// Interface
		httpHandler.NewHandler,
	),

	fx.Invoke(
		registerHandlers,
		func(bus events.EventBus, eh *eventbus.TodoEventHandler) {
			eh.Register(bus)
		},
	),
)

func registerHandlers(
	commandBus cqrs.CommandBus,
	queryBus cqrs.QueryBus,
	domainSvc *service.TodoDomainService,
	eventBus events.EventBus,
) {
	commandBus.Register(command.CreateTodoCommand{}, command.NewCreateTodoHandler(domainSvc, eventBus))
	commandBus.Register(command.UpdateTodoCommand{}, command.NewUpdateTodoHandler(domainSvc, eventBus))
	commandBus.Register(command.DeleteTodoCommand{}, command.NewDeleteTodoHandler(domainSvc, eventBus))
	commandBus.Register(command.CompleteTodoCommand{}, command.NewCompleteTodoHandler(domainSvc, eventBus))

	queryBus.Register(query.GetTodoQuery{}, query.NewGetTodoHandler(domainSvc))
	queryBus.Register(query.ListTodosQuery{}, query.NewListTodosHandler(domainSvc))
	queryBus.Register(query.SearchTodosQuery{}, query.NewSearchTodosHandler(domainSvc))
}
