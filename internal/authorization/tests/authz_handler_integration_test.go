package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authzHttp "github.com/IDTS-LAB/go-codebase/internal/authorization/interfaces/http"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/application/query"
	authzRepo "github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/middleware"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
)

type mockEnforcer struct{}

func (m *mockEnforcer) ReloadPolicies(ctx context.Context) error               { return nil }
func (m *mockEnforcer) ReloadUserPolicies(ctx context.Context, userID uuid.UUID) error { return nil }
func (m *mockEnforcer) Enforce(userID uuid.UUID, resource, action string) (bool, error) {
	return true, nil
}

type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Meta    json.RawMessage `json:"meta,omitempty"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func setupHandler() *authzHttp.Handler {
	roleRepo := authzRepo.NewRoleRepository(db, &tenantfilter.Config{})
	permRepo := authzRepo.NewPermissionRepository(db, &tenantfilter.Config{})
	userRoleRepo := authzRepo.NewUserRoleRepository(db)
	rolePermRepo := authzRepo.NewRolePermissionRepository(db)
	enf := &mockEnforcer{}

	cmdBus := cqrs.NewInMemoryCommandBus()
	cmdBus.Register(command.CreateRoleCommand{}, command.NewCreateRoleHandler(roleRepo))
	cmdBus.Register(command.UpdateRoleCommand{}, command.NewUpdateRoleHandler(roleRepo))
	cmdBus.Register(command.DeleteRoleCommand{}, command.NewDeleteRoleHandler(roleRepo))
	cmdBus.Register(command.CreatePermissionCommand{}, command.NewCreatePermissionHandler(permRepo))
	cmdBus.Register(command.UpdatePermissionCommand{}, command.NewUpdatePermissionHandler(permRepo))
	cmdBus.Register(command.DeletePermissionCommand{}, command.NewDeletePermissionHandler(permRepo))
	cmdBus.Register(command.AssignRoleCommand{}, command.NewAssignRoleHandler(roleRepo, userRoleRepo, enf))
	cmdBus.Register(command.UnassignRoleCommand{}, command.NewUnassignRoleHandler(userRoleRepo, enf))
	cmdBus.Register(command.AssignPermissionCommand{}, command.NewAssignPermissionHandler(roleRepo, permRepo, rolePermRepo, enf))
	cmdBus.Register(command.UnassignPermissionCommand{}, command.NewUnassignPermissionHandler(rolePermRepo, enf))

	queryBus := cqrs.NewInMemoryQueryBus()
	queryBus.Register(query.GetRoleQuery{}, query.NewGetRoleHandler(roleRepo))
	queryBus.Register(query.ListRolesQuery{}, query.NewListRolesHandler(roleRepo))
	queryBus.Register(query.GetPermissionQuery{}, query.NewGetPermissionHandler(permRepo))
	queryBus.Register(query.ListPermissionsQuery{}, query.NewListPermissionsHandler(permRepo))
	queryBus.Register(query.GetUserRolesQuery{}, query.NewGetUserRolesHandler(userRoleRepo))
	queryBus.Register(query.GetRolePermissionsQuery{}, query.NewGetRolePermissionsHandler(rolePermRepo))
	queryBus.Register(query.CheckPermissionQuery{}, query.NewCheckPermissionHandler(enf))

	v := validator.New()
	return authzHttp.NewHandler(cmdBus, queryBus, v)
}

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func withUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func decodeResponse(t *testing.T, body []byte) apiResponse {
	t.Helper()
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, string(body))
	}
	return resp
}

func TestHandler_CreateAndGetRole(t *testing.T) {
	h := setupHandler()
	roleName := fmt.Sprintf("h-create-get-%s", uuid.New().String())

	body := fmt.Sprintf(`{"name":"%s","description":"Test role desc"}`, roleName)
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/roles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateRole(w, req)
	resp := decodeResponse(t, w.Body.Bytes())

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, string(w.Body.Bytes()))
	}
	if !resp.Success {
		t.Fatal("expected success")
	}

	var createdRole struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(resp.Data, &createdRole); err != nil {
		t.Fatalf("unmarshal role: %v", err)
	}
	if createdRole.Name != roleName {
		t.Errorf("expected name %q, got %q", roleName, createdRole.Name)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/auth/sessions/roles/"+createdRole.ID, nil)
	getReq = withChiParams(getReq, map[string]string{"id": createdRole.ID})
	w2 := httptest.NewRecorder()
	h.GetRole(w2, getReq)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, string(w2.Body.Bytes()))
	}
	var gotRole struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	resp2 := decodeResponse(t, w2.Body.Bytes())
	if err := json.Unmarshal(resp2.Data, &gotRole); err != nil {
		t.Fatalf("unmarshal role: %v", err)
	}
	if gotRole.Name != roleName {
		t.Errorf("expected name %q, got %q", roleName, gotRole.Name)
	}
}

func TestHandler_CreateAndListRoles(t *testing.T) {
	h := setupHandler()
	roleName := fmt.Sprintf("h-list-%s", uuid.New().String())

	body := fmt.Sprintf(`{"name":"%s"}`, roleName)
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/roles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateRole(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/auth/sessions/roles", nil)
	w2 := httptest.NewRecorder()
	h.ListRoles(w2, listReq)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, string(w2.Body.Bytes()))
	}

	resp := decodeResponse(t, w2.Body.Bytes())
	var roles []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(resp.Data, &roles); err != nil {
		t.Fatalf("unmarshal roles: %v", err)
	}
	var found bool
	for _, r := range roles {
		if r.Name == roleName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("role %q not found in list", roleName)
	}
}

func TestHandler_CreateUpdateAndGetRole(t *testing.T) {
	h := setupHandler()
	roleName := fmt.Sprintf("h-upd-%s", uuid.New().String())

	body := fmt.Sprintf(`{"name":"%s","description":"Original"}`, roleName)
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/roles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateRole(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w.Body.Bytes())
	var created struct {
		ID string `json:"id"`
	}
	json.Unmarshal(resp.Data, &created)

	updatedName := roleName + "-updated"
	updateBody := fmt.Sprintf(`{"name":"%s","description":"Updated desc"}`, updatedName)
	putReq := httptest.NewRequest(http.MethodPut, "/auth/sessions/roles/"+created.ID, bytes.NewReader([]byte(updateBody)))
	putReq.Header.Set("Content-Type", "application/json")
	putReq = withChiParams(putReq, map[string]string{"id": created.ID})
	w2 := httptest.NewRecorder()
	h.UpdateRole(w2, putReq)

	if w2.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", w2.Code, string(w2.Body.Bytes()))
	}

	getReq := httptest.NewRequest(http.MethodGet, "/auth/sessions/roles/"+created.ID, nil)
	getReq = withChiParams(getReq, map[string]string{"id": created.ID})
	w3 := httptest.NewRecorder()
	h.GetRole(w3, getReq)
	if w3.Code != http.StatusOK {
		t.Fatalf("get after update: %d %s", w3.Code, w3.Body.String())
	}

	resp3 := decodeResponse(t, w3.Body.Bytes())
	var updated struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	json.Unmarshal(resp3.Data, &updated)
	if updated.Name != updatedName {
		t.Errorf("expected name %q, got %q", updatedName, updated.Name)
	}
	if updated.Description != "Updated desc" {
		t.Errorf("expected desc 'Updated desc', got %q", updated.Description)
	}
}

func TestHandler_CreateDeleteAndGetRole(t *testing.T) {
	h := setupHandler()
	roleName := fmt.Sprintf("h-del-%s", uuid.New().String())

	body := fmt.Sprintf(`{"name":"%s"}`, roleName)
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/roles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateRole(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w.Body.Bytes())
	var created struct {
		ID string `json:"id"`
	}
	json.Unmarshal(resp.Data, &created)

	delReq := httptest.NewRequest(http.MethodDelete, "/auth/sessions/roles/"+created.ID, nil)
	delReq = withChiParams(delReq, map[string]string{"id": created.ID})
	w2 := httptest.NewRecorder()
	h.DeleteRole(w2, delReq)
	if w2.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", w2.Code, string(w2.Body.Bytes()))
	}

	getReq := httptest.NewRequest(http.MethodGet, "/auth/sessions/roles/"+created.ID, nil)
	getReq = withChiParams(getReq, map[string]string{"id": created.ID})
	w3 := httptest.NewRecorder()
	h.GetRole(w3, getReq)
	resp3 := decodeResponse(t, w3.Body.Bytes())
	if resp3.Success {
		t.Error("expected error after delete, got success")
	}
}

func TestHandler_CreateAndGetPermission(t *testing.T) {
	h := setupHandler()
	permName := fmt.Sprintf("h-perm-get-%s", uuid.New().String())

	body := fmt.Sprintf(`{"name":"%s","resource":"users","action":"read"}`, permName)
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/permissions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreatePermission(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w.Body.Bytes())
	var created struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Resource string `json:"resource"`
		Action   string `json:"action"`
	}
	json.Unmarshal(resp.Data, &created)
	if created.Name != permName {
		t.Errorf("expected name %q, got %q", permName, created.Name)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/auth/sessions/permissions/"+created.ID, nil)
	getReq = withChiParams(getReq, map[string]string{"id": created.ID})
	w2 := httptest.NewRecorder()
	h.GetPermission(w2, getReq)
	if w2.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w2.Code, w2.Body.String())
	}
	resp2 := decodeResponse(t, w2.Body.Bytes())
	var got struct {
		Name string `json:"name"`
	}
	json.Unmarshal(resp2.Data, &got)
	if got.Name != permName {
		t.Errorf("expected name %q, got %q", permName, got.Name)
	}
}

func TestHandler_CreateAndListPermissions(t *testing.T) {
	h := setupHandler()
	permName := fmt.Sprintf("h-perm-list-%s", uuid.New().String())

	body := fmt.Sprintf(`{"name":"%s","resource":"users","action":"read"}`, permName)
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/permissions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreatePermission(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/auth/sessions/permissions", nil)
	w2 := httptest.NewRecorder()
	h.ListPermissions(w2, listReq)
	if w2.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w2.Code, w2.Body.String())
	}

	resp := decodeResponse(t, w2.Body.Bytes())
	var perms []struct {
		Name string `json:"name"`
	}
	json.Unmarshal(resp.Data, &perms)
	var found bool
	for _, p := range perms {
		if p.Name == permName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("permission %q not found in list", permName)
	}
}

func TestHandler_CreateUpdateAndGetPermission(t *testing.T) {
	h := setupHandler()
	permName := fmt.Sprintf("h-perm-upd-%s", uuid.New().String())

	body := fmt.Sprintf(`{"name":"%s","resource":"users","action":"read"}`, permName)
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/permissions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreatePermission(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w.Body.Bytes())
	var created struct {
		ID string `json:"id"`
	}
	json.Unmarshal(resp.Data, &created)

	updatedName := permName + "-v2"
	updBody := fmt.Sprintf(`{"name":"%s","resource":"users","action":"write"}`, updatedName)
	putReq := httptest.NewRequest(http.MethodPut, "/auth/sessions/permissions/"+created.ID, bytes.NewReader([]byte(updBody)))
	putReq.Header.Set("Content-Type", "application/json")
	putReq = withChiParams(putReq, map[string]string{"id": created.ID})
	w2 := httptest.NewRecorder()
	h.UpdatePermission(w2, putReq)
	if w2.Code != http.StatusOK {
		t.Fatalf("update: %d %s", w2.Code, w2.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/auth/sessions/permissions/"+created.ID, nil)
	getReq = withChiParams(getReq, map[string]string{"id": created.ID})
	w3 := httptest.NewRecorder()
	h.GetPermission(w3, getReq)
	if w3.Code != http.StatusOK {
		t.Fatalf("get after update: %d %s", w3.Code, w3.Body.String())
	}

	resp3 := decodeResponse(t, w3.Body.Bytes())
	var updated struct {
		Name     string `json:"name"`
		Action   string `json:"action"`
	}
	json.Unmarshal(resp3.Data, &updated)
	if updated.Name != updatedName {
		t.Errorf("expected name %q, got %q", updatedName, updated.Name)
	}
	if updated.Action != "write" {
		t.Errorf("expected action 'write', got %q", updated.Action)
	}
}

func TestHandler_CreateDeleteAndGetPermission(t *testing.T) {
	h := setupHandler()
	permName := fmt.Sprintf("h-perm-del-%s", uuid.New().String())

	body := fmt.Sprintf(`{"name":"%s","resource":"users","action":"read"}`, permName)
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/permissions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreatePermission(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w.Body.Bytes())
	var created struct {
		ID string `json:"id"`
	}
	json.Unmarshal(resp.Data, &created)

	delReq := httptest.NewRequest(http.MethodDelete, "/auth/sessions/permissions/"+created.ID, nil)
	delReq = withChiParams(delReq, map[string]string{"id": created.ID})
	w2 := httptest.NewRecorder()
	h.DeletePermission(w2, delReq)
	if w2.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", w2.Code, w2.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/auth/sessions/permissions/"+created.ID, nil)
	getReq = withChiParams(getReq, map[string]string{"id": created.ID})
	w3 := httptest.NewRecorder()
	h.GetPermission(w3, getReq)
	resp3 := decodeResponse(t, w3.Body.Bytes())
	if resp3.Success {
		t.Error("expected error after delete, got success")
	}
}

func TestHandler_AssignAndGetRolePermissions(t *testing.T) {
	h := setupHandler()
	roleName := fmt.Sprintf("h-rp-%s", uuid.New().String())
	permName := fmt.Sprintf("h-rp-perm-%s", uuid.New().String())

	roleBody := fmt.Sprintf(`{"name":"%s"}`, roleName)
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/roles", bytes.NewReader([]byte(roleBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateRole(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create role: %d %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w.Body.Bytes())
	var role struct {
		ID string `json:"id"`
		Name string `json:"name"`
	}
	json.Unmarshal(resp.Data, &role)

	permBody := fmt.Sprintf(`{"name":"%s","resource":"docs","action":"read"}`, permName)
	req2 := httptest.NewRequest(http.MethodPost, "/auth/sessions/permissions", bytes.NewReader([]byte(permBody)))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.CreatePermission(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("create perm: %d %s", w2.Code, w2.Body.String())
	}
	resp2 := decodeResponse(t, w2.Body.Bytes())
	var perm struct {
		ID string `json:"id"`
	}
	json.Unmarshal(resp2.Data, &perm)

	assignBody := fmt.Sprintf(`{"role_id":"%s","permission_id":"%s"}`, role.ID, perm.ID)
	assignReq := httptest.NewRequest(http.MethodPost, "/auth/sessions/roles/"+role.ID+"/permissions", bytes.NewReader([]byte(assignBody)))
	assignReq.Header.Set("Content-Type", "application/json")
	assignReq = withChiParams(assignReq, map[string]string{"roleId": role.ID})
	w3 := httptest.NewRecorder()
	h.AssignPermission(w3, assignReq)
	if w3.Code != http.StatusOK {
		t.Fatalf("assign perm: %d %s", w3.Code, w3.Body.String())
	}

	getPermsReq := httptest.NewRequest(http.MethodGet, "/auth/sessions/roles/"+role.ID+"/permissions", nil)
	getPermsReq = withChiParams(getPermsReq, map[string]string{"roleId": role.ID})
	w4 := httptest.NewRecorder()
	h.GetRolePermissions(w4, getPermsReq)
	if w4.Code != http.StatusOK {
		t.Fatalf("get role perms: %d %s", w4.Code, w4.Body.String())
	}

	resp4 := decodeResponse(t, w4.Body.Bytes())
	var perms []struct {
		Name string `json:"name"`
	}
	json.Unmarshal(resp4.Data, &perms)
	var found bool
	for _, p := range perms {
		if p.Name == permName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("permission %q not found in role permissions", permName)
	}
}

func TestHandler_AssignAndRemoveUserRole(t *testing.T) {
	h := setupHandler()
	roleName := fmt.Sprintf("h-ur-%s", uuid.New().String())

	roleBody := fmt.Sprintf(`{"name":"%s"}`, roleName)
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/roles", bytes.NewReader([]byte(roleBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateRole(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create role: %d %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w.Body.Bytes())
	var role struct {
		ID string `json:"id"`
	}
	json.Unmarshal(resp.Data, &role)

	userID := uuid.New()
	assignBody := fmt.Sprintf(`{"user_id":"%s","role_id":"%s"}`, userID.String(), role.ID)
	assignReq := httptest.NewRequest(http.MethodPost, "/auth/sessions/users/"+userID.String()+"/roles", bytes.NewReader([]byte(assignBody)))
	assignReq.Header.Set("Content-Type", "application/json")
	assignReq = withChiParams(assignReq, map[string]string{"userId": userID.String()})
	w2 := httptest.NewRecorder()
	h.AssignRole(w2, assignReq)
	if w2.Code != http.StatusOK {
		t.Fatalf("assign role: %d %s", w2.Code, w2.Body.String())
	}

	getURReq := httptest.NewRequest(http.MethodGet, "/auth/sessions/users/"+userID.String()+"/roles", nil)
	getURReq = withChiParams(getURReq, map[string]string{"userId": userID.String()})
	w3 := httptest.NewRecorder()
	h.GetUserRoles(w3, getURReq)
	if w3.Code != http.StatusOK {
		t.Fatalf("get user roles: %d %s", w3.Code, w3.Body.String())
	}

	resp3 := decodeResponse(t, w3.Body.Bytes())
	var userRoles []struct {
		Name string `json:"name"`
	}
	json.Unmarshal(resp3.Data, &userRoles)
	var found bool
	for _, ur := range userRoles {
		if ur.Name == roleName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("role %q not found in user roles", roleName)
	}

	removeReq := httptest.NewRequest(http.MethodDelete, "/auth/sessions/users/"+userID.String()+"/roles/"+role.ID, nil)
	removeReq = withChiParams(removeReq, map[string]string{"userId": userID.String(), "roleId": role.ID})
	w4 := httptest.NewRecorder()
	h.RemoveRole(w4, removeReq)
	if w4.Code != http.StatusOK {
		t.Fatalf("remove role: %d %s", w4.Code, w4.Body.String())
	}

	getURReq2 := httptest.NewRequest(http.MethodGet, "/auth/sessions/users/"+userID.String()+"/roles", nil)
	getURReq2 = withChiParams(getURReq2, map[string]string{"userId": userID.String()})
	w5 := httptest.NewRecorder()
	h.GetUserRoles(w5, getURReq2)
	resp5 := decodeResponse(t, w5.Body.Bytes())
	var userRolesAfter []struct {
		Name string `json:"name"`
	}
	json.Unmarshal(resp5.Data, &userRolesAfter)
	for _, ur := range userRolesAfter {
		if ur.Name == roleName {
			t.Error("role still assigned after remove")
			break
		}
	}
}

func TestHandler_ValidationErrorOnCreateRole(t *testing.T) {
	h := setupHandler()

	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/roles", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateRole(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w.Body.Bytes())
	if resp.Error == nil {
		t.Fatal("expected error body")
	}
	if resp.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %q", resp.Error.Code)
	}
}

func TestHandler_BadJSONOnCreateRole(t *testing.T) {
	h := setupHandler()

	req := httptest.NewRequest(http.MethodPost, "/auth/sessions/roles", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateRole(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w.Body.Bytes())
	if resp.Error == nil {
		t.Fatal("expected error body")
	}
	if resp.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %q", resp.Error.Code)
	}
	if resp.Error.Message != "invalid request body" {
		t.Errorf("expected 'invalid request body', got %q", resp.Error.Message)
	}
}
