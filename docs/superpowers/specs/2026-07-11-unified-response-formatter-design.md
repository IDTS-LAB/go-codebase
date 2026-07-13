# Unified Response Formatter Design

**Date:** 2026-07-11  
**Topic:** Response standardization across HTTP handlers  
**Status:** Approved

## Goal

Provide three complementary mechanisms so that HTTP handlers produce the unified `APIResponse` envelope with minimal boilerplate, while remaining testable and backward-compatible with the existing codebase.

The unified envelope is defined as:

```json
{
  "success": true,
  "data":    { ... },
  "meta":    { "page": 1, "per_page": 20, "total": 100, "total_pages": 5 },
  "error":   { "code": "NOT_FOUND", "message": "..." }
}
```

## Current State

- `internal/shared/utils/utils.go` defines `APIResponse`, `PaginationMeta`, `ErrorBody`, and low-level `Respond*` functions.
- `internal/shared/utils/handler.go` defines `Handle`, `HandleCreated`, `HandleNoContent`, and `HandlePaginated` helpers.
- Handlers in `todo`, `authentication`, `authorization`, and `user` domains write responses manually using a mix of `RespondSuccess`, `RespondCreated`, `RespondPaginated`, `RespondError`, and `MapError`.
- There is no middleware safety net: a handler that writes raw JSON accidentally bypasses the envelope.

## Design

### 1. Generic Helpers

Location: `internal/shared/utils/handler.go`

These are one-line functions handlers call at the end of a request handler. They centralize error mapping and envelope creation.

```go
func Handle(w http.ResponseWriter, data interface{}, err error)
func HandleCreated(w http.ResponseWriter, data interface{}, err error)
func HandleNoContent(w http.ResponseWriter, err error)
func HandlePaginated(w http.ResponseWriter, data interface{}, page, perPage, total int, err error)
```

Responsibilities:
- Map `err` to the correct HTTP status and error code via `MapError`.
- Write the success envelope with the appropriate HTTP status and optional pagination meta.

Use when:
- Refactoring existing handlers quickly.
- The handler is a method on an existing struct and you do not want to change its signature.

### 2. Automatic Middleware Formatter

Location: `internal/shared/middleware/formatter.go`

A middleware that buffers the handler’s response body, then rewrites it if the handler did not emit a full `APIResponse` envelope.

Behavior:
1. Wrap `http.ResponseWriter` in a `formattingWriter` that captures `WriteHeader` calls and body bytes without sending them downstream.
2. Run the inner handler.
3. If the captured body is empty, write the buffered status and headers as-is.
4. If the captured body parses as JSON and contains a top-level `success` field, assume it is already an envelope and pass it through.
5. If the captured body is raw JSON and status < 400, wrap it as:
   ```json
   { "success": true, "data": <raw-body>, "meta": null }
   ```
6. If status >= 400 and the body is plain text, wrap it as:
   ```json
   { "success": false, "data": null, "error": { "code": "<status-text>", "message": "<body>" } }
   ```
7. If status >= 400 and the body is raw JSON, treat it as `{ "code": "...", "message": "..." }` and wrap it in the error envelope. If those keys are missing, use the status text and raw JSON string as the message.
8. For pagination, recognize `utils.PaginatedPayload[T]`:
   ```go
   type PaginatedPayload[T any] struct {
       Data       []T          `json:"data"`
       Pagination PaginationMeta `json:"pagination"`
   }
   ```
   When detected, unwrap it into `data` and `meta` in the final envelope.

Use when:
- As a global safety net so no handler can accidentally bypass the envelope.
- When you want handlers to write raw DTOs directly and let the middleware handle formatting.

Ordering:
- Place the formatter middleware **after** panic recovery, request ID, logging, and tracing, but **before** authentication/authorization error responses are written. This ensures auth errors are also normalized if they bypass helpers.

### 3. Handler Adapter / Controller Pattern

Location: `internal/shared/httpadapter/adapter.go`

A small adapter layer that lets handlers be written as pure functions returning `(T, error)` instead of interacting with `http.ResponseWriter`.

```go
func Adapt[T any](fn func(ctx context.Context, r *http.Request) (T, error)) http.HandlerFunc
func AdaptCreated[T any](fn func(ctx context.Context, r *http.Request) (T, error)) http.HandlerFunc
func AdaptNoContent(fn func(ctx context.Context, r *http.Request) error) http.HandlerFunc
func AdaptPaginated[T any](fn func(ctx context.Context, r *http.Request) (PaginatedResult[T], error)) http.HandlerFunc
```

Where:

```go
type PaginatedResult[T any] struct {
    Data       []T
    Page       int
    PerPage    int
    Total      int
}
```

Responsibilities:
- Decode JSON body if needed (optional helper; handlers may still decode themselves).
- Call the provided function.
- Forward the result and error to `utils.Handle*`, producing the correct envelope.

Use when:
- Writing new handlers from scratch.
- The handler is a thin translation layer between HTTP and application services.
- You want to test business logic without an `http.ResponseWriter`.

## Integration

```
HTTP Request
    │
    ▼
[Recovery / RequestID / Logger / Tracer]
    │
    ▼
[Authentication / Authorization]
    │
    ▼
[Response Formatter Middleware]  ◄── safety net
    │
    ▼
[Handler]
    │
    ├── uses utils.Handle* directly
    ├── writes raw JSON (formatter wraps it)
    └── is a pure function wired via httpadapter.Adapt*
```

## Migration Strategy

1. Implement and add the formatter middleware globally. No handler changes are required at this step.
2. Refactor existing handlers incrementally:
   - Replace manual `RespondSuccess/RespondCreated/RespondPaginated/MapError` blocks with `utils.Handle*`.
   - For handlers that are already thin, consider converting to `httpadapter.Adapt*`.
3. Update Swagger annotations to reference `utils.APIResponse` (already largely done).
4. Add unit tests for middleware and adapter before relying on them.

## Error Mapping

`utils.MapError` remains the single source of truth for mapping domain errors to HTTP statuses and codes. The middleware does not duplicate this logic; it only normalizes responses that did not use `MapError`.

## Testing

### Middleware tests
- Raw success JSON is wrapped in the success envelope.
- Raw error text with 4xx/5xx status is wrapped in the error envelope.
- Existing full envelopes are passed through unchanged.
- Empty body is passed through unchanged.
- `PaginatedPayload` is unwrapped into `data` + `meta`.
- `Content-Type: application/json` is set on wrapped responses.

### Adapter tests
- Successful function result produces a 200 success envelope.
- Error result produces the mapped error envelope.
- Created adapter produces 201.
- No-content adapter produces 200 with `data: null`.
- Paginated adapter computes `total_pages` correctly.

## Files Added / Modified

- `internal/shared/utils/handler.go` — refine helpers.
- `internal/shared/utils/utils.go` — add `PaginatedPayload` and `PaginatedResult` types.
- `internal/shared/middleware/formatter.go` — new formatter middleware.
- `internal/shared/httpadapter/adapter.go` — new adapter package.
- Router setup — register formatter middleware.
- Existing handler files — opportunistically migrate to helpers/adapter.
- Tests — add middleware and adapter unit tests.
