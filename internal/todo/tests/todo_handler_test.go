package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/query"
	appService "github.com/IDTS-LAB/go-codebase/internal/todo/application/service"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/service"
	httpHandler "github.com/IDTS-LAB/go-codebase/internal/todo/interfaces/http"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTodoRepo struct {
	mock.Mock
}

func (m *MockTodoRepo) Create(ctx context.Context, todo *entity.Todo) error {
	args := m.Called(ctx, todo)
	return args.Error(0)
}

func (m *MockTodoRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Todo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Todo), args.Error(1)
}

func (m *MockTodoRepo) GetAll(ctx context.Context, offset, limit int) ([]*entity.Todo, int, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*entity.Todo), args.Int(1), args.Error(2)
}

func (m *MockTodoRepo) Update(ctx context.Context, todo *entity.Todo) error {
	args := m.Called(ctx, todo)
	return args.Error(0)
}

func (m *MockTodoRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTodoRepo) Search(ctx context.Context, queryStr string, offset, limit int) ([]*entity.Todo, int, error) {
	args := m.Called(ctx, queryStr, offset, limit)
	return args.Get(0).([]*entity.Todo), args.Int(1), args.Error(2)
}

func setupHandler(t *testing.T) (*httpHandler.Handler, *MockTodoRepo) {
	t.Helper()
	repo := new(MockTodoRepo)
	domainSvc := service.NewTodoDomainService(repo)

	createH := command.NewCreateTodoHandler(domainSvc)
	updateH := command.NewUpdateTodoHandler(domainSvc)
	deleteH := command.NewDeleteTodoHandler(domainSvc)
	completeH := command.NewCompleteTodoHandler(domainSvc)
	getH := query.NewGetTodoHandler(domainSvc)
	listH := query.NewListTodosHandler(domainSvc)
	searchH := query.NewSearchTodosHandler(domainSvc)

	appSvc := appService.NewTodoAppService(createH, updateH, deleteH, completeH, getH, listH, searchH)
	v := validator.New()
	h := httpHandler.NewHandler(appSvc, v)
	return h, repo
}

func withChiContext(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestCreateTodo_Success(t *testing.T) {
	h, repo := setupHandler(t)

	body, _ := json.Marshal(dto.CreateTodoRequest{Title: "Buy milk", Description: "2%"})
	repo.On("Create", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateTodo(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	repo.AssertExpectations(t)
}

func TestCreateTodo_ValidationError(t *testing.T) {
	h, _ := setupHandler(t)

	body, _ := json.Marshal(dto.CreateTodoRequest{Title: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateTodo(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetTodo_Success(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	todo := newTestTodoWithDesc(id, "Test", "Desc")
	repo.On("GetByID", mock.Anything, id).Return(todo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+id.String(), nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.GetTodo(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	repo.AssertExpectations(t)
}

func TestGetTodo_NotFound(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, service.ErrTodoNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+id.String(), nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.GetTodo(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	repo.AssertExpectations(t)
}

func TestDeleteTodo_Success(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	todo := newTestTodo(id, "Test")
	repo.On("GetByID", mock.Anything, id).Return(todo, nil)
	repo.On("Delete", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/"+id.String(), nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.DeleteTodo(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	repo.AssertExpectations(t)
}

func TestCompleteTodo_AlreadyDone(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	todo := newTestTodoCompleted(id, "Test")
	repo.On("GetByID", mock.Anything, id).Return(todo, nil)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/"+id.String()+"/complete", nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.CompleteTodo(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	repo.AssertExpectations(t)
}

func TestCompleteTodo_Success(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	todo := newTestTodo(id, "Test")
	repo.On("GetByID", mock.Anything, id).Return(todo, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/"+id.String()+"/complete", nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.CompleteTodo(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	repo.AssertExpectations(t)
}

func TestSearchTodos_Success(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	todos := []*entity.Todo{newTestTodo(id, "Test")}
	repo.On("Search", mock.Anything, "test", 0, 20).Return(todos, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/search?q=test", nil)
	rr := httptest.NewRecorder()

	h.SearchTodos(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	repo.AssertExpectations(t)
}

func TestSearchTodos_MissingQuery(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/search", nil)
	rr := httptest.NewRecorder()

	h.SearchTodos(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
