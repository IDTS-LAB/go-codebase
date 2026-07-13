# Unified Response Formatter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build three complementary response-formatting mechanisms (helpers, middleware, adapter) so handlers always produce the unified `APIResponse` envelope with minimal boilerplate.

**Architecture:** A formatter middleware buffers handler output and normalizes unwrapped responses; `utils.Handle*` helpers provide one-line manual formatting; an `httpadapter` package lets handlers be written as pure `(T, error)` functions.

**Tech Stack:** Go 1.24, Chi router, standard `encoding/json`, `testify/assert`.

## Global Constraints

- All HTTP responses must use the unified `APIResponse` envelope.
- Error mapping stays centralized in `utils.MapError`.
- Middleware must be backward-compatible with existing `Respond*` helpers to avoid double-wrapping.
- New code must include unit tests with table-driven cases.
- Run `go test ./...`, `go vet ./...`, and `go build ./...` before each commit.

---

## File Structure

- `internal/shared/utils/utils.go` — existing envelope types; add `PaginatedPayload[T]` and `PaginatedResult[T]`.
- `internal/shared/utils/handler.go` — existing `Handle*` helpers; keep and ensure they call `MapError` correctly.
- `internal/shared/middleware/formatter.go` — new formatter middleware with buffered response writer.
- `internal/shared/middleware/formatter_test.go` — new middleware unit tests.
- `internal/shared/httpadapter/adapter.go` — new adapter functions for pure handlers.
- `internal/shared/httpadapter/adapter_test.go` — new adapter unit tests.
- `internal/shared/router/router.go` or equivalent — register formatter middleware.
- Handler files — opportunistically migrate to `utils.Handle*` or `httpadapter.Adapt*`.

---

### Task 1: Add Pagination Types

**Files:**
- Modify: `internal/shared/utils/utils.go`
- Test: `internal/shared/utils/utils_test.go` (create if missing)

**Interfaces:**
- Consumes: nothing new.
- Produces:
  ```go
  type PaginatedPayload[T any] struct {
      Data       []T            `json:"data"`
      Pagination PaginationMeta `json:"pagination"`
  }
  type PaginatedResult[T any] struct {
      Data    []T
      Page    int
      PerPage int
      Total   int
  }
  ```

- [ ] **Step 1: Add types to `utils.go`**

  Add the following structs after `ErrorBody`:

  ```go
  type PaginatedPayload[T any] struct {
      Data       []T            `json:"data"`
      Pagination PaginationMeta `json:"pagination"`
  }

  type PaginatedResult[T any] struct {
      Data    []T
      Page    int
      PerPage int
      Total   int
  }
  ```

- [ ] **Step 2: Verify build**

  Run: `go build ./internal/shared/utils/...`
  Expected: success

- [ ] **Step 3: Commit**

  ```bash
  git add internal/shared/utils/utils.go
  git commit -m "chore(utils): add PaginatedPayload and PaginatedResult types"
  ```

---

### Task 2: Implement Response Formatter Middleware

**Files:**
- Create: `internal/shared/middleware/formatter.go`
- Create: `internal/shared/middleware/formatter_test.go`

**Interfaces:**
- Consumes: `utils.APIResponse`, `utils.PaginationMeta`, `utils.PaginatedPayload[T]`.
- Produces:
  ```go
  func ResponseFormatter() func(http.Handler) http.Handler
  ```

- [ ] **Step 1: Write failing middleware test**

  Create `internal/shared/middleware/formatter_test.go`:

  ```go
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

      var resp utils.APIResponse
      err := json.Unmarshal(rec.Body.Bytes(), &resp)
      assert.NoError(t, err)
      assert.True(t, resp.Success)
      assert.NotNil(t, resp.Meta)
      assert.Equal(t, 1, resp.Meta.Page)
  }
  ```

- [ ] **Step 2: Run tests to verify they fail**

  Run: `go test ./internal/shared/middleware/... -run TestResponseFormatter -v`
  Expected: failures because `ResponseFormatter` does not exist.

- [ ] **Step 3: Implement formatter middleware**

  Create `internal/shared/middleware/formatter.go`:

  ```go
  package middleware

  import (
      "bytes"
      "encoding/json"
      "net/http"

      "github.com/IDTS-LAB/go-codebase/internal/shared/utils"
  )

  func ResponseFormatter() func(http.Handler) http.Handler {
      return func(next http.Handler) http.Handler {
          return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              fw := &formattingWriter{ResponseWriter: w, statusCode: http.StatusOK}
              next.ServeHTTP(fw, r)

              if len(fw.body) == 0 {
                  w.WriteHeader(fw.statusCode)
                  return
              }

              if isEnvelope(fw.body) {
                  w.Header().Set("Content-Type", "application/json")
                  w.WriteHeader(fw.statusCode)
                  w.Write(fw.body)
                  return
              }

              w.Header().Set("Content-Type", "application/json")
              w.WriteHeader(fw.statusCode)

              if fw.statusCode >= 400 {
                  var errBody struct {
                      Code    string `json:"code"`
                      Message string `json:"message"`
                  }
                  if json.Unmarshal(fw.body, &errBody) == nil && errBody.Message != "" {
                      json.NewEncoder(w).Encode(utils.APIResponse{
                          Success: false,
                          Error:   &utils.ErrorBody{Code: errBody.Code, Message: errBody.Message},
                      })
                      return
                  }
                  json.NewEncoder(w).Encode(utils.APIResponse{
                      Success: false,
                      Error:   &utils.ErrorBody{Code: http.StatusText(fw.statusCode), Message: string(bytes.TrimSpace(fw.body))},
                  })
                  return
              }

              var paginated struct {
                  Data       interface{} `json:"data"`
                  Pagination interface{} `json:"pagination"`
              }
              if json.Unmarshal(fw.body, &paginated) == nil && paginated.Data != nil && paginated.Pagination != nil {
                  var meta utils.PaginationMeta
                  metaBytes, _ := json.Marshal(paginated.Pagination)
                  json.Unmarshal(metaBytes, &meta)
                  json.NewEncoder(w).Encode(utils.APIResponse{
                      Success: true,
                      Data:    paginated.Data,
                      Meta:    &meta,
                  })
                  return
              }

              var raw interface{}
              json.Unmarshal(fw.body, &raw)
              json.NewEncoder(w).Encode(utils.APIResponse{
                  Success: true,
                  Data:    raw,
                  Meta:    nil,
              })
          })
      }
  }

  type formattingWriter struct {
      http.ResponseWriter
      statusCode int
      body       []byte
      wroteHeader bool
  }

  func (w *formattingWriter) WriteHeader(code int) {
      if w.wroteHeader {
          return
      }
      w.statusCode = code
      w.wroteHeader = true
  }

  func (w *formattingWriter) Write(b []byte) (int, error) {
      w.body = append(w.body, b...)
      return len(b), nil
  }

  func (w *formattingWriter) Header() http.Header {
      return w.ResponseWriter.Header()
  }

  func isEnvelope(body []byte) bool {
      var envelope struct {
          Success *bool `json:"success"`
      }
      if err := json.Unmarshal(body, &envelope); err != nil {
          return false
      }
      return envelope.Success != nil
  }
  ```

- [ ] **Step 4: Run tests to verify they pass**

  Run: `go test ./internal/shared/middleware/... -run TestResponseFormatter -v`
  Expected: all tests pass.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/shared/middleware/formatter.go internal/shared/middleware/formatter_test.go
  git commit -m "feat(middleware): add response formatter middleware"
  ```

---

### Task 3: Wire Formatter Middleware into Router

**Files:**
- Modify: router setup file (find where Chi middleware is registered, e.g., `internal/shared/router/router.go` or `cmd/api/main.go`)

**Interfaces:**
- Consumes: `middleware.ResponseFormatter()`.
- Produces: formatter middleware active on all routes.

- [ ] **Step 1: Find router setup**

  Run: `grep -R "Use(" --include="*.go" . | grep -i middleware` or search for `chi.NewRouter()`.
  Identify the file where middleware is registered.

- [ ] **Step 2: Register formatter middleware**

  Add `middleware.ResponseFormatter()` near the end of the middleware chain, after auth but before route handlers. Example:

  ```go
  r.Use(middleware.RequestID)
  r.Use(middleware.Logger(log))
  r.Use(middleware.ErrorHandler(log, errorRepo))
  r.Use(middleware.ErrorRecorder(log, errorRepo))
  r.Use(middleware.Tracer())
  r.Use(middleware.ResponseFormatter()) // add this line
  r.Use(middleware.Authentication(tokenSvc))
  ```

  Exact placement depends on the router file found in Step 1. The formatter should run after recovery/logging/tracing and before auth so auth error responses are also normalized if they bypass helpers.

- [ ] **Step 3: Verify build**

  Run: `go build ./...`
  Expected: success.

- [ ] **Step 4: Commit**

  ```bash
  git add <router-file>
  git commit -m "feat(router): wire response formatter middleware"
  ```

---

### Task 4: Implement Handler Adapter Package

**Files:**
- Create: `internal/shared/httpadapter/adapter.go`
- Create: `internal/shared/httpadapter/adapter_test.go`

**Interfaces:**
- Consumes: `utils.Handle`, `utils.HandleCreated`, `utils.HandleNoContent`, `utils.HandlePaginated`, `utils.PaginatedResult[T]`.
- Produces:
  ```go
  func Adapt[T any](fn func(ctx context.Context, r *http.Request) (T, error)) http.HandlerFunc
  func AdaptCreated[T any](fn func(ctx context.Context, r *http.Request) (T, error)) http.HandlerFunc
  func AdaptNoContent(fn func(ctx context.Context, r *http.Request) error) http.HandlerFunc
  func AdaptPaginated[T any](fn func(ctx context.Context, r *http.Request) (PaginatedResult[T], error)) http.HandlerFunc
  ```

- [ ] **Step 1: Write failing adapter tests**

  Create `internal/shared/httpadapter/adapter_test.go`:

  ```go
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
      assert.Equal(t, 1, resp.Meta.TotalPages)
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
  ```

- [ ] **Step 2: Run tests to verify they fail**

  Run: `go test ./internal/shared/httpadapter/... -v`
  Expected: failures because package does not exist.

- [ ] **Step 3: Implement adapter package**

  Create `internal/shared/httpadapter/adapter.go`:

  ```go
  package httpadapter

  import (
      "context"
      "net/http"

      "github.com/IDTS-LAB/go-codebase/internal/shared/utils"
  )

  func Adapt[T any](fn func(ctx context.Context, r *http.Request) (T, error)) http.HandlerFunc {
      return func(w http.ResponseWriter, r *http.Request) {
          data, err := fn(r.Context(), r)
          utils.Handle(w, data, err)
      }
  }

  func AdaptCreated[T any](fn func(ctx context.Context, r *http.Request) (T, error)) http.HandlerFunc {
      return func(w http.ResponseWriter, r *http.Request) {
          data, err := fn(r.Context(), r)
          utils.HandleCreated(w, data, err)
      }
  }

  func AdaptNoContent(fn func(ctx context.Context, r *http.Request) error) http.HandlerFunc {
      return func(w http.ResponseWriter, r *http.Request) {
          err := fn(r.Context(), r)
          utils.HandleNoContent(w, err)
      }
  }

  func AdaptPaginated[T any](fn func(ctx context.Context, r *http.Request) (utils.PaginatedResult[T], error)) http.HandlerFunc {
      return func(w http.ResponseWriter, r *http.Request) {
          result, err := fn(r.Context(), r)
          utils.HandlePaginated(w, result.Data, result.Page, result.PerPage, result.Total, err)
      }
  }
  ```

- [ ] **Step 4: Run tests to verify they pass**

  Run: `go test ./internal/shared/httpadapter/... -v`
  Expected: all tests pass.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/shared/httpadapter/
  git commit -m "feat(httpadapter): add pure-function handler adapters"
  ```

---

### Task 5: Refactor Existing Handlers to Use Helpers or Adapter

**Files:**
- Modify: `internal/authentication/interfaces/http/handlers.go`
- Modify: `internal/todo/interfaces/http/handlers.go`
- Modify: `internal/authorization/interfaces/http/handlers.go`
- Modify: `internal/user/interfaces/http/handler.go`

**Interfaces:**
- Consumes: `utils.Handle`, `utils.HandleCreated`, `utils.HandleNoContent`, `utils.HandlePaginated`.
- Produces: shorter, consistent handlers that produce the same envelope.

- [ ] **Step 1: Refactor authentication handlers**

  Replace patterns like:
  ```go
  user, err := h.svc.Register(...)
  if err != nil {
      utils.MapError(w, err)
      return
  }
  utils.RespondCreated(w, user)
  ```
  with:
  ```go
  user, err := h.svc.Register(...)
  utils.HandleCreated(w, user, err)
  ```

  Do this for all endpoints in `internal/authentication/interfaces/http/handlers.go` where it does not change behavior. Keep custom branches (e.g., `ErrInvalidVerifyToken` → `RespondBadRequest`) if they provide more specific messages than `MapError`.

- [ ] **Step 2: Refactor todo handlers**

  Apply the same replacement pattern in `internal/todo/interfaces/http/handlers.go`.

- [ ] **Step 3: Refactor authorization handlers**

  Apply the same replacement pattern in `internal/authorization/interfaces/http/handlers.go`.

- [ ] **Step 4: Refactor user handlers**

  Apply the same replacement pattern in `internal/user/interfaces/http/handler.go`.

- [ ] **Step 5: Run tests and fix failures**

  Run: `go test ./...`
  Expected: all tests pass. Fix any test that asserts on exact response structure if the refactor changes it.

- [ ] **Step 6: Commit**

  ```bash
  git add internal/authentication/interfaces/http/handlers.go \
             internal/todo/interfaces/http/handlers.go \
             internal/authorization/interfaces/http/handlers.go \
             internal/user/interfaces/http/handler.go
  git commit -m "refactor(handlers): use utils.Handle* helpers"
  ```

---

### Task 6: Regenerate Swagger and Run Final Verification

**Files:**
- Modify: `docs/swagger.json`, `docs/swagger.yaml` (if generated)
- Test: entire suite

- [ ] **Step 1: Regenerate Swagger docs**

  Run: `make swagger` (or `swag init -g cmd/api/main.go`, whichever is configured).

- [ ] **Step 2: Run full verification**

  Run:
  ```bash
  go build ./...
  go vet ./...
  go test ./...
  ```
  Expected: all pass.

- [ ] **Step 3: Commit**

  ```bash
  git add docs/
  git commit -m "docs(swagger): regenerate after response formatter refactor"
  ```

---

## Self-Review Checklist

- [ ] Spec coverage: helpers, middleware, adapter, router wiring, handler refactor, tests, swagger regeneration are all represented.
- [ ] No placeholders: every step includes exact code or exact commands.
- [ ] Type consistency: `PaginatedPayload`, `PaginatedResult`, `APIResponse`, `ErrorBody`, `PaginationMeta` match across tasks.
- [ ] Middleware ordering: formatter placed after recovery/logging/tracing and before auth so error responses are normalized.
- [ ] Backward compatibility: formatter detects existing envelopes and passes them through without double-wrapping.
