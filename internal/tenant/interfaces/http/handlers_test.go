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
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/query"
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

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestCreate_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.CreateTenantCommand{}, &mockHandler{
		result: dto.TenantResponse{ID: uuid.New().String(), Name: "Acme Corp", Slug: "acme-corp"},
	})

	body, _ := json.Marshal(dto.CreateTenantRequest{Name: "Acme Corp", Slug: "acme-corp"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewReader(body))
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
	h := NewHandler(cmdBus, qBus, validator.New())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewReader([]byte("{invalid")))
	r.Header.Set("Content-Type", "application/json")

	h.Create(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_ValidationError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	body, _ := json.Marshal(dto.CreateTenantRequest{Name: "", Slug: ""})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	h.Create(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.CreateTenantCommand{}, &mockHandler{err: errors.New("db error")})

	body, _ := json.Marshal(dto.CreateTenantRequest{Name: "Acme", Slug: "acme"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	h.Create(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
}

func TestCreate_Conflict(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cmdBus.Register(command.CreateTenantCommand{}, &mockHandler{
		err: domain.NewDomainError(domain.ErrAlreadyExists, "TENANT_EXISTS", "tenant with this slug already exists"),
	})

	body, _ := json.Marshal(dto.CreateTenantRequest{Name: "Acme", Slug: "acme"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	h.Create(w, r)

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
	assert.Equal(t, "CONFLICT", resp["error"].(map[string]interface{})["code"])
}

func TestList_Success_DefaultLimit(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	qBus.Register(query.ListTenantsQuery{}, &mockHandler{
		result: dto.TenantListResponse{Tenants: []dto.TenantResponse{}, Limit: 20},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tenants", nil)

	h.List(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
	assert.NotNil(t, resp["meta"])
}

func TestList_Success_WithCursor(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	cursor := "next-cursor"
	qBus.Register(query.ListTenantsQuery{}, &mockHandler{
		result: dto.TenantListResponse{
			Tenants:    []dto.TenantResponse{},
			NextCursor: &cursor,
			HasNext:    true,
			Limit:      10,
		},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tenants?cursor=abc&limit=10", nil)

	h.List(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
}

func TestList_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	qBus.Register(query.ListTenantsQuery{}, &mockHandler{err: errors.New("db error")})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tenants", nil)

	h.List(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
}

func TestGetByID_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	id := uuid.New()
	qBus.Register(query.GetTenantQuery{}, &mockHandler{
		result: dto.TenantResponse{ID: id.String(), Name: "Acme", Slug: "acme"},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tenants/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.GetByID(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.True(t, resp["success"].(bool))
}

func TestGetByID_InvalidUUID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tenants/invalid", nil)
	r = withChiParams(r, map[string]string{"id": "invalid"})

	h.GetByID(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetByID_NotFound(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	id := uuid.New()
	qBus.Register(query.GetTenantQuery{}, &mockHandler{
		err: domain.NewDomainError(domain.ErrNotFound, "TENANT_NOT_FOUND", "tenant not found"),
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tenants/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.GetByID(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetByID_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	id := uuid.New()
	qBus.Register(query.GetTenantQuery{}, &mockHandler{err: errors.New("db error")})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tenants/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.GetByID(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
}

func TestUpdate_Success(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	id := uuid.New()
	cmdBus.Register(command.UpdateTenantCommand{}, &mockHandler{
		result: dto.TenantResponse{ID: id.String(), Name: "Updated", Slug: "acme"},
	})

	name := "Updated"
	body, _ := json.Marshal(dto.UpdateTenantRequest{Name: &name})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/tenants/"+id.String(), bytes.NewReader(body))
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
	h := NewHandler(cmdBus, qBus, validator.New())

	name := "Updated"
	body, _ := json.Marshal(dto.UpdateTenantRequest{Name: &name})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/tenants/invalid", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withChiParams(r, map[string]string{"id": "invalid"})

	h.Update(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_BadJSON(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	id := uuid.New()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/tenants/"+id.String(), bytes.NewReader([]byte("{invalid")))
	r.Header.Set("Content-Type", "application/json")
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Update(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_NotFound(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	id := uuid.New()
	cmdBus.Register(command.UpdateTenantCommand{}, &mockHandler{
		err: domain.NewDomainError(domain.ErrNotFound, "TENANT_NOT_FOUND", "tenant not found"),
	})

	name := "Updated"
	body, _ := json.Marshal(dto.UpdateTenantRequest{Name: &name})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/tenants/"+id.String(), bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Update(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdate_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	id := uuid.New()
	cmdBus.Register(command.UpdateTenantCommand{}, &mockHandler{err: errors.New("db error")})

	name := "Updated"
	body, _ := json.Marshal(dto.UpdateTenantRequest{Name: &name})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/tenants/"+id.String(), bytes.NewReader(body))
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
	h := NewHandler(cmdBus, qBus, validator.New())

	id := uuid.New()
	cmdBus.Register(command.DeleteTenantCommand{}, &mockHandler{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/tenants/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Delete(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDelete_InvalidUUID(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/tenants/invalid", nil)
	r = withChiParams(r, map[string]string{"id": "invalid"})

	h.Delete(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDelete_NotFound(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	id := uuid.New()
	cmdBus.Register(command.DeleteTenantCommand{}, &mockHandler{
		err: domain.NewDomainError(domain.ErrNotFound, "TENANT_NOT_FOUND", "tenant not found"),
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/tenants/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Delete(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDelete_BusError(t *testing.T) {
	cmdBus := cqrs.NewInMemoryCommandBus()
	qBus := cqrs.NewInMemoryQueryBus()
	h := NewHandler(cmdBus, qBus, validator.New())

	id := uuid.New()
	cmdBus.Register(command.DeleteTenantCommand{}, &mockHandler{err: errors.New("db error")})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/tenants/"+id.String(), nil)
	r = withChiParams(r, map[string]string{"id": id.String()})

	h.Delete(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["success"].(bool))
}
