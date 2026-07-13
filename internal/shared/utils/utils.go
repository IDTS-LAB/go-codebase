package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"runtime/debug"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
)

var IsProduction bool

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
	Stack   string      `json:"stack,omitempty"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type CursorMeta struct {
	NextCursor *string `json:"next_cursor"`
	PrevCursor *string `json:"prev_cursor"`
	HasNext    bool    `json:"has_next"`
	HasPrev    bool    `json:"has_prev"`
	Limit      int     `json:"limit"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

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

func RespondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func RespondSuccess(w http.ResponseWriter, data interface{}) {
	RespondJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

func RespondCreated(w http.ResponseWriter, data interface{}) {
	RespondJSON(w, http.StatusCreated, APIResponse{Success: true, Data: data})
}

func RespondPaginated(w http.ResponseWriter, data interface{}, page, perPage, total int) {
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 0 {
		totalPages = 0
	}
	RespondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta: &PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func RespondCursorPaginated(w http.ResponseWriter, data interface{}, nextCursor, prevCursor *string, hasNext, hasPrev bool, limit int) {
	RespondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta: CursorMeta{
			NextCursor: nextCursor,
			PrevCursor: prevCursor,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
			Limit:      limit,
		},
	})
}

func RespondError(w http.ResponseWriter, status int, code, message string) {
	RespondJSON(w, status, APIResponse{
		Success: false,
		Error:   &ErrorBody{Code: code, Message: message},
	})
}

func RespondBadRequest(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", message)
}

func RespondUnauthorized(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func RespondForbidden(w http.ResponseWriter, code, message string) {
	RespondError(w, http.StatusForbidden, code, message)
}

func RespondNotFound(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusNotFound, "NOT_FOUND", message)
}

func RespondConflict(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusConflict, "CONFLICT", message)
}

func RespondInternalError(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

func RespondInternalErrorFromRequest(w http.ResponseWriter, r *http.Request, message string) {
	if IsProduction {
		RespondInternalError(w, "internal server error")
		return
	}
	resp := APIResponse{
		Success: false,
		Error:   &ErrorBody{Code: "INTERNAL_ERROR", Message: message},
	}
	if info, ok := GetErrorInfo(r.Context()); ok {
		resp.Stack = info.Stack
	}
	RespondJSON(w, http.StatusInternalServerError, resp)
}

func MapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		RespondNotFound(w, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists) || errors.Is(err, domain.ErrConflict):
		RespondConflict(w, err.Error())
	case errors.Is(err, domain.ErrValidation):
		RespondBadRequest(w, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		RespondForbidden(w, "FORBIDDEN", err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		RespondUnauthorized(w, err.Error())
	default:
		RespondInternalError(w, "internal server error")
	}
}

func MapErrorFromRequest(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		RespondNotFound(w, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists) || errors.Is(err, domain.ErrConflict):
		RespondConflict(w, err.Error())
	case errors.Is(err, domain.ErrValidation):
		RespondBadRequest(w, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		RespondForbidden(w, "FORBIDDEN", err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		RespondUnauthorized(w, err.Error())
	default:
		stack := string(debug.Stack())
		ctx := SetErrorInfo(r.Context(), err, stack)
		RespondInternalErrorFromRequest(w, r.WithContext(ctx), "internal server error")
	}
}
