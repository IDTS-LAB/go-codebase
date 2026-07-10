package todo

import (
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/query"
	appService "github.com/IDTS-LAB/go-codebase/internal/todo/application/service"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	"github.com/IDTS-LAB/go-codebase/internal/todo/infrastructure/eventbus"
	"github.com/IDTS-LAB/go-codebase/internal/todo/infrastructure/persistence"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/todo/interfaces/http"
	"go.uber.org/fx"
)

var Module = fx.Module("todo",
	fx.Provide(
		// Infrastructure - NewTodoRepository returns repository.TodoRepository
		persistence.NewTodoRepository,

		// Domain
		service.NewTodoDomainService,

		// Application Commands
		command.NewCreateTodoHandler,
		command.NewUpdateTodoHandler,
		command.NewDeleteTodoHandler,
		command.NewCompleteTodoHandler,

		// Application Queries
		query.NewGetTodoHandler,
		query.NewListTodosHandler,
		query.NewSearchTodosHandler,

		// Application Service
		appService.NewTodoAppService,

		// Events
		eventbus.NewTodoEventHandler,

		// Interface
		httpHandler.NewHandler,
	),

	fx.Invoke(
		func(bus events.EventBus, eh *eventbus.TodoEventHandler) {
			eh.Register(bus)
		},
	),
)
