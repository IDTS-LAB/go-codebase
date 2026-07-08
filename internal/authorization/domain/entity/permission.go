package entity

import "github.com/IDTS-LAB/go-codebase/internal/core/domain"

type Permission struct {
	domain.Entity
	Name        string `json:"name"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
}

func NewPermission(name, description, resource, action string) *Permission {
	return &Permission{
		Entity:      domain.NewEntity(),
		Name:        name,
		Description: description,
		Resource:    resource,
		Action:      action,
	}
}
