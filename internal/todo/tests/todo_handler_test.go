package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/events"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/todo/application/query"
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

func (m *MockTodoRepo) GetAll(ctx context.Context, cursor *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error) {
	args := m.Called(ctx, cursor, limit)
	return args.Get(0).([]*entity.Todo), nil, nil, false, false, args.Error(1)
}

func (m *MockTodoRepo) Update(ctx context.Context, todo *entity.Todo) error {
	args := m.Called(ctx, todo)
	return args.Error(0)
}

func (m *MockTodoRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTodoRepo) Search(ctx context.Context, queryStr string, cursor *string, limit int) ([]*entity.Todo, *string, *string, bool, bool, error) {
	args := m.Called(ctx, queryStr, cursor, limit)
	return args.Get(0).([]*entity.Todo), nil, nil, false, false, args.Error(1)
}

func setupHandler(t *testing.T) (*httpHandler.Handler, *MockTodoRepo) {
	t.Helper()
	repo := new(MockTodoRepo)
	domainSvc := service.NewTodoDomainService(repo)
	eventBus := events.NewInMemoryEventBus()

	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()

	cmdBus.Register(command.CreateTodoCommand{}, command.NewCreateTodoHandler(domainSvc, eventBus))
	cmdBus.Register(command.UpdateTodoCommand{}, command.NewUpdateTodoHandler(domainSvc, eventBus))
	cmdBus.Register(command.DeleteTodoCommand{}, command.NewDeleteTodoHandler(domainSvc, eventBus))
	cmdBus.Register(command.CompleteTodoCommand{}, command.NewCompleteTodoHandler(domainSvc, eventBus))

	qBus.Register(query.GetTodoQuery{}, query.NewGetTodoHandler(domainSvc))
	qBus.Register(query.ListTodosQuery{}, query.NewListTodosHandler(domainSvc))
	qBus.Register(query.SearchTodosQuery{}, query.NewSearchTodosHandler(domainSvc))

	v := validator.New()
	h := httpHandler.NewHandler(cmdBus, qBus, v)
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
	assert.NotNil(t, resp["data"])
	assert.Nil(t, resp["meta"])
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
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "VALIDATION_ERROR", resp["error"].(map[string]interface{})["code"])
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
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
	assert.Nil(t, resp["meta"])
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
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "NOT_FOUND", resp["error"].(map[string]interface{})["code"])
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
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	assert.Nil(t, resp["meta"])
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
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "CONFLICT", resp["error"].(map[string]interface{})["code"])
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
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
	assert.Nil(t, resp["meta"])
	repo.AssertExpectations(t)
}

func TestSearchTodos_Success(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	todos := []*entity.Todo{newTestTodo(id, "Test")}
	repo.On("Search", mock.Anything, "test", (*string)(nil), 20).Return(todos, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/search?q=test", nil)
	rr := httptest.NewRecorder()

	h.SearchTodos(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
	assert.NotNil(t, resp["meta"])
	repo.AssertExpectations(t)
}

func TestSearchTodos_MissingQuery(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/search", nil)
	rr := httptest.NewRecorder()

	h.SearchTodos(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "VALIDATION_ERROR", resp["error"].(map[string]interface{})["code"])
}

func TestGetTodo_InvalidID(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": "not-a-uuid"})

	h.GetTodo(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
}

func TestListTodos_Success(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	todos := []*entity.Todo{newTestTodo(id, "Test")}
	repo.On("GetAll", mock.Anything, (*string)(nil), 20).Return(todos, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	rr := httptest.NewRecorder()

	h.ListTodos(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
	assert.NotNil(t, resp["meta"])
	repo.AssertExpectations(t)
}

func TestListTodos_WithCursor(t *testing.T) {
	h, repo := setupHandler(t)

	cursor := "next-cursor"
	todos := []*entity.Todo{}
	repo.On("GetAll", mock.Anything, &cursor, 10).Return(todos, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos?cursor=next-cursor&limit=10", nil)
	rr := httptest.NewRecorder()

	h.ListTodos(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	repo.AssertExpectations(t)
}

func TestListTodos_RepoError(t *testing.T) {
	h, repo := setupHandler(t)

	repo.On("GetAll", mock.Anything, (*string)(nil), 20).Return([]*entity.Todo{}, service.ErrTodoNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	rr := httptest.NewRecorder()

	h.ListTodos(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	repo.AssertExpectations(t)
}

func TestUpdateTodo_Success(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	todo := newTestTodoWithDesc(id, "Updated Title", "Updated Desc")
	repo.On("GetByID", mock.Anything, id).Return(todo, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil)

	body, _ := json.Marshal(dto.UpdateTodoRequest{Title: "Updated Title", Description: "Updated Desc"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todos/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.UpdateTodo(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
	assert.Nil(t, resp["meta"])
	repo.AssertExpectations(t)
}

func TestUpdateTodo_InvalidID(t *testing.T) {
	h, _ := setupHandler(t)

	body, _ := json.Marshal(dto.UpdateTodoRequest{Title: "Updated Title"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todos/not-a-uuid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": "not-a-uuid"})

	h.UpdateTodo(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
}

func TestUpdateTodo_BadJSON(t *testing.T) {
	h, _ := setupHandler(t)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todos/"+id.String(), bytes.NewReader([]byte("{not json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.UpdateTodo(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
}

func TestUpdateTodo_NotFound(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, service.ErrTodoNotFound)

	body, _ := json.Marshal(dto.UpdateTodoRequest{Title: "Updated Title"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todos/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.UpdateTodo(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "NOT_FOUND", resp["error"].(map[string]interface{})["code"])
	repo.AssertExpectations(t)
}

func TestDeleteTodo_InvalidID(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": "not-a-uuid"})

	h.DeleteTodo(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
}

func TestDeleteTodo_NotFound(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, service.ErrTodoNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/"+id.String(), nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.DeleteTodo(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "NOT_FOUND", resp["error"].(map[string]interface{})["code"])
	repo.AssertExpectations(t)
}

func TestCompleteTodo_NotFound(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, service.ErrTodoNotFound)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/"+id.String()+"/complete", nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.CompleteTodo(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "NOT_FOUND", resp["error"].(map[string]interface{})["code"])
	repo.AssertExpectations(t)
}

func TestCompleteTodo_InvalidID(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/not-a-uuid/complete", nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": "not-a-uuid"})

	h.CompleteTodo(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
}

func TestCreateTodo_BadJSON(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateTodo(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
}

func TestCreateTodo_RepoError(t *testing.T) {
	h, repo := setupHandler(t)

	body, _ := json.Marshal(dto.CreateTodoRequest{Title: "Buy milk", Description: "2%"})
	repo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateTodo(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	repo.AssertExpectations(t)
}

func TestGetTodo_RepoError(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+id.String(), nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.GetTodo(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "NOT_FOUND", resp["error"].(map[string]interface{})["code"])
	repo.AssertExpectations(t)
}

func TestUpdateTodo_RepoError(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, errors.New("db error"))

	body, _ := json.Marshal(dto.UpdateTodoRequest{Title: "Updated Title"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todos/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.UpdateTodo(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "NOT_FOUND", resp["error"].(map[string]interface{})["code"])
	repo.AssertExpectations(t)
}

func TestDeleteTodo_RepoError(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/"+id.String(), nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.DeleteTodo(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "NOT_FOUND", resp["error"].(map[string]interface{})["code"])
	repo.AssertExpectations(t)
}

func TestCompleteTodo_RepoError(t *testing.T) {
	h, repo := setupHandler(t)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/"+id.String()+"/complete", nil)
	rr := httptest.NewRecorder()
	req = withChiContext(req, map[string]string{"id": id.String()})

	h.CompleteTodo(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	assert.Equal(t, "NOT_FOUND", resp["error"].(map[string]interface{})["code"])
	repo.AssertExpectations(t)
}

func TestSearchTodos_RepoError(t *testing.T) {
	h, repo := setupHandler(t)

	repo.On("Search", mock.Anything, "test", (*string)(nil), 20).Return([]*entity.Todo{}, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/search?q=test", nil)
	rr := httptest.NewRecorder()

	h.SearchTodos(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
	repo.AssertExpectations(t)
}
