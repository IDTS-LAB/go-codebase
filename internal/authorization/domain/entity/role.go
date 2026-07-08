package entity

import "github.com/IDTS-LAB/go-codebase/internal/core/domain"

type Role struct {
	domain.Entity
	Name        string `json:"name"`
	Description string `json:"description"`
}

func NewRole(name, description string) *Role {
	return &Role{
		Entity:      domain.NewEntity(),
		Name:        name,
		Description: description,
	}
}
