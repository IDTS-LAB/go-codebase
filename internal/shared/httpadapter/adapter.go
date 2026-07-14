package httpadapter

import (
	"context"
	"net/http"

	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
)

func Adapt[T any](fn func(ctx context.Context, r *http.Request) (T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fn(r.Context(), r)
		utils.Handle(w, r, data, err)
	}
}

func AdaptCreated[T any](fn func(ctx context.Context, r *http.Request) (T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fn(r.Context(), r)
		utils.HandleCreated(w, r, data, err)
	}
}

func AdaptNoContent(fn func(ctx context.Context, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(r.Context(), r)
		utils.HandleNoContent(w, r, err)
	}
}

func AdaptPaginated[T any](fn func(ctx context.Context, r *http.Request) (utils.PaginatedResult[T], error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := fn(r.Context(), r)
		utils.HandlePaginated(w, r, result.Data, result.Page, result.PerPage, result.Total, err)
	}
}
