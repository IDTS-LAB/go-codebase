package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/stretchr/testify/assert"
)

func TestAdapt_ReturnsSuccessEnvelope(t *testing.T) {
	handler := Adapt(func(ctx context.Context, r *http.Request) (map[string]string, error) {
		return map[string]string{"id": "1"}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp utils.APIResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
}

func TestAdaptCreated_Returns201(t *testing.T) {
	handler := AdaptCreated(func(ctx context.Context, r *http.Request) (map[string]string, error) {
		return map[string]string{"id": "1"}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp utils.APIResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
}

func TestAdaptNoContent_ReturnsSuccessWithNilData(t *testing.T) {
	handler := AdaptNoContent(func(ctx context.Context, r *http.Request) error {
		return nil
	})

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp utils.APIResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Nil(t, resp.Data)
}

func TestAdaptPaginated_ReturnsPaginationMeta(t *testing.T) {
	handler := AdaptPaginated(func(ctx context.Context, r *http.Request) (utils.PaginatedResult[map[string]string], error) {
		return utils.PaginatedResult[map[string]string]{
			Data:    []map[string]string{{"id": "1"}},
			Page:    1,
			PerPage: 20,
			Total:   1,
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp utils.APIResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Meta)
	meta := resp.Meta.(map[string]interface{})
	assert.Equal(t, float64(1), meta["total_pages"])
}

func TestAdapt_MapsDomainError(t *testing.T) {
	handler := Adapt(func(ctx context.Context, r *http.Request) (map[string]string, error) {
		return nil, domain.ErrNotFound
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var resp utils.APIResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
}
