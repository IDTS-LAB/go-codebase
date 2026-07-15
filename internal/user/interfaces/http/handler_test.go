package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	authEntity "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/user/public"
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

type mockProfileProvider struct {
	profile *public.UserProfile
	err     error
}

func (m *mockProfileProvider) GetProfile(ctx context.Context, userID uuid.UUID) (*public.UserProfile, error) {
	return m.profile, m.err
}

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func withUserID(r *http.Request, userID uuid.UUID) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, userID.String()))
}

func TestCreate_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	userID := uuid.New()
	cmdBus.Register(command.CreateUserCommand{}, &mockHandler{
		result: &authEntity.User{
			Entity: domain.Entity{ID: userID},
			Email:  "new@example.com",
			Name:   "New User",
		},
	})

	body, _ := json.Marshal(CreateUserRequest{Email: "new@example.com", Name: "New User", IsActive: true})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	h.Create(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
}

func TestCreate_BadJSON(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader([]byte("{invalid")))
	r.Header.Set("Content-Type", "application/json")

	h.Create(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Nil(t, resp["data"])
}

func TestCreate(t *testing.T) {
	t.Run("validation error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		h := NewHandler(cmdBus, qBus)

		cmdBus.Register(command.CreateUserCommand{}, &mockHandler{err: domain.ErrValidation})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader([]byte("{}")))
		r.Header.Set("Content-Type", "application/json")

		h.Create(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		assert.False(t, resp["success"].(bool))
	})

	t.Run("bus error status", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		h := NewHandler(cmdBus, qBus)

		cmdBus.Register(command.CreateUserCommand{}, &mockHandler{err: errors.New("db error")})

		body, _ := json.Marshal(CreateUserRequest{Email: "fail@example.com", Name: "Fail"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.Create(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		assert.False(t, resp["success"].(bool))
	})
}

func TestCreate_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	cmdBus.Register(command.CreateUserCommand{}, &mockHandler{err: errors.New("db error")})

	body, _ := json.Marshal(CreateUserRequest{Email: "fail@example.com", Name: "Fail"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	h.Create(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
}

func TestList_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	qBus.Register(query.ListUsersQuery{}, &mockHandler{
		result: query.ListUsersResult{
			Users:   []*authEntity.User{},
			Limit:   20,
		},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users", nil)

	h.List(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
}

func TestList_WithCursor(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	qBus.Register(query.ListUsersQuery{}, &mockHandler{
		result: query.ListUsersResult{
			Users:   []*authEntity.User{},
			HasNext: true,
			Limit:   10,
		},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users?cursor=abc&limit=10", nil)

	h.List(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestList_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	qBus.Register(query.ListUsersQuery{}, &mockHandler{err: errors.New("db error")})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users", nil)

	h.List(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
}

func TestGet_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	qBus.Register(query.GetUserQuery{}, &mockHandler{
		result: &authEntity.User{
			Entity: domain.Entity{ID: id},
			Email:  "test@example.com",
			Name:   "Test User",
		},
	})

	utils.IsProduction = true
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Get(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
}

func TestGet_InvalidUUID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/invalid", nil)
	r = withChiParams(r, map[string]string{"id": "invalid"})

	h.Get(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGet_NotFound(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	qBus.Register(query.GetUserQuery{}, &mockHandler{
		err: domain.ErrNotFound,
	})

	utils.IsProduction = true
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Get(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGet_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	qBus.Register(query.GetUserQuery{}, &mockHandler{err: errors.New("db error")})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Get(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
}

func TestMe_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	userID := uuid.New()
	profileProv := &mockProfileProvider{
		profile: &public.UserProfile{
			ID:        userID.String(),
			Email:     "test@example.com",
			Name:      "Test User",
			Roles:     []string{"admin"},
			IsActive:  true,
			CreatedAt: "2024-01-01T00:00:00Z",
			UpdatedAt: "2024-01-01T00:00:00Z",
		},
	}
	h := &Handler{commandBus: cmdBus, queryBus: qBus, profileProv: profileProv}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	r = withUserID(r, userID)

	h.Me(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
}

func TestMe_Unauthorized(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/me", nil)

	h.Me(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMe_InvalidUUID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	r = r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, "not-a-uuid"))

	h.Me(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMe_ProfileError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	userID := uuid.New()
	profileProv := &mockProfileProvider{err: errors.New("profile error")}
	h := &Handler{commandBus: cmdBus, queryBus: qBus, profileProv: profileProv}

	utils.IsProduction = true
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	r = withUserID(r, userID)

	h.Me(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
}

func TestUpdate_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	cmdBus.Register(command.UpdateUserCommand{}, &mockHandler{
		result: &authEntity.User{
			Entity: domain.Entity{ID: id},
			Email:  "updated@example.com",
			Name:   "Updated User",
		},
	})

	body, _ := json.Marshal(UpdateUserRequest{Name: "Updated User", Email: "updated@example.com"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/users/"+id.String(), bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Update(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
}

func TestUpdate_InvalidUUID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	body, _ := json.Marshal(UpdateUserRequest{Name: "Test"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/users/invalid", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withChiParams(r, map[string]string{"id": "invalid"})

	h.Update(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_BadJSON(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/users/"+id.String(), bytes.NewReader([]byte("{invalid")))
	r.Header.Set("Content-Type", "application/json")
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Update(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	cmdBus.Register(command.UpdateUserCommand{}, &mockHandler{err: errors.New("db error")})

	body, _ := json.Marshal(UpdateUserRequest{Name: "Test"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/users/"+id.String(), bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Update(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
}

func TestDelete_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	cmdBus.Register(command.DeleteUserCommand{}, &mockHandler{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Delete(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDelete_InvalidUUID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/users/invalid", nil)
	r = withChiParams(r, map[string]string{"id": "invalid"})

	h.Delete(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDelete_NotFound(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	cmdBus.Register(command.DeleteUserCommand{}, &mockHandler{err: domain.ErrNotFound})

	utils.IsProduction = true
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Delete(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDelete_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus)

	id := uuid.New()
	cmdBus.Register(command.DeleteUserCommand{}, &mockHandler{err: errors.New("db error")})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Delete(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
}
