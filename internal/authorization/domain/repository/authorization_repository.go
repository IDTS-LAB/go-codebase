package repository

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/google/uuid"
)

type RoleRepository interface {
	Create(ctx context.Context, role *entity.Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Role, error)
	GetByName(ctx context.Context, name string) (*entity.Role, error)
	GetAll(ctx context.Context, offset, limit int) ([]*entity.Role, int, error)
	Update(ctx context.Context, role *entity.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type PermissionRepository interface {
	Create(ctx context.Context, perm *entity.Permission) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error)
	GetByName(ctx context.Context, name string) (*entity.Permission, error)
	GetAll(ctx context.Context, offset, limit int) ([]*entity.Permission, int, error)
	Update(ctx context.Context, perm *entity.Permission) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type UserRoleRepository interface {
	Assign(ctx context.Context, ur entity.UserRole) error
	Remove(ctx context.Context, userID, roleID uuid.UUID) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]entity.UserRole, error)
	GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error)
}

type RolePermissionRepository interface {
	Assign(ctx context.Context, rp entity.RolePermission) error
	Remove(ctx context.Context, roleID, permissionID uuid.UUID) error
	GetByRoleID(ctx context.Context, roleID uuid.UUID) ([]entity.RolePermission, error)
	GetPermissionsByRoleID(ctx context.Context, roleID uuid.UUID) ([]*entity.Permission, error)
}
