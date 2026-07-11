package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	coredomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) CreateRole(ctx context.Context, name, description string) (*entity.Role, error) {
	args := m.Called(ctx, name, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Role), args.Error(1)
}

func (m *mockService) ListRoles(ctx context.Context, page, perPage int) ([]*entity.Role, int, error) {
	args := m.Called(ctx, page, perPage)
	return args.Get(0).([]*entity.Role), args.Int(1), args.Error(2)
}

func (m *mockService) GetRole(ctx context.Context, id uuid.UUID) (*entity.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Role), args.Error(1)
}

func (m *mockService) UpdateRole(ctx context.Context, id uuid.UUID, name, description string) (*entity.Role, error) {
	args := m.Called(ctx, id, name, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Role), args.Error(1)
}

func (m *mockService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockService) CreatePermission(ctx context.Context, name, description, resource, action string) (*entity.Permission, error) {
	args := m.Called(ctx, name, description, resource, action)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Permission), args.Error(1)
}

func (m *mockService) ListPermissions(ctx context.Context, page, perPage int) ([]*entity.Permission, int, error) {
	args := m.Called(ctx, page, perPage)
	return args.Get(0).([]*entity.Permission), args.Int(1), args.Error(2)
}

func (m *mockService) GetPermission(ctx context.Context, id uuid.UUID) (*entity.Permission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Permission), args.Error(1)
}

func (m *mockService) UpdatePermission(ctx context.Context, id uuid.UUID, name, description, resource, action string) (*entity.Permission, error) {
	args := m.Called(ctx, id, name, description, resource, action)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Permission), args.Error(1)
}

func (m *mockService) DeletePermission(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockService) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *mockService) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *mockService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*entity.Role), args.Error(1)
}

func (m *mockService) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *mockService) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *mockService) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*entity.Permission, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).([]*entity.Permission), args.Error(1)
}

func (m *mockService) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	args := m.Called(ctx, userID, resource, action)
	return args.Bool(0), args.Error(1)
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
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		roleID := uuid.New()
		expected := &entity.Role{
			Entity:      coredomain.Entity{ID: roleID},
			Name:        "admin",
			Description: "Administrator",
		}

		svc.On("CreateRole", mock.Anything, "admin", "Administrator").Return(expected, nil)

		body, _ := json.Marshal(dto.CreateRoleRequest{Name: "admin", Description: "Administrator"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.CreateRole(w, r)

		assert.Equal(t, http.StatusCreated, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader([]byte(`{"name":""}`)))
		r.Header.Set("Content-Type", "application/json")

		h.CreateRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := responseMap(t, w)
		assert.False(t, resp["success"].(bool))
	})
}

func TestListRoles(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		roles := []*entity.Role{{Name: "admin"}}
		svc.On("ListRoles", mock.Anything, 1, 20).Return(roles, 1, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles", nil)

		h.ListRoles(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		assert.NotNil(t, resp["meta"])
		svc.AssertExpectations(t)
	})
}

func TestGetRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		roleID := uuid.New()
		expected := &entity.Role{
			Entity: coredomain.Entity{ID: roleID},
			Name:   "admin",
		}

		svc.On("GetRole", mock.Anything, roleID).Return(expected, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.GetRole(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})

	t.Run("invalid UUID", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles/invalid", nil)
		r = withChiParams(r, map[string]string{"id": "invalid"})

		h.GetRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		roleID := uuid.New()
		svc.On("GetRole", mock.Anything, roleID).Return(nil, coredomain.ErrNotFound)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.GetRole(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code)
		svc.AssertExpectations(t)
	})
}

func TestUpdateRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		roleID := uuid.New()
		updated := &entity.Role{
			Entity:      coredomain.Entity{ID: roleID},
			Name:        "admin",
			Description: "Updated",
		}

		svc.On("UpdateRole", mock.Anything, roleID, "admin", "Updated").Return(updated, nil)

		body, _ := json.Marshal(dto.UpdateRoleRequest{Name: "admin", Description: "Updated"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/roles/"+roleID.String(), bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.UpdateRole(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})
}

func TestDeleteRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		roleID := uuid.New()
		svc.On("DeleteRole", mock.Anything, roleID).Return(nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.DeleteRole(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})
}

func TestCreatePermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		permID := uuid.New()
		expected := &entity.Permission{
			Entity:      coredomain.Entity{ID: permID},
			Name:        "read",
			Resource:    "users",
			Action:      "read",
		}

		svc.On("CreatePermission", mock.Anything, "read", "Read users", "users", "read").Return(expected, nil)

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
		svc.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/permissions", bytes.NewReader([]byte(`{}`)))
		r.Header.Set("Content-Type", "application/json")

		h.CreatePermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := responseMap(t, w)
		assert.False(t, resp["success"].(bool))
	})
}

func TestListPermissions(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		perms := []*entity.Permission{{Name: "read", Resource: "users", Action: "read"}}
		svc.On("ListPermissions", mock.Anything, 1, 20).Return(perms, 1, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/permissions", nil)

		h.ListPermissions(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		assert.NotNil(t, resp["meta"])
		svc.AssertExpectations(t)
	})
}

func TestGetPermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		permID := uuid.New()
		expected := &entity.Permission{
			Entity:   coredomain.Entity{ID: permID},
			Name:     "read",
			Resource: "users",
			Action:   "read",
		}

		svc.On("GetPermission", mock.Anything, permID).Return(expected, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"id": permID.String()})

		h.GetPermission(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		permID := uuid.New()
		svc.On("GetPermission", mock.Anything, permID).Return(nil, coredomain.ErrNotFound)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"id": permID.String()})

		h.GetPermission(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code)
		svc.AssertExpectations(t)
	})
}

func TestUpdatePermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		permID := uuid.New()
		updated := &entity.Permission{
			Entity:      coredomain.Entity{ID: permID},
			Name:        "write",
			Resource:    "users",
			Action:      "write",
		}

		svc.On("UpdatePermission", mock.Anything, permID, "write", "Write users", "users", "write").Return(updated, nil)

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
		svc.AssertExpectations(t)
	})
}

func TestDeletePermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		permID := uuid.New()
		svc.On("DeletePermission", mock.Anything, permID).Return(nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"id": permID.String()})

		h.DeletePermission(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})
}

func TestAssignRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		userID := uuid.New()
		roleID := uuid.New()
		svc.On("AssignRoleToUser", mock.Anything, userID, roleID).Return(nil)

		body, _ := json.Marshal(dto.AssignRoleRequest{UserID: userID, RoleID: roleID})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/roles", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.AssignRole(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/users/"+uuid.New().String()+"/roles", bytes.NewReader([]byte(`{}`)))
		r.Header.Set("Content-Type", "application/json")

		h.AssignRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRemoveRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		userID := uuid.New()
		roleID := uuid.New()
		svc.On("RemoveRoleFromUser", mock.Anything, userID, roleID).Return(nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/users/"+userID.String()+"/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"userId": userID.String(), "roleId": roleID.String()})

		h.RemoveRole(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})
}

func TestGetUserRoles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		userID := uuid.New()
		roles := []*entity.Role{{Name: "admin"}}
		svc.On("GetUserRoles", mock.Anything, userID).Return(roles, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/roles", nil)
		r = withChiParams(r, map[string]string{"userId": userID.String()})

		h.GetUserRoles(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})
}

func TestAssignPermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		roleID := uuid.New()
		permID := uuid.New()
		svc.On("AssignPermissionToRole", mock.Anything, roleID, permID).Return(nil)

		body, _ := json.Marshal(dto.AssignPermissionRequest{RoleID: roleID, PermissionID: permID})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles/"+roleID.String()+"/permissions", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.AssignPermission(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles/"+uuid.New().String()+"/permissions", bytes.NewReader([]byte(`{}`)))
		r.Header.Set("Content-Type", "application/json")

		h.AssignPermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRemovePermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		roleID := uuid.New()
		permID := uuid.New()
		svc.On("RemovePermissionFromRole", mock.Anything, roleID, permID).Return(nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/roles/"+roleID.String()+"/permissions/"+permID.String(), nil)
		r = withChiParams(r, map[string]string{"roleId": roleID.String(), "permissionId": permID.String()})

		h.RemovePermission(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})
}

func TestGetRolePermissions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		roleID := uuid.New()
		perms := []*entity.Permission{{Name: "read", Resource: "users", Action: "read"}}
		svc.On("GetRolePermissions", mock.Anything, roleID).Return(perms, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/roles/"+roleID.String()+"/permissions", nil)
		r = withChiParams(r, map[string]string{"roleId": roleID.String()})

		h.GetRolePermissions(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})
}

func TestCheckPermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		userID := uuid.New()
		svc.On("CheckPermission", mock.Anything, userID, "users", "read").Return(true, nil)

		body, _ := json.Marshal(dto.CheckPermissionRequest{Resource: "users", Action: "read"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/check-permission", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = withUserID(r, userID)

		h.CheckPermission(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := responseMap(t, w)
		assert.True(t, resp["success"].(bool))
		svc.AssertExpectations(t)
	})

	t.Run("unauthorized", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		body, _ := json.Marshal(dto.CheckPermissionRequest{Resource: "users", Action: "read"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/check-permission", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		h.CheckPermission(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		svc.AssertNotCalled(t, "CheckPermission")
	})

	t.Run("error from invalid context userID", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		body, _ := json.Marshal(dto.CheckPermissionRequest{Resource: "users", Action: "read"})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/check-permission", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r = r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, "not-a-uuid"))

		h.CheckPermission(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandlerErrors(t *testing.T) {
	t.Run("create role bad JSON", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader([]byte(`{invalid json`)))
		r.Header.Set("Content-Type", "application/json")

		h.CreateRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update role bad JSON", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		roleID := uuid.New()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/roles/"+roleID.String(), bytes.NewReader([]byte(`{invalid}`)))
		r.Header.Set("Content-Type", "application/json")
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.UpdateRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update role invalid UUID", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/roles/invalid", bytes.NewReader([]byte(`{"name":"admin"}`)))
		r.Header.Set("Content-Type", "application/json")
		r = withChiParams(r, map[string]string{"id": "invalid"})

		h.UpdateRole(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete role error", func(t *testing.T) {
		svc := new(mockService)
		v := validator.New()
		h := &Handler{svc: svc, validator: v}

		roleID := uuid.New()
		svc.On("DeleteRole", mock.Anything, roleID).Return(errors.New("db error"))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/roles/"+roleID.String(), nil)
		r = withChiParams(r, map[string]string{"id": roleID.String()})

		h.DeleteRole(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		svc.AssertExpectations(t)
	})
}
