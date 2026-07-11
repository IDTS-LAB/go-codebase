package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	appService "github.com/IDTS-LAB/go-codebase/internal/todo/application/service"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	appService *appService.TodoAppService
	validator  *validator.Validator
}

func NewHandler(appService *appService.TodoAppService, v *validator.Validator) *Handler {
	return &Handler{appService: appService, validator: v}
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
	resp, err := h.appService.CreateTodo(r.Context(), req)
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
	page := 1
	perPage := 20
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		fmt.Sscanf(pp, "%d", &perPage)
	}
	if perPage > 100 {
		perPage = 100
	}
	resp, err := h.appService.ListTodos(r.Context(), page, perPage)
	utils.HandlePaginated(w, resp.Todos, page, perPage, resp.Total, err)
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
	resp, err := h.appService.GetTodo(r.Context(), id)
	if err != nil && errors.Is(err, service.ErrTodoNotFound) {
		utils.RespondNotFound(w, "todo not found")
		return
	}
	utils.Handle(w, resp, err)
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
	resp, err := h.appService.UpdateTodo(r.Context(), id, req)
	if err != nil && errors.Is(err, service.ErrTodoNotFound) {
		utils.RespondNotFound(w, "todo not found")
		return
	}
	utils.Handle(w, resp, err)
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
	err = h.appService.DeleteTodo(r.Context(), id)
	if err != nil && errors.Is(err, service.ErrTodoNotFound) {
		utils.RespondNotFound(w, "todo not found")
		return
	}
	utils.HandleNoContent(w, err)
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
	resp, err := h.appService.CompleteTodo(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTodoNotFound):
			utils.RespondNotFound(w, "todo not found")
		case errors.Is(err, service.ErrTodoAlreadyDone):
			utils.RespondConflict(w, "todo is already completed")
		default:
			utils.MapError(w, err)
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
	page := 1
	perPage := 20
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		fmt.Sscanf(pp, "%d", &perPage)
	}
	resp, err := h.appService.SearchTodos(r.Context(), queryStr, page, perPage)
	utils.HandlePaginated(w, resp.Todos, page, perPage, resp.Total, err)
}
