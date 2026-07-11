package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) List(ctx context.Context, offset, limit int) ([]*entity.User, int, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*entity.User), args.Int(1), args.Error(2)
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *mockUserRepo) Update(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func makeTestUser(id uuid.UUID) *entity.User {
	u := entity.NewUser("john@example.com", "hashed", "John Doe")
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
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	id1 := uuid.New()
	id2 := uuid.New()
	users := []*entity.User{makeTestUser(id1), makeTestUser(id2)}
	repo.On("List", mock.Anything, 0, 20).Return(users, 2, nil)

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

	repo.AssertExpectations(t)
}

func TestHandler_List_WithPagination(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	repo.On("List", mock.Anything, 10, 5).Return([]*entity.User{}, 0, nil)

	req := httptest.NewRequest(http.MethodGet, "/users?offset=10&limit=5", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	repo.AssertExpectations(t)
}

func TestHandler_List_ClampsLimit(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	// limit > 100 should be clamped to 20
	repo.On("List", mock.Anything, 0, 20).Return([]*entity.User{}, 0, nil)

	req := httptest.NewRequest(http.MethodGet, "/users?limit=200", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	repo.AssertExpectations(t)
}

func TestHandler_Get_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	id := uuid.New()
	user := makeTestUser(id)
	repo.On("GetByID", mock.Anything, id).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)

	repo.AssertExpectations(t)
}

func TestHandler_Get_InvalidUUID(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

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
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	repo.AssertExpectations(t)
}

func TestHandler_Get_ServiceError(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, errors.New("unexpected error"))

	req := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Get(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	repo.AssertExpectations(t)
}

func TestHandler_Me_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	id := uuid.New()
	user := makeTestUser(id)
	repo.On("GetByID", mock.Anything, id).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = setUserIDOnCtx(req, id.String())
	w := httptest.NewRecorder()
	h.Me(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)

	repo.AssertExpectations(t)
}

func TestHandler_Me_Unauthenticated(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

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
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = setUserIDOnCtx(req, "not-a-valid-uuid")
	w := httptest.NewRecorder()
	h.Me(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	id := uuid.New()
	user := makeTestUser(id)
	repo.On("GetByID", mock.Anything, id).Return(user, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == id && u.Name == "Jane Doe" && u.Email == "jane@example.com" && !u.IsActive
	})).Return(nil)

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

	repo.AssertExpectations(t)
}

func TestHandler_Update_InvalidUUID(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPut, "/users/invalid", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "id", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_InvalidBody(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/users/"+id.String(), strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	body := `{"name":"Jane Doe"}`
	req := httptest.NewRequest(http.MethodPut, "/users/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	repo.AssertExpectations(t)
}

func TestHandler_Delete_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	id := uuid.New()
	user := makeTestUser(id)
	repo.On("GetByID", mock.Anything, id).Return(user, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == id && u.IsDeleted()
	})).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "user deleted", resp.Data.(map[string]interface{})["message"])

	repo.AssertExpectations(t)
}

func TestHandler_Delete_InvalidUUID(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/users/invalid", nil)
	req = setChiURLParam(req, "id", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Delete_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	svc := service.NewUserService(repo)
	h := NewHandler(svc)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	req = setChiURLParam(req, "id", id.String())
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	repo.AssertExpectations(t)
}
