package entity

import "github.com/google/uuid"

type RolePermission struct {
	RoleID       uuid.UUID `json:"role_id"`
	PermissionID uuid.UUID `json:"permission_id"`
}

func NewRolePermission(roleID, permissionID uuid.UUID) RolePermission {
	return RolePermission{RoleID: roleID, PermissionID: permissionID}
}
