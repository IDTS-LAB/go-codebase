package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/query"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	coredomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHandler struct {
	result any
	err    error
}

func (h *mockHandler) Handle(ctx context.Context, _ any) (any, error) {
	return h.result, h.err
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

func responseMap(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func TestCreateRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		expected := &entity.Role{
			Entity:      coredomain.Entity{ID: roleID},
			Name:        "admin",
			Description: "Administrator",
		}
		cmdBus.Register(command.CreateRoleCommand{}, &mockHandler{result: expected})

		body, _ := json.Marshal(dto.CreateRoleRequest{Name: "admin", Description: "Administrator"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.CreateRole(w, r)

		assert.Equal(t, http.StatusCreated, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("validation error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader([]byte(`{"name":""}`)))
		r.Header.Set("Content-Type", "application/json")

		h.CreateRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := responseMap(t, w)
		assert.False(t, resp["success"].(bool))
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		cmdBus.Register(command.CreateRoleCommand{}, &mockHandler{err: errors.New("db error")})

		body, _ := json.Marshal(dto.CreateRoleRequest{Name: "admin", Description: "Administrator"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.CreateRole(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestListRoles(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roles := query.ListRolesResult{
			Roles: []*entity.Role{{Name: "admin"}},
		}
		qBus.Register(query.ListRolesQuery{}, &mockHandler{result: roles})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles", nil)

		h.ListRoles(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		assert.NotNil(t, resp["meta"])
	})
}

func TestGetRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		expected := &entity.Role{
			Entity: coredomain.Entity{ID: roleID},
			Name:   "admin",
		}
		qBus.Register(query.GetRoleQuery{}, &mockHandler{result: expected})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.GetRole(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("invalid UUID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles/invalid", nil)
		r = withChiParams(r, map[string]string{"id": "invalid"})

		h.GetRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		qBus.Register(query.GetRoleQuery{}, &mockHandler{result: nil, err: coredomain.ErrNotFound})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.GetRole(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		qBus.Register(query.GetRoleQuery{}, &mockHandler{err: errors.New("db error")})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.GetRole(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUpdateRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		updated := &entity.Role{
			Entity:      coredomain.Entity{ID: roleID},
			Name:        "admin",
			Description: "Updated",
		}
		cmdBus.Register(command.UpdateRoleCommand{}, &mockHandler{result: updated})

		body, _ := json.Marshal(dto.UpdateRoleRequest{Name: "admin", Description: "Updated"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/roles/"+roleID.String(), bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.UpdateRole(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})
}

func TestDeleteRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		cmdBus.Register(command.DeleteRoleCommand{}, &mockHandler{})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.DeleteRole(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})
}

func TestCreatePermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		permID := uuid.New()
		expected := &entity.Permission{
			Entity:   coredomain.Entity{ID: permID},
			Name:     "read",
			Resource: "users",
			Action:   "read",
		}
		cmdBus.Register(command.CreatePermissionCommand{}, &mockHandler{result: expected})

		body, _ := json.Marshal(dto.CreatePermissionRequest{
			Name: "read", Description: "Read users", Resource: "users", Action: "read",
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/permissions", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.CreatePermission(w, r)

		assert.Equal(t, http.StatusCreated, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("validation error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/permissions", bytes.NewReader([]byte(`{}`)))
		r.Header.Set("Content-Type", "application/json")

		h.CreatePermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := responseMap(t, w)
		assert.False(t, resp["success"].(bool))
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		cmdBus.Register(command.CreatePermissionCommand{}, &mockHandler{err: errors.New("db error")})

		body, _ := json.Marshal(dto.CreatePermissionRequest{
			Name: "read", Description: "Read users", Resource: "users", Action: "read",
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/permissions", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.CreatePermission(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestListPermissions(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		perms := query.ListPermissionsResult{
			Permissions: []*entity.Permission{{Name: "read", Resource: "users", Action: "read"}},
		}
		qBus.Register(query.ListPermissionsQuery{}, &mockHandler{result: perms})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/permissions", nil)

		h.ListPermissions(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		assert.NotNil(t, resp["meta"])
	})
}

func TestGetPermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		permID := uuid.New()
		expected := &entity.Permission{
			Entity:   coredomain.Entity{ID: permID},
			Name:     "read",
			Resource: "users",
			Action:   "read",
		}
		qBus.Register(query.GetPermissionQuery{}, &mockHandler{result: expected})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"id": permID.String()})

		h.GetPermission(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("not found", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		permID := uuid.New()
		qBus.Register(query.GetPermissionQuery{}, &mockHandler{result: nil, err: coredomain.ErrNotFound})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"id": permID.String()})

		h.GetPermission(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		permID := uuid.New()
		qBus.Register(query.GetPermissionQuery{}, &mockHandler{err: errors.New("db error")})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"id": permID.String()})

		h.GetPermission(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUpdatePermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		permID := uuid.New()
		updated := &entity.Permission{
			Entity:   coredomain.Entity{ID: permID},
			Name:     "write",
			Resource: "users",
			Action:   "write",
		}
		cmdBus.Register(command.UpdatePermissionCommand{}, &mockHandler{result: updated})

		body, _ := json.Marshal(dto.UpdatePermissionRequest{
			Name: "write", Description: "Write users", Resource: "users", Action: "write",
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/permissions/"+permID.String(), bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withChiParams(r, map[string]string{"id": permID.String()})

		h.UpdatePermission(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		permID := uuid.New()
		cmdBus.Register(command.UpdatePermissionCommand{}, &mockHandler{err: errors.New("db error")})

		body, _ := json.Marshal(dto.UpdatePermissionRequest{
			Name: "write", Description: "Write users", Resource: "users", Action: "write",
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/permissions/"+permID.String(), bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withChiParams(r, map[string]string{"id": permID.String()})

		h.UpdatePermission(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestDeletePermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		permID := uuid.New()
		cmdBus.Register(command.DeletePermissionCommand{}, &mockHandler{})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"id": permID.String()})

		h.DeletePermission(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})
}

func TestAssignRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		userID := uuid.New()
		roleID := uuid.New()
		cmdBus.Register(command.AssignRoleCommand{}, &mockHandler{})

		body, _ := json.Marshal(dto.AssignRoleRequest{UserID: userID, RoleID: roleID})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/roles", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.AssignRole(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("validation error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/users/"+uuid.New().String()+"/roles", bytes.NewReader([]byte(`{}`)))
		r.Header.Set("Content-Type", "application/json")

		h.AssignRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		cmdBus.Register(command.AssignRoleCommand{}, &mockHandler{err: errors.New("db error")})

		body, _ := json.Marshal(dto.AssignRoleRequest{UserID: uuid.New(), RoleID: uuid.New()})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/users/"+uuid.New().String()+"/roles", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.AssignRole(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRemoveRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		userID := uuid.New()
		roleID := uuid.New()
		cmdBus.Register(command.UnassignRoleCommand{}, &mockHandler{})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/users/"+userID.String()+"/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"userId": userID.String(), "roleId": roleID.String()})

		h.RemoveRole(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		userID := uuid.New()
		roleID := uuid.New()
		cmdBus.Register(command.UnassignRoleCommand{}, &mockHandler{err: errors.New("db error")})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/users/"+userID.String()+"/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"userId": userID.String(), "roleId": roleID.String()})

		h.RemoveRole(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGetUserRoles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		userID := uuid.New()
		roles := []*entity.Role{{Name: "admin"}}
		qBus.Register(query.GetUserRolesQuery{}, &mockHandler{result: roles})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/roles", nil)
		r = withChiParams(r, map[string]string{"userId": userID.String()})

		h.GetUserRoles(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		userID := uuid.New()
		qBus.Register(query.GetUserRolesQuery{}, &mockHandler{err: errors.New("db error")})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/roles", nil)
		r = withChiParams(r, map[string]string{"userId": userID.String()})

		h.GetUserRoles(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAssignPermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		permID := uuid.New()
		cmdBus.Register(command.AssignPermissionCommand{}, &mockHandler{})

		body, _ := json.Marshal(dto.AssignPermissionRequest{RoleID: roleID, PermissionID: permID})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles/"+roleID.String()+"/permissions", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.AssignPermission(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("validation error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles/"+uuid.New().String()+"/permissions", bytes.NewReader([]byte(`{}`)))
		r.Header.Set("Content-Type", "application/json")

		h.AssignPermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		cmdBus.Register(command.AssignPermissionCommand{}, &mockHandler{err: errors.New("db error")})

		body, _ := json.Marshal(dto.AssignPermissionRequest{RoleID: uuid.New(), PermissionID: uuid.New()})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles/"+uuid.New().String()+"/permissions", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.AssignPermission(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRemovePermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		permID := uuid.New()
		cmdBus.Register(command.UnassignPermissionCommand{}, &mockHandler{})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/roles/"+roleID.String()+"/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"roleId": roleID.String(), "permissionId": permID.String()})

		h.RemovePermission(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		permID := uuid.New()
		cmdBus.Register(command.UnassignPermissionCommand{}, &mockHandler{err: errors.New("db error")})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/roles/"+roleID.String()+"/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"roleId": roleID.String(), "permissionId": permID.String()})

		h.RemovePermission(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGetRolePermissions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		perms := []*entity.Permission{{Name: "read", Resource: "users", Action: "read"}}
		qBus.Register(query.GetRolePermissionsQuery{}, &mockHandler{result: perms})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles/"+roleID.String()+"/permissions", nil)
		r = withChiParams(r, map[string]string{"roleId": roleID.String()})

		h.GetRolePermissions(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		qBus.Register(query.GetRolePermissionsQuery{}, &mockHandler{err: errors.New("db error")})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles/"+roleID.String()+"/permissions", nil)
		r = withChiParams(r, map[string]string{"roleId": roleID.String()})

		h.GetRolePermissions(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCheckPermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		userID := uuid.New()
		qBus.Register(query.CheckPermissionQuery{}, &mockHandler{result: true})

		body, _ := json.Marshal(dto.CheckPermissionRequest{Resource: "users", Action: "read"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/check-permission", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withUserID(r, userID)

		h.CheckPermission(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("unauthorized", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		body, _ := json.Marshal(dto.CheckPermissionRequest{Resource: "users", Action: "read"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/check-permission", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.CheckPermission(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error from invalid context userID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		body, _ := json.Marshal(dto.CheckPermissionRequest{Resource: "users", Action: "read"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/check-permission", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, "not-a-uuid"))

		h.CheckPermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bus error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		userID := uuid.New()
		qBus.Register(query.CheckPermissionQuery{}, &mockHandler{err: errors.New("db error")})

		body, _ := json.Marshal(dto.CheckPermissionRequest{Resource: "users", Action: "read"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/check-permission", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withUserID(r, userID)

		h.CheckPermission(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandlerErrors(t *testing.T) {
	t.Run("create role bad JSON", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader([]byte(`{invalid json`)))
		r.Header.Set("Content-Type", "application/json")

		h.CreateRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update role bad JSON", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/roles/"+roleID.String(), bytes.NewReader([]byte(`{invalid}`)))
		r.Header.Set("Content-Type", "application/json")
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.UpdateRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update role invalid UUID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/roles/invalid", bytes.NewReader([]byte(`{"name":"admin"}`)))
		r.Header.Set("Content-Type", "application/json")
		r = withChiParams(r, map[string]string{"id": "invalid"})

		h.UpdateRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete role error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		cmdBus.Register(command.DeleteRoleCommand{}, &mockHandler{err: errors.New("db error")})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.DeleteRole(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("delete role invalid UUID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/roles/invalid", nil)
		r = withChiParams(r, map[string]string{"id": "invalid"})

		h.DeleteRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update role not found", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		cmdBus.Register(command.UpdateRoleCommand{}, &mockHandler{err: errors.New("not found")})

		body, _ := json.Marshal(dto.UpdateRoleRequest{Name: "admin"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/roles/"+roleID.String(), bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.UpdateRole(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})



	t.Run("create permission bad JSON", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/permissions", bytes.NewReader([]byte(`{invalid json`)))
		r.Header.Set("Content-Type", "application/json")

		h.CreatePermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update permission invalid UUID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/permissions/invalid", bytes.NewReader([]byte(`{"name":"read"}`)))
		r.Header.Set("Content-Type", "application/json")
		r = withChiParams(r, map[string]string{"id": "invalid"})

		h.UpdatePermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update permission bad JSON", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		permID := uuid.New()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/permissions/"+permID.String(), bytes.NewReader([]byte(`{invalid}`)))
		r.Header.Set("Content-Type", "application/json")
		r = withChiParams(r, map[string]string{"id": permID.String()})

		h.UpdatePermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete permission invalid UUID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/permissions/invalid", nil)
		r = withChiParams(r, map[string]string{"id": "invalid"})

		h.DeletePermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete permission error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		permID := uuid.New()
		cmdBus.Register(command.DeletePermissionCommand{}, &mockHandler{err: errors.New("db error")})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"id": permID.String()})

		h.DeletePermission(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("remove role invalid user ID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/users/invalid/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"userId": "invalid", "roleId": roleID.String()})

		h.RemoveRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("remove role invalid role ID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		userID := uuid.New()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/users/"+userID.String()+"/roles/invalid", nil)
		r = withChiParams(r, map[string]string{"userId": userID.String(), "roleId": "invalid"})

		h.RemoveRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get user roles invalid user ID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/users/invalid/roles", nil)
		r = withChiParams(r, map[string]string{"userId": "invalid"})

		h.GetUserRoles(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("remove permission invalid role ID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		permID := uuid.New()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/roles/invalid/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"roleId": "invalid", "permissionId": permID.String()})

		h.RemovePermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("remove permission invalid permission ID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		roleID := uuid.New()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/roles/"+roleID.String()+"/permissions/invalid", nil)
		r = withChiParams(r, map[string]string{"roleId": roleID.String(), "permissionId": "invalid"})

		h.RemovePermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get role permissions invalid role ID", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles/invalid/permissions", nil)
		r = withChiParams(r, map[string]string{"roleId": "invalid"})

		h.GetRolePermissions(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("list roles error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		qBus.Register(query.ListRolesQuery{}, &mockHandler{err: errors.New("db error")})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles", nil)

		h.ListRoles(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("list permissions error", func(t *testing.T) {
		cmdBus := cqrs.NewInMemoryCommandBus()
		qBus := cqrs.NewInMemoryQueryBus()
		v := validator.New()
		h := NewHandler(cmdBus, qBus, v)

		qBus.Register(query.ListPermissionsQuery{}, &mockHandler{err: errors.New("db error")})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/permissions", nil)

		h.ListPermissions(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
