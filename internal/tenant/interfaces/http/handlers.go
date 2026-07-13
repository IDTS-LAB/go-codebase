package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/cqrs"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/command"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/query"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	commandBus cqrs.CommandBus
	queryBus   cqrs.QueryBus
	v          *validator.Validator
}

func NewHandler(commandBus cqrs.CommandBus, queryBus cqrs.QueryBus, v *validator.Validator) *Handler {
	return &Handler{commandBus: commandBus, queryBus: queryBus, v: v}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	if err := h.v.Validate(req); err != nil {
		utils.RespondBadRequest(w, err.Error())
		return
	}
	resp, err := h.commandBus.Dispatch(r.Context(), command.CreateTenantCommand{
		Name:     req.Name,
		Slug:     req.Slug,
		Domain:   req.Domain,
		Settings: req.Settings,
	})
	if err != nil && errors.Is(err, domain.ErrAlreadyExists) {
		utils.RespondConflict(w, err.Error())
		return
	}
	utils.HandleCreated(w, resp, err)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	cursorStr := r.URL.Query().Get("cursor")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	var cursor *string
	if cursorStr != "" {
		cursor = &cursorStr
	}

	resp, err := h.queryBus.Ask(r.Context(), query.ListTenantsQuery{Cursor: cursor, Limit: limit})
	if err != nil {
		utils.MapErrorFromRequest(w, r, err)
		return
	}
	listResp := resp.(dto.TenantListResponse)
	utils.RespondCursorPaginated(w, listResp.Tenants, listResp.NextCursor, listResp.PrevCursor, listResp.HasNext, listResp.HasPrev, listResp.Limit)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid tenant ID")
		return
	}
	resp, err := h.queryBus.Ask(r.Context(), query.GetTenantQuery{ID: id})
	if err != nil && errors.Is(err, domain.ErrNotFound) {
		utils.RespondNotFound(w, "tenant not found")
		return
	}
	utils.Handle(w, resp, err)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid tenant ID")
		return
	}
	var req dto.UpdateTenantRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondBadRequest(w, "invalid request body")
		return
	}
	resp, err := h.commandBus.Dispatch(r.Context(), command.UpdateTenantCommand{
		ID:       id,
		Name:     req.Name,
		Domain:   req.Domain,
		Settings: req.Settings,
		IsActive: req.IsActive,
	})
	if err != nil && errors.Is(err, domain.ErrNotFound) {
		utils.RespondNotFound(w, "tenant not found")
		return
	}
	utils.Handle(w, resp, err)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid tenant ID")
		return
	}
	_, err = h.commandBus.Dispatch(r.Context(), command.DeleteTenantCommand{ID: id})
	if err != nil && errors.Is(err, domain.ErrNotFound) {
		utils.RespondNotFound(w, "tenant not found")
		return
	}
	utils.HandleNoContent(w, err)
}
