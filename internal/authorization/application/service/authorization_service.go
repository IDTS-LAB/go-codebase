package service

import (
	"context"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/casbin"
	coredomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
)

type AuthorizationService struct {
	roleRepo     repository.RoleRepository
	permRepo     repository.PermissionRepository
	userRoleRepo repository.UserRoleRepository
	rolePermRepo repository.RolePermissionRepository
	enforcer     *casbin.Enforcer
}

func NewAuthorizationService(
	roleRepo repository.RoleRepository,
	permRepo repository.PermissionRepository,
	userRoleRepo repository.UserRoleRepository,
	rolePermRepo repository.RolePermissionRepository,
	enforcer *casbin.Enforcer,
) *AuthorizationService {
	return &AuthorizationService{
		roleRepo:     roleRepo,
		permRepo:     permRepo,
		userRoleRepo: userRoleRepo,
		rolePermRepo: rolePermRepo,
		enforcer:     enforcer,
	}
}

func (s *AuthorizationService) CreateRole(ctx context.Context, name, description string) (*entity.Role, error) {
	existing, _ := s.roleRepo.GetByName(ctx, name)
	if existing != nil {
		return nil, coredomain.ErrConflict
	}
	role := entity.NewRole(name, description)
	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *AuthorizationService) GetRole(ctx context.Context, id uuid.UUID) (*entity.Role, error) {
	return s.roleRepo.GetByID(ctx, id)
}

func (s *AuthorizationService) ListRoles(ctx context.Context, page, perPage int) ([]*entity.Role, int, error) {
	offset := (page - 1) * perPage
	return s.roleRepo.GetAll(ctx, offset, perPage)
}

func (s *AuthorizationService) UpdateRole(ctx context.Context, id uuid.UUID, name, description string) (*entity.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != "" {
		role.Name = name
	}
	if description != "" {
		role.Description = description
	}
	role.Touch()
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *AuthorizationService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return s.roleRepo.Delete(ctx, id)
}

func (s *AuthorizationService) CreatePermission(ctx context.Context, name, description, resource, action string) (*entity.Permission, error) {
	existing, _ := s.permRepo.GetByName(ctx, name)
	if existing != nil {
		return nil, coredomain.ErrConflict
	}
	perm := entity.NewPermission(name, description, resource, action)
	if err := s.permRepo.Create(ctx, perm); err != nil {
		return nil, err
	}
	return perm, nil
}

func (s *AuthorizationService) GetPermission(ctx context.Context, id uuid.UUID) (*entity.Permission, error) {
	return s.permRepo.GetByID(ctx, id)
}

func (s *AuthorizationService) ListPermissions(ctx context.Context, page, perPage int) ([]*entity.Permission, int, error) {
	offset := (page - 1) * perPage
	return s.permRepo.GetAll(ctx, offset, perPage)
}

func (s *AuthorizationService) UpdatePermission(ctx context.Context, id uuid.UUID, name, description, resource, action string) (*entity.Permission, error) {
	perm, err := s.permRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != "" {
		perm.Name = name
	}
	if description != "" {
		perm.Description = description
	}
	if resource != "" {
		perm.Resource = resource
	}
	if action != "" {
		perm.Action = action
	}
	perm.Touch()
	if err := s.permRepo.Update(ctx, perm); err != nil {
		return nil, err
	}
	return perm, nil
}

func (s *AuthorizationService) DeletePermission(ctx context.Context, id uuid.UUID) error {
	return s.permRepo.Delete(ctx, id)
}

func (s *AuthorizationService) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if err := s.userRoleRepo.Assign(ctx, entity.NewUserRole(userID, roleID)); err != nil {
		return err
	}
	return s.enforcer.ReloadUserPolicies(ctx, userID)
}

func (s *AuthorizationService) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	if err := s.userRoleRepo.Remove(ctx, userID, roleID); err != nil {
		return err
	}
	return s.enforcer.ReloadUserPolicies(ctx, userID)
}

func (s *AuthorizationService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error) {
	return s.userRoleRepo.GetRolesByUserID(ctx, userID)
}

func (s *AuthorizationService) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	_, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	_, err = s.permRepo.GetByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if err := s.rolePermRepo.Assign(ctx, entity.NewRolePermission(roleID, permissionID)); err != nil {
		return err
	}
	return s.enforcer.ReloadPolicies(ctx)
}

func (s *AuthorizationService) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	if err := s.rolePermRepo.Remove(ctx, roleID, permissionID); err != nil {
		return err
	}
	return s.enforcer.ReloadPolicies(ctx)
}

func (s *AuthorizationService) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*entity.Permission, error) {
	return s.rolePermRepo.GetPermissionsByRoleID(ctx, roleID)
}

func (s *AuthorizationService) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	return s.enforcer.Enforce(userID, resource, action)
}
