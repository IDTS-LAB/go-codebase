package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	coreDomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/application/dto"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/repository"
	"github.com/google/uuid"
)

var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrTenantExists   = errors.New("tenant already exists")
)

type TenantService struct {
	repo repository.TenantRepository
}

func NewTenantService(repo repository.TenantRepository) *TenantService {
	return &TenantService{repo: repo}
}

func (s *TenantService) Create(ctx context.Context, req dto.CreateTenantRequest) (dto.TenantResponse, error) {
	existing, _ := s.repo.GetBySlug(ctx, req.Slug)
	if existing != nil {
		return dto.TenantResponse{}, coreDomain.NewDomainError(coreDomain.ErrAlreadyExists, "TENANT_EXISTS", "tenant with this slug already exists")
	}

	now := time.Now()
	settings := req.Settings
	if settings == nil {
		settings = json.RawMessage("{}")
	}

	tenant := &entity.Tenant{
		ID:        uuid.New(),
		Name:      req.Name,
		Slug:      req.Slug,
		Domain:    req.Domain,
		Settings:  settings,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, tenant); err != nil {
		return dto.TenantResponse{}, err
	}

	return toResponse(tenant), nil
}

func (s *TenantService) GetByID(ctx context.Context, id uuid.UUID) (dto.TenantResponse, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.TenantResponse{}, coreDomain.NewDomainError(coreDomain.ErrNotFound, "TENANT_NOT_FOUND", "tenant not found")
	}

	return toResponse(tenant), nil
}

func (s *TenantService) List(ctx context.Context, page, perPage int) (dto.TenantListResponse, error) {
	offset := (page - 1) * perPage
	tenants, total, err := s.repo.List(ctx, offset, perPage)
	if err != nil {
		return dto.TenantListResponse{}, err
	}

	responses := make([]dto.TenantResponse, len(tenants))
	for i, t := range tenants {
		responses[i] = toResponse(&t)
	}

	return dto.TenantListResponse{Tenants: responses, Total: total}, nil
}

func (s *TenantService) Update(ctx context.Context, id uuid.UUID, name *string, domain *string, settings json.RawMessage, isActive *bool) (dto.TenantResponse, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.TenantResponse{}, coreDomain.NewDomainError(coreDomain.ErrNotFound, "TENANT_NOT_FOUND", "tenant not found")
	}

	if name != nil {
		tenant.Name = *name
	}
	if domain != nil {
		tenant.Domain = domain
	}
	if settings != nil {
		tenant.Settings = settings
	}
	if isActive != nil {
		tenant.IsActive = *isActive
	}
	tenant.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tenant); err != nil {
		return dto.TenantResponse{}, err
	}

	return toResponse(tenant), nil
}

func (s *TenantService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return coreDomain.NewDomainError(coreDomain.ErrNotFound, "TENANT_NOT_FOUND", "tenant not found")
	}

	return s.repo.Delete(ctx, id)
}

func toResponse(t *entity.Tenant) dto.TenantResponse {
	return dto.TenantResponse{
		ID:        t.ID.String(),
		Name:      t.Name,
		Slug:      t.Slug,
		Domain:    t.Domain,
		Settings:  t.Settings,
		IsActive:  t.IsActive,
		CreatedAt: t.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339Nano),
	}
}
