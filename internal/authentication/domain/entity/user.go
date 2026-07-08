package entity

import (
	"time"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
)

type User struct {
	domain.Entity
	Email              string     `json:"email"`
	Password           string     `json:"-"`
	Name               string     `json:"name"`
	IsActive           bool       `json:"is_active"`
	FailedLoginAttempts int       `json:"failed_login_attempts"`
	LockedUntil        *time.Time `json:"locked_until,omitempty"`
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

func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

func (u *User) Lock(duration time.Duration) {
	u.FailedLoginAttempts++
	lockedUntil := time.Now().Add(duration)
	u.LockedUntil = &lockedUntil
}

func (u *User) Unlock() {
	u.FailedLoginAttempts = 0
	u.LockedUntil = nil
}
