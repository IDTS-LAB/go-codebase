package entity

import "github.com/google/uuid"

type UserRole struct {
	UserID uuid.UUID `json:"user_id"`
	RoleID uuid.UUID `json:"role_id"`
}

func NewUserRole(userID, roleID uuid.UUID) UserRole {
	return UserRole{UserID: userID, RoleID: roleID}
}
