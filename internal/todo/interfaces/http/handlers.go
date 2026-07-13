package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	commandBus cqrs.CommandBus
	queryBus   cqrs.QueryBus
	validator  *validator.Validator
}

func NewHandler(commandBus cqrs.CommandBus, queryBus cqrs.QueryBus, v *validator.Validator) *Handler {
	return &Handler{commandBus: commandBus, queryBus: queryBus, validator: v}
}

// CreateTodo godoc
// @Summary Create a new todo
// @Description Create a new todo item
// @Tags todos
// @Accept json
// @Produce json
// @Param request body dto.CreateTodoRequest true "Todo to create"
// @Success 201 {object} utils.APIResponse{data=dto.TodoResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /todos [post]
func (h *Handler) CreateTodo(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	resp, err := h.commandBus.Dispatch(r.Context(), command.CreateTodoCommand{
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidTitle) {
			utils.RespondBadRequest(w, err.Error())
			return
		}
	}
	utils.HandleCreated(w, resp, err)
}

// ListTodos godoc
// @Summary List all todos
// @Description Get a list of all todo items with pagination
// @Tags todos
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} utils.APIResponse{data=dto.TodoListResponse}
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /todos [get]
func (h *Handler) ListTodos(w http.ResponseWriter, r *http.Request) {
	cursorStr := r.URL.Query().Get("cursor")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	var cursor *string
	if cursorStr != "" {
		cursor = &cursorStr
	}

	resp, err := h.queryBus.Ask(r.Context(), query.ListTodosQuery{Cursor: cursor, Limit: limit})
	if err != nil {
		utils.MapErrorFromRequest(w, r, err)
		return
	}
	result := resp.(query.ListTodosResult)
	utils.RespondCursorPaginated(w, result.Todos, result.NextCursor, result.PrevCursor, result.HasNext, result.HasPrev, result.Limit)
}

// GetTodo godoc
// @Summary Get a todo by ID
// @Description Get a single todo item by its ID
// @Tags todos
// @Produce json
// @Param id path string true "Todo ID"
// @Success 200 {object} utils.APIResponse{data=dto.TodoResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /todos/{id} [get]
func (h *Handler) GetTodo(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid todo ID")
		return
	}
	resp, err := h.queryBus.Ask(r.Context(), query.GetTodoQuery{ID: id})
	if err != nil {
		if errors.Is(err, service.ErrTodoNotFound) {
			utils.RespondNotFound(w, "todo not found")
			return
		}
		utils.Handle(w, nil, err)
		return
	}
	utils.RespondSuccess(w, resp)
}

// UpdateTodo godoc
// @Summary Update a todo
// @Description Update an existing todo item
// @Tags todos
// @Accept json
// @Produce json
// @Param id path string true "Todo ID"
// @Param request body dto.UpdateTodoRequest true "Fields to update"
// @Success 200 {object} utils.APIResponse{data=dto.TodoResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /todos/{id} [put]
func (h *Handler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid todo ID")
		return
	}
	var req dto.UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	resp, err := h.commandBus.Dispatch(r.Context(), command.UpdateTodoCommand{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, service.ErrTodoNotFound) {
			utils.RespondNotFound(w, "todo not found")
			return
		}
		utils.Handle(w, nil, err)
		return
	}
	utils.RespondSuccess(w, resp)
}

// DeleteTodo godoc
// @Summary Delete a todo
// @Description Delete a todo item by ID
// @Tags todos
// @Param id path string true "Todo ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Security BearerAuth
// @Router /todos/{id} [delete]
func (h *Handler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid todo ID")
		return
	}
	_, err = h.commandBus.Dispatch(r.Context(), command.DeleteTodoCommand{ID: id})
	if err != nil {
		if errors.Is(err, service.ErrTodoNotFound) {
			utils.RespondNotFound(w, "todo not found")
			return
		}
		utils.Handle(w, nil, err)
		return
	}
	utils.RespondSuccess(w, map[string]string{"message": "todo deleted"})
}

// CompleteTodo godoc
// @Summary Complete a todo
// @Description Mark a todo item as completed
// @Tags todos
// @Param id path string true "Todo ID"
// @Success 200 {object} utils.APIResponse{data=dto.TodoResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 409 {object} utils.APIResponse
// @Security BearerAuth
// @Router /todos/{id}/complete [patch]
func (h *Handler) CompleteTodo(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid todo ID")
		return
	}
	resp, err := h.commandBus.Dispatch(r.Context(), command.CompleteTodoCommand{ID: id})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTodoNotFound):
			utils.RespondNotFound(w, "todo not found")
		case errors.Is(err, service.ErrTodoAlreadyDone):
			utils.RespondConflict(w, "todo is already completed")
		default:
			utils.MapErrorFromRequest(w, r, err)
		}
		return
	}
	utils.RespondSuccess(w, resp)
}

// SearchTodos godoc
// @Summary Search todos
// @Description Search todos by title or description
// @Tags todos
// @Produce json
// @Param q query string true "Search query"
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} utils.APIResponse{data=dto.TodoListResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Security BearerAuth
// @Router /todos/search [get]
func (h *Handler) SearchTodos(w http.ResponseWriter, r *http.Request) {
	queryStr := r.URL.Query().Get("q")
	if queryStr == "" {
		utils.RespondBadRequest(w, "search query is required")
		return
	}
	cursorStr := r.URL.Query().Get("cursor")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	var cursor *string
	if cursorStr != "" {
		cursor = &cursorStr
	}

	resp, err := h.queryBus.Ask(r.Context(), query.SearchTodosQuery{Query: queryStr, Cursor: cursor, Limit: limit})
	if err != nil {
		utils.MapErrorFromRequest(w, r, err)
		return
	}
	result := resp.(query.SearchTodosResult)
	utils.RespondCursorPaginated(w, result.Todos, result.NextCursor, result.PrevCursor, result.HasNext, result.HasPrev, result.Limit)
}
