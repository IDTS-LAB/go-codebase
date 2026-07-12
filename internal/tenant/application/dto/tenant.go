package dto

import "encoding/json"

type CreateTenantRequest struct {
	Name     string          `json:"name" validate:"required"`
	Slug     string          `json:"slug" validate:"required"`
	Domain   *string         `json:"domain"`
	Settings json.RawMessage `json:"settings"`
}

type UpdateTenantRequest struct {
	Name     *string         `json:"name"`
	Domain   *string         `json:"domain"`
	Settings json.RawMessage `json:"settings"`
	IsActive *bool           `json:"is_active"`
}

type TenantResponse struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Slug      string          `json:"slug"`
	Domain    *string         `json:"domain,omitempty"`
	Settings  json.RawMessage `json:"settings"`
	IsActive  bool            `json:"is_active"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

type TenantListResponse struct {
	Tenants []TenantResponse `json:"tenants"`
	Total   int              `json:"total"`
}
