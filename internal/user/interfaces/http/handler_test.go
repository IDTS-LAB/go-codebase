package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authEntity "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/query"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockHandler struct {
	result any
	err    error
}

func (h *mockHandler) Handle(ctx context.Context, _ any) (any, error) {
	return h.result, h.err
}

func makeTestUser(id uuid.UUID) *authEntity.User {
	u := authEntity.NewUser("john@example.com", "hashed", "John Doe")
	u.ID = id
	return u
}

func setChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func setUserIDOnCtx(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func decodeAPIResponse(t *testing.T, body []byte) utils.APIResponse {
	var resp utils.APIResponse
	err := json.Unmarshal(body, &resp)
	assert.NoError(t, err)
	return resp
}

func TestHandler_List(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id1 := uuid.New()
	id2 := uuid.New()
	users := []*authEntity.User{makeTestUser(id1), makeTestUser(id2)}
	qBus.Register(query.ListUsersQuery{}, &mockHandler{result: query.ListUsersResult{Users: users, Total: 2}})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)

	listData, _ := json.Marshal(resp.Data)
	var usersResp []UserResponse
	json.Unmarshal(listData, &usersResp)
	assert.Len(t, usersResp, 2)
	assert.Equal(t, id1.String(), usersResp[0].ID)
	assert.NotNil(t, resp.Meta)
	assert.Equal(t, 2, resp.Meta.Total)
}

func TestHandler_List_WithPagination(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	qBus.Register(query.ListUsersQuery{}, &mockHandler{result: query.ListUsersResult{Users: []*authEntity.User{}, Total: 0}})

	req := httptest.NewRequest(http.MethodGet, "/users?offset=10&limit=5", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_List_ClampsLimit(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	qBus.Register(query.ListUsersQuery{}, &mockHandler{result: query.ListUsersResult{Users: []*authEntity.User{}, Total: 0}})

	req := httptest.NewRequest(http.MethodGet, "/users?limit=200", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	user := makeTestUser(id)
	qBus.Register(query.GetUserQuery{}, &mockHandler{result: user})

	req := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

func TestHandler_Get_InvalidUUID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	req := httptest.NewRequest(http.MethodGet, "/users/invalid", nil)
	req = setChiURLParam(req, "id", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestHandler_Get_NotFound(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	qBus.Register(query.GetUserQuery{}, &mockHandler{err: domain.ErrNotFound})

	req := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Get_ServiceError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	qBus.Register(query.GetUserQuery{}, &mockHandler{err: errors.New("unexpected error")})

	req := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_Me_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	user := makeTestUser(id)
	qBus.Register(query.GetUserQuery{}, &mockHandler{result: user})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = setUserIDOnCtx(req, id.String())
	w := httptest.NewRecorder()
	h.Me(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

func TestHandler_Me_Unauthenticated(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	h.Me(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
}

func TestHandler_Me_InvalidUserID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = setUserIDOnCtx(req, "not-a-valid-uuid")
	w := httptest.NewRecorder()
	h.Me(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	user := makeTestUser(id)
	user.Name = "Jane Doe"
	user.Email = "jane@example.com"
	user.IsActive = false
	cmdBus.Register(command.UpdateUserCommand{}, &mockHandler{result: user})

	body := `{"name":"Jane Doe","email":"jane@example.com","is_active":false}`
	req := httptest.NewRequest(http.MethodPut, "/users/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

func TestHandler_Update_InvalidUUID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	req := httptest.NewRequest(http.MethodPut, "/users/invalid", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "id", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_InvalidBody(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/users/"+id.String(), strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_NotFound(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	cmdBus.Register(command.UpdateUserCommand{}, &mockHandler{err: domain.ErrNotFound})

	body := `{"name":"Jane Doe"}`
	req := httptest.NewRequest(http.MethodPut, "/users/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Delete_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	cmdBus.Register(command.DeleteUserCommand{}, &mockHandler{})

	req := httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "user deleted", resp.Data.(map[string]interface{})["message"])
}

func TestHandler_Delete_InvalidUUID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	req := httptest.NewRequest(http.MethodDelete, "/users/invalid", nil)
	req = setChiURLParam(req, "id", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Delete_NotFound(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	cmdBus.Register(command.DeleteUserCommand{}, &mockHandler{err: domain.ErrNotFound})

	req := httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
