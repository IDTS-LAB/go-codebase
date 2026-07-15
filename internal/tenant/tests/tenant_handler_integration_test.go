package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/query"
	tenantRepo "github.com/IDTS-LAB/go-codebase/internal/tenant/infrastructure/persistence"
	tenantHttp "github.com/IDTS-LAB/go-codebase/internal/tenant/interfaces/http"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Meta    json.RawMessage `json:"meta,omitempty"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func setupHandler() *tenantHttp.Handler {
	repo := tenantRepo.NewTenantRepository(db)
	commandBus := cqrs.NewInMemoryCommandBus()
	queryBus := cqrs.NewInMemoryQueryBus()
	v := validator.New()

	commandBus.Register(command.CreateTenantCommand{}, command.NewCreateTenantHandler(repo))
	commandBus.Register(command.UpdateTenantCommand{}, command.NewUpdateTenantHandler(repo))
	commandBus.Register(command.DeleteTenantCommand{}, command.NewDeleteTenantHandler(repo))
	queryBus.Register(query.GetTenantQuery{}, query.NewGetTenantHandler(repo))
	queryBus.Register(query.ListTenantsQuery{}, query.NewListTenantsHandler(repo))

	return tenantHttp.NewHandler(commandBus, queryBus, v)
}

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func createTenant(t *testing.T, handler *tenantHttp.Handler, name, slug string) *httptest.ResponseRecorder {
	t.Helper()
	body := dto.CreateTenantRequest{Name: name, Slug: slug}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)
	return w
}

func parseResponse(t *testing.T, body []byte) apiResponse {
	t.Helper()
	var resp apiResponse
	err := json.Unmarshal(body, &resp)
	assert.NoError(t, err)
	return resp
}

func TestTenantHandler_CreateAndGetByID(t *testing.T) {
	handler := setupHandler()

	slug := "test-corp-" + uuid.New().String()[:8]
	w := createTenant(t, handler, "Test Corp", slug)
	assert.Equal(t, http.StatusCreated, w.Code)

	createResp := parseResponse(t, w.Body.Bytes())
	assert.True(t, createResp.Success)

	var tenant dto.TenantResponse
	err := json.Unmarshal(createResp.Data, &tenant)
	assert.NoError(t, err)
	assert.Equal(t, "Test Corp", tenant.Name)
	assert.Equal(t, slug, tenant.Slug)

	req := httptest.NewRequest(http.MethodGet, "/tenants/"+tenant.ID, nil)
	req = withChiParams(req, map[string]string{"id": tenant.ID})
	w2 := httptest.NewRecorder()
	handler.GetByID(w2, req)

	assert.Equal(t, http.StatusOK, w2.Code)

	getResp := parseResponse(t, w2.Body.Bytes())
	assert.True(t, getResp.Success)

	var got dto.TenantResponse
	err = json.Unmarshal(getResp.Data, &got)
	assert.NoError(t, err)
	assert.Equal(t, tenant.Name, got.Name)
	assert.Equal(t, tenant.Slug, got.Slug)
}

func TestTenantHandler_CreateAndList(t *testing.T) {
	handler := setupHandler()

	w := createTenant(t, handler, "List Corp", "list-corp-"+uuid.New().String()[:8])
	assert.Equal(t, http.StatusCreated, w.Code)

	createResp := parseResponse(t, w.Body.Bytes())
	var tenant dto.TenantResponse
	json.Unmarshal(createResp.Data, &tenant)

	req := httptest.NewRequest(http.MethodGet, "/tenants?limit=100", nil)
	w2 := httptest.NewRecorder()
	handler.List(w2, req)

	assert.Equal(t, http.StatusOK, w2.Code)

	listResp := parseResponse(t, w2.Body.Bytes())
	assert.True(t, listResp.Success)

	var tenants []dto.TenantResponse
	err := json.Unmarshal(listResp.Data, &tenants)
	assert.NoError(t, err)

	found := false
	for _, tnt := range tenants {
		if tnt.ID == tenant.ID {
			found = true
			assert.Equal(t, "List Corp", tnt.Name)
			break
		}
	}
	assert.True(t, found, "created tenant should appear in list")
}

func TestTenantHandler_CreateAndUpdate(t *testing.T) {
	handler := setupHandler()

	w := createTenant(t, handler, "Before Corp", "before-corp-"+uuid.New().String()[:8])
	assert.Equal(t, http.StatusCreated, w.Code)

	createResp := parseResponse(t, w.Body.Bytes())
	var tenant dto.TenantResponse
	json.Unmarshal(createResp.Data, &tenant)

	newName := "After Corp"
	updateReq := dto.UpdateTenantRequest{Name: &newName}
	b, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/tenants/"+tenant.ID, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": tenant.ID})
	w2 := httptest.NewRecorder()
	handler.Update(w2, req)

	assert.Equal(t, http.StatusOK, w2.Code)

	updateResp := parseResponse(t, w2.Body.Bytes())
	assert.True(t, updateResp.Success)

	var updated dto.TenantResponse
	json.Unmarshal(updateResp.Data, &updated)
	assert.Equal(t, newName, updated.Name)

	req3 := httptest.NewRequest(http.MethodGet, "/tenants/"+tenant.ID, nil)
	req3 = withChiParams(req3, map[string]string{"id": tenant.ID})
	w3 := httptest.NewRecorder()
	handler.GetByID(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code)

	getResp := parseResponse(t, w3.Body.Bytes())
	var got dto.TenantResponse
	json.Unmarshal(getResp.Data, &got)
	assert.Equal(t, newName, got.Name)
}

func TestTenantHandler_CreateAndDelete(t *testing.T) {
	handler := setupHandler()

	w := createTenant(t, handler, "Delete Corp", "delete-corp-"+uuid.New().String()[:8])
	assert.Equal(t, http.StatusCreated, w.Code)

	createResp := parseResponse(t, w.Body.Bytes())
	var tenant dto.TenantResponse
	json.Unmarshal(createResp.Data, &tenant)

	req := httptest.NewRequest(http.MethodDelete, "/tenants/"+tenant.ID, nil)
	req = withChiParams(req, map[string]string{"id": tenant.ID})
	w2 := httptest.NewRecorder()
	handler.Delete(w2, req)

	assert.Equal(t, http.StatusOK, w2.Code)

	req3 := httptest.NewRequest(http.MethodGet, "/tenants/"+tenant.ID, nil)
	req3 = withChiParams(req3, map[string]string{"id": tenant.ID})
	w3 := httptest.NewRecorder()
	handler.GetByID(w3, req3)

	assert.Equal(t, http.StatusNotFound, w3.Code)
}

func TestTenantHandler_CreateDuplicateSlug(t *testing.T) {
	handler := setupHandler()

	slug := "dup-slug-" + uuid.New().String()[:8]
	w := createTenant(t, handler, "First", slug)
	assert.Equal(t, http.StatusCreated, w.Code)

	w2 := createTenant(t, handler, "Second", slug)
	assert.Equal(t, http.StatusConflict, w2.Code)

	resp := parseResponse(t, w2.Body.Bytes())
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "CONFLICT", resp.Error.Code)
}

func TestTenantHandler_GetByID_NotFound(t *testing.T) {
	handler := setupHandler()

	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/tenants/"+id, nil)
	req = withChiParams(req, map[string]string{"id": id})
	w := httptest.NewRecorder()
	handler.GetByID(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResponse(t, w.Body.Bytes())
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
}

func TestTenantHandler_Update_NotFound(t *testing.T) {
	handler := setupHandler()

	id := uuid.New().String()
	newName := "Nope"
	body := dto.UpdateTenantRequest{Name: &newName}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/tenants/"+id, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": id})
	w := httptest.NewRecorder()
	handler.Update(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResponse(t, w.Body.Bytes())
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
}

func TestTenantHandler_Delete_NotFound(t *testing.T) {
	handler := setupHandler()

	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/tenants/"+id, nil)
	req = withChiParams(req, map[string]string{"id": id})
	w := httptest.NewRecorder()
	handler.Delete(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResponse(t, w.Body.Bytes())
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
}

func TestTenantHandler_Create_ValidationError(t *testing.T) {
	handler := setupHandler()

	body := dto.CreateTenantRequest{}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse(t, w.Body.Bytes())
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestTenantHandler_Create_BadJSON(t *testing.T) {
	handler := setupHandler()

	req := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse(t, w.Body.Bytes())
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}
