package utils

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
)

type APIResponse struct {
	Success bool            `json:"success"`
	Data    interface{}     `json:"data"`
	Meta    *PaginationMeta `json:"meta"`
	Error   *ErrorBody      `json:"error,omitempty"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
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
	json.NewEncoder(w).Encode(payload)
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
