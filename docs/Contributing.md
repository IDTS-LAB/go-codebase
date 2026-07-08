# Contributing

## Code Style

- Follow standard Go conventions
- Use `gofmt` and `goimports`
- Run `make lint` before committing
- Keep functions small and focused
- Handle all errors

## Swagger Annotations

When adding or modifying HTTP handlers, add swagger annotations:

```go
// CreateTodo godoc
// @Summary Create a new todo
// @Description Create a new todo item
// @Tags todos
// @Accept json
// @Produce json
// @Param request body dto.CreateTodoRequest true "Todo to create"
// @Success 201 {object} utils.SuccessResponse{data=dto.TodoResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /todos [post]
```

After modifying annotations, regenerate docs:

```bash
make swagger
```

## Commit Messages

Use conventional commits:
- `feat: add new feature`
- `fix: fix bug`
- `docs: update documentation`
- `refactor: refactor code`
- `test: add tests`

## Pull Requests

1. Create a feature branch
2. Make your changes
3. Add tests
4. Update swagger docs if handlers changed: `make swagger`
5. Ensure all tests pass: `make test`
6. Ensure linter passes: `make lint`
7. Submit a PR

## Architecture

Follow the existing architecture:
- DDD for domain modeling
- CQRS for command/query separation
- Clean Architecture for dependency direction
- Modular Monolith for module isolation
- Loose coupling via interfaces in `internal/core/domain/`
