package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/stretchr/testify/assert"
)

func TestResponseFormatter_WrapsRawSuccessJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "1", "name": "todo"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ResponseFormatter()(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp utils.APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	assert.Nil(t, resp.Error)
}

func TestResponseFormatter_PassesThroughExistingEnvelope(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		utils.RespondSuccess(w, map[string]string{"id": "1"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ResponseFormatter()(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp utils.APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
}

func TestResponseFormatter_WrapsRawErrorText(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ResponseFormatter()(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp utils.APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "Not Found", resp.Error.Code)
	assert.Equal(t, "not found", resp.Error.Message)
}

func TestResponseFormatter_UnwrapsPaginatedPayload(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := utils.PaginatedPayload[map[string]string]{
			Data: []map[string]string{{"id": "1"}},
			Pagination: utils.PaginationMeta{
				Page:       1,
				PerPage:    20,
				Total:      1,
				TotalPages: 1,
			},
		}
		json.NewEncoder(w).Encode(payload)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ResponseFormatter()(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp utils.APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Meta)
	assert.Equal(t, 1, resp.Meta.Page)
}
