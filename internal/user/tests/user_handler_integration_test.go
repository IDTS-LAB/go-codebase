package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/user/application/query"
	userRepo "github.com/IDTS-LAB/go-codebase/internal/user/infrastructure/persistence"
	userHttp "github.com/IDTS-LAB/go-codebase/internal/user/interfaces/http"
)

type mockRoleProvider struct{}

func (m *mockRoleProvider) GetUserRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	return nil, nil
}

func setupUserHandler(t *testing.T) (*userHttp.Handler, *sql.DB) {
	t.Helper()
	repo := userRepo.NewUserRepository(db, &tenantfilter.Config{})

	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()

	cmdBus.Register(command.CreateUserCommand{}, command.NewCreateUserHandler(repo))
	cmdBus.Register(command.UpdateUserCommand{}, command.NewUpdateUserHandler(repo))
	cmdBus.Register(command.DeleteUserCommand{}, command.NewDeleteUserHandler(repo))
	qBus.Register(query.GetUserQuery{}, query.NewGetUserHandler(repo, &mockRoleProvider{}))
	qBus.Register(query.ListUsersQuery{}, query.NewListUsersHandler(repo))

	h := userHttp.NewHandler(cmdBus, qBus)
	return h, db
}

func setupRouter(h *userHttp.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/users", h.Create)
	r.Get("/users", h.List)
	r.Get("/users/{id}", h.Get)
	r.Put("/users/{id}", h.Update)
	r.Delete("/users/{id}", h.Delete)
	return r
}

type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Meta    json.RawMessage `json:"meta"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type userResponse struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	IsActive  bool     `json:"is_active"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

func TestUserHandler_CreateAndGet(t *testing.T) {
	h, _ := setupUserHandler(t)
	router := setupRouter(h)

	email := "handler-create-get-" + uuid.New().String() + "@test.com"
	body := map[string]interface{}{
		"email":     email,
		"name":      "Create Get Test",
		"is_active": true,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var createResp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	assert.NoError(t, err)
	assert.True(t, createResp.Success)

	var createdUser userResponse
	err = json.Unmarshal(createResp.Data, &createdUser)
	assert.NoError(t, err)
	assert.Equal(t, "Create Get Test", createdUser.Name)
	assert.True(t, createdUser.IsActive)

	req = httptest.NewRequest("GET", "/users/"+createdUser.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var getResp apiResponse
	err = json.Unmarshal(w.Body.Bytes(), &getResp)
	assert.NoError(t, err)
	assert.True(t, getResp.Success)

	var getUser userResponse
	err = json.Unmarshal(getResp.Data, &getUser)
	assert.NoError(t, err)
	assert.Equal(t, createdUser.ID, getUser.ID)
	assert.Equal(t, email, getUser.Email)
	assert.Equal(t, "Create Get Test", getUser.Name)
	assert.True(t, getUser.IsActive)
}

func TestUserHandler_List(t *testing.T) {
	h, _ := setupUserHandler(t)
	router := setupRouter(h)

	email1 := "handler-list-1-" + uuid.New().String() + "@test.com"
	body1 := map[string]interface{}{
		"email":     email1,
		"name":      "List One",
		"is_active": true,
	}
	bodyBytes1, _ := json.Marshal(body1)
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(bodyBytes1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	email2 := "handler-list-2-" + uuid.New().String() + "@test.com"
	body2 := map[string]interface{}{
		"email":     email2,
		"name":      "List Two",
		"is_active": true,
	}
	bodyBytes2, _ := json.Marshal(body2)
	req = httptest.NewRequest("POST", "/users", bytes.NewReader(bodyBytes2))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	req = httptest.NewRequest("GET", "/users", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResp)
	assert.NoError(t, err)
	assert.True(t, listResp.Success)

	var users []userResponse
	err = json.Unmarshal(listResp.Data, &users)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 2)

	emails := make(map[string]bool)
	for _, u := range users {
		emails[u.Email] = true
	}
	assert.True(t, emails[email1])
	assert.True(t, emails[email2])
}

func TestUserHandler_Update(t *testing.T) {
	h, _ := setupUserHandler(t)
	router := setupRouter(h)

	updateEmail := "handler-update-" + uuid.New().String() + "@test.com"
	createBody := map[string]interface{}{
		"email":     updateEmail,
		"name":      "Before Update",
		"is_active": true,
	}
	bodyBytes, _ := json.Marshal(createBody)
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var createResp apiResponse
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdUser userResponse
	json.Unmarshal(createResp.Data, &createdUser)

	updatedEmail := "handler-updated-" + uuid.New().String() + "@test.com"
	updateBody := map[string]interface{}{
		"email":     updatedEmail,
		"name":      "After Update",
		"is_active": true,
	}
	updateBytes, _ := json.Marshal(updateBody)
	req = httptest.NewRequest("PUT", "/users/"+createdUser.ID, bytes.NewReader(updateBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updateResp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &updateResp)
	assert.NoError(t, err)
	assert.True(t, updateResp.Success)

	var updatedUser userResponse
	err = json.Unmarshal(updateResp.Data, &updatedUser)
	assert.NoError(t, err)
	assert.Equal(t, updatedEmail, updatedUser.Email)
	assert.Equal(t, "After Update", updatedUser.Name)

	req = httptest.NewRequest("GET", "/users/"+createdUser.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var getResp apiResponse
	json.Unmarshal(w.Body.Bytes(), &getResp)
	var fetchedUser userResponse
	json.Unmarshal(getResp.Data, &fetchedUser)
	assert.Equal(t, updatedEmail, fetchedUser.Email)
	assert.Equal(t, "After Update", fetchedUser.Name)
}

func TestUserHandler_Delete(t *testing.T) {
	h, _ := setupUserHandler(t)
	router := setupRouter(h)

	deleteEmail := "handler-delete-" + uuid.New().String() + "@test.com"
	createBody := map[string]interface{}{
		"email":     deleteEmail,
		"name":      "Delete Test",
		"is_active": true,
	}
	bodyBytes, _ := json.Marshal(createBody)
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var createResp apiResponse
	json.Unmarshal(w.Body.Bytes(), &createResp)
	var createdUser userResponse
	json.Unmarshal(createResp.Data, &createdUser)

	req = httptest.NewRequest("DELETE", "/users/"+createdUser.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var deleteResp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &deleteResp)
	assert.NoError(t, err)
	assert.True(t, deleteResp.Success)

	req = httptest.NewRequest("GET", "/users/"+createdUser.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var getResp apiResponse
	err = json.Unmarshal(w.Body.Bytes(), &getResp)
	assert.NoError(t, err)
	assert.True(t, getResp.Success)

	var fetchedUser userResponse
	err = json.Unmarshal(getResp.Data, &fetchedUser)
	assert.NoError(t, err)
	assert.Equal(t, createdUser.ID, fetchedUser.ID)
	assert.Equal(t, deleteEmail, fetchedUser.Email)
	assert.Equal(t, "Delete Test", fetchedUser.Name)
}

func TestUserHandler_Create_Conflict(t *testing.T) {
	h, _ := setupUserHandler(t)
	router := setupRouter(h)

	conflictEmail := "handler-conflict-" + uuid.New().String() + "@test.com"
	body := map[string]interface{}{
		"email":     conflictEmail,
		"name":      "Conflict Test",
		"is_active": true,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	req = httptest.NewRequest("POST", "/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "CONFLICT", resp.Error.Code)
}

func TestUserHandler_Get_NotFound(t *testing.T) {
	h, _ := setupUserHandler(t)
	router := setupRouter(h)

	req := httptest.NewRequest("GET", "/users/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
}
