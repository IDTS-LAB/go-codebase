package entity

import "github.com/IDTS-LAB/go-codebase/internal/core/domain"

type User struct {
	domain.Entity
	Email    string `json:"email"`
	Password string `json:"-"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

func NewUser(email, password, name string) *User {
	return &User{
		Entity:   domain.NewEntity(),
		Email:    email,
		Password: password,
		Name:     name,
		IsActive: true,
	}
}
