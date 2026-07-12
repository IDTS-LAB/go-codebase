package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
	page := 1
	perPage := 20
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		fmt.Sscanf(pp, "%d", &perPage)
	}
	if perPage > 100 {
		perPage = 100
	}
	resp, err := h.queryBus.Ask(r.Context(), query.ListTenantsQuery{Page: page, PerPage: perPage})
	if err != nil {
		utils.HandlePaginated(w, nil, 0, 0, 0, err)
		return
	}
	listResp := resp.(dto.TenantListResponse)
	utils.HandlePaginated(w, listResp.Tenants, page, perPage, listResp.Total, nil)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
