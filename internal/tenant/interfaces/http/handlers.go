package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/utils"
	"github.com/IDTS-LAB/go-codebase/internal/shared/validator"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/dto"
	appService "github.com/IDTS-LAB/go-codebase/internal/tenant/application/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *appService.TenantService
	v   *validator.Validator
}

func NewHandler(svc *appService.TenantService, v *validator.Validator) *Handler {
	return &Handler{svc: svc, v: v}
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
	resp, err := h.svc.Create(r.Context(), req)
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
	resp, err := h.svc.List(r.Context(), page, perPage)
	utils.HandlePaginated(w, resp.Tenants, page, perPage, resp.Total, err)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondBadRequest(w, "invalid tenant ID")
		return
	}
	resp, err := h.svc.GetByID(r.Context(), id)
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
	resp, err := h.svc.Update(r.Context(), id, req.Name, req.Domain, req.Settings, req.IsActive)
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
	err = h.svc.Delete(r.Context(), id)
	if err != nil && errors.Is(err, domain.ErrNotFound) {
		utils.RespondNotFound(w, "tenant not found")
		return
	}
	utils.HandleNoContent(w, err)
}
