package service

import (
	"context"
	"testing"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/repository"
	coredomain "github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRoleRepo struct {
	mock.Mock
}

func (m *mockRoleRepo) Create(ctx context.Context, role *entity.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *mockRoleRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Role), args.Error(1)
}

func (m *mockRoleRepo) GetByName(ctx context.Context, name string) (*entity.Role, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Role), args.Error(1)
}

func (m *mockRoleRepo) GetAll(ctx context.Context, offset, limit int) ([]*entity.Role, int, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*entity.Role), args.Int(1), args.Error(2)
}

func (m *mockRoleRepo) Update(ctx context.Context, role *entity.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *mockRoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockPermRepo struct {
	mock.Mock
}

func (m *mockPermRepo) Create(ctx context.Context, perm *entity.Permission) error {
	args := m.Called(ctx, perm)
	return args.Error(0)
}

func (m *mockPermRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Permission), args.Error(1)
}

func (m *mockPermRepo) GetByName(ctx context.Context, name string) (*entity.Permission, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Permission), args.Error(1)
}

func (m *mockPermRepo) GetAll(ctx context.Context, offset, limit int) ([]*entity.Permission, int, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*entity.Permission), args.Int(1), args.Error(2)
}

func (m *mockPermRepo) Update(ctx context.Context, perm *entity.Permission) error {
	args := m.Called(ctx, perm)
	return args.Error(0)
}

func (m *mockPermRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockUserRoleRepo struct {
	mock.Mock
}

func (m *mockUserRoleRepo) Assign(ctx context.Context, ur entity.UserRole) error {
	args := m.Called(ctx, ur)
	return args.Error(0)
}

func (m *mockUserRoleRepo) Remove(ctx context.Context, userID, roleID uuid.UUID) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *mockUserRoleRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]entity.UserRole, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]entity.UserRole), args.Error(1)
}

func (m *mockUserRoleRepo) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*entity.Role), args.Error(1)
}

type mockRolePermRepo struct {
	mock.Mock
}

func (m *mockRolePermRepo) Assign(ctx context.Context, rp entity.RolePermission) error {
	args := m.Called(ctx, rp)
	return args.Error(0)
}

func (m *mockRolePermRepo) Remove(ctx context.Context, roleID, permissionID uuid.UUID) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *mockRolePermRepo) GetByRoleID(ctx context.Context, roleID uuid.UUID) ([]entity.RolePermission, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).([]entity.RolePermission), args.Error(1)
}

func (m *mockRolePermRepo) GetPermissionsByRoleID(ctx context.Context, roleID uuid.UUID) ([]*entity.Permission, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).([]*entity.Permission), args.Error(1)
}

type mockEnforcer struct {
	mock.Mock
}

func (m *mockEnforcer) ReloadPolicies(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockEnforcer) ReloadUserPolicies(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockEnforcer) Enforce(userID uuid.UUID, resource, action string) (bool, error) {
	args := m.Called(userID, resource, action)
	return args.Bool(0), args.Error(1)
}

func TestCreateRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		roleRepo := new(mockRoleRepo)
		svc := &AuthorizationService{
			roleRepo: roleRepo,
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		roleRepo.On("GetByName", mock.Anything, "admin").Return(nil, nil)
		roleRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *entity.Role) bool {
			return r.Name == "admin" && r.Description == "Administrator"
		})).Return(nil)

		role, err := svc.CreateRole(context.Background(), "admin", "Administrator")

		assert.NoError(t, err)
		assert.Equal(t, "admin", role.Name)
		assert.Equal(t, "Administrator", role.Description)
		roleRepo.AssertExpectations(t)
	})

	t.Run("duplicate name error", func(t *testing.T) {
		roleRepo := new(mockRoleRepo)
		svc := &AuthorizationService{
			roleRepo: roleRepo,
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		existing := &entity.Role{Name: "admin"}
		roleRepo.On("GetByName", mock.Anything, "admin").Return(existing, nil)

		role, err := svc.CreateRole(context.Background(), "admin", "Administrator")

		assert.Error(t, err)
		assert.ErrorIs(t, err, coredomain.ErrConflict)
		assert.Nil(t, role)
		roleRepo.AssertExpectations(t)
	})
}

func TestListRoles(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		roleRepo := new(mockRoleRepo)
		svc := &AuthorizationService{
			roleRepo: roleRepo,
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		expected := []*entity.Role{
			{Name: "admin"},
			{Name: "user"},
		}
		roleRepo.On("GetAll", mock.Anything, 0, 20).Return(expected, 2, nil)

		roles, total, err := svc.ListRoles(context.Background(), 1, 20)

		assert.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, roles, 2)
		roleRepo.AssertExpectations(t)
	})

	t.Run("empty result", func(t *testing.T) {
		roleRepo := new(mockRoleRepo)
		svc := &AuthorizationService{
			roleRepo: roleRepo,
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		roleRepo.On("GetAll", mock.Anything, 0, 20).Return([]*entity.Role{}, 0, nil)

		roles, total, err := svc.ListRoles(context.Background(), 1, 20)

		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, roles)
		roleRepo.AssertExpectations(t)
	})
}

func TestGetRole(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		roleRepo := new(mockRoleRepo)
		svc := &AuthorizationService{
			roleRepo: roleRepo,
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		roleID := uuid.New()
		expected := &entity.Role{
			Entity: coredomain.Entity{ID: roleID},
			Name:   "admin",
		}
		roleRepo.On("GetByID", mock.Anything, roleID).Return(expected, nil)

		role, err := svc.GetRole(context.Background(), roleID)

		assert.NoError(t, err)
		assert.Equal(t, "admin", role.Name)
		assert.Equal(t, roleID, role.ID)
		roleRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		roleRepo := new(mockRoleRepo)
		svc := &AuthorizationService{
			roleRepo: roleRepo,
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		roleID := uuid.New()
		roleRepo.On("GetByID", mock.Anything, roleID).Return(nil, coredomain.ErrNotFound)

		role, err := svc.GetRole(context.Background(), roleID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, coredomain.ErrNotFound)
		assert.Nil(t, role)
		roleRepo.AssertExpectations(t)
	})
}

func TestUpdateRole(t *testing.T) {
	t.Run("success with all fields", func(t *testing.T) {
		roleRepo := new(mockRoleRepo)
		svc := &AuthorizationService{
			roleRepo: roleRepo,
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		roleID := uuid.New()
		existing := &entity.Role{
			Entity:      coredomain.Entity{ID: roleID},
			Name:        "old",
			Description: "old desc",
		}
		roleRepo.On("GetByID", mock.Anything, roleID).Return(existing, nil)
		roleRepo.On("Update", mock.Anything, mock.MatchedBy(func(r *entity.Role) bool {
			return r.Name == "new" && r.Description == "new desc" && r.ID == roleID
		})).Return(nil)

		role, err := svc.UpdateRole(context.Background(), roleID, "new", "new desc")

		assert.NoError(t, err)
		assert.Equal(t, "new", role.Name)
		assert.Equal(t, "new desc", role.Description)
		roleRepo.AssertExpectations(t)
	})

	t.Run("updates only non-empty fields", func(t *testing.T) {
		roleRepo := new(mockRoleRepo)
		svc := &AuthorizationService{
			roleRepo: roleRepo,
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		roleID := uuid.New()
		existing := &entity.Role{
			Entity:      coredomain.Entity{ID: roleID},
			Name:        "admin",
			Description: "old desc",
		}
		roleRepo.On("GetByID", mock.Anything, roleID).Return(existing, nil)
		roleRepo.On("Update", mock.Anything, mock.MatchedBy(func(r *entity.Role) bool {
			return r.Name == "admin" && r.Description == "new desc"
		})).Return(nil)

		role, err := svc.UpdateRole(context.Background(), roleID, "", "new desc")

		assert.NoError(t, err)
		assert.Equal(t, "admin", role.Name)
		assert.Equal(t, "new desc", role.Description)
		roleRepo.AssertExpectations(t)
	})
}

func TestDeleteRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		roleRepo := new(mockRoleRepo)
		svc := &AuthorizationService{
			roleRepo: roleRepo,
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		roleID := uuid.New()
		roleRepo.On("Delete", mock.Anything, roleID).Return(nil)

		err := svc.DeleteRole(context.Background(), roleID)

		assert.NoError(t, err)
		roleRepo.AssertExpectations(t)
	})
}

func TestCreatePermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		permRepo := new(mockPermRepo)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: permRepo,
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		permRepo.On("GetByName", mock.Anything, "read").Return(nil, nil)
		permRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *entity.Permission) bool {
			return p.Name == "read" && p.Resource == "users" && p.Action == "read"
		})).Return(nil)

		perm, err := svc.CreatePermission(context.Background(), "read", "Read users", "users", "read")

		assert.NoError(t, err)
		assert.Equal(t, "read", perm.Name)
		assert.Equal(t, "users", perm.Resource)
		assert.Equal(t, "read", perm.Action)
		permRepo.AssertExpectations(t)
	})

	t.Run("duplicate name error", func(t *testing.T) {
		permRepo := new(mockPermRepo)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: permRepo,
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		existing := &entity.Permission{Name: "read"}
		permRepo.On("GetByName", mock.Anything, "read").Return(existing, nil)

		perm, err := svc.CreatePermission(context.Background(), "read", "Read users", "users", "read")

		assert.Error(t, err)
		assert.ErrorIs(t, err, coredomain.ErrConflict)
		assert.Nil(t, perm)
		permRepo.AssertExpectations(t)
	})
}

func TestListPermissions(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		permRepo := new(mockPermRepo)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: permRepo,
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		expected := []*entity.Permission{
			{Name: "read", Resource: "users", Action: "read"},
		}
		permRepo.On("GetAll", mock.Anything, 0, 20).Return(expected, 1, nil)

		perms, total, err := svc.ListPermissions(context.Background(), 1, 20)

		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, perms, 1)
		permRepo.AssertExpectations(t)
	})
}

func TestGetPermission(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		permRepo := new(mockPermRepo)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: permRepo,
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		permID := uuid.New()
		expected := &entity.Permission{
			Entity:   coredomain.Entity{ID: permID},
			Name:     "read",
			Resource: "users",
			Action:   "read",
		}
		permRepo.On("GetByID", mock.Anything, permID).Return(expected, nil)

		perm, err := svc.GetPermission(context.Background(), permID)

		assert.NoError(t, err)
		assert.Equal(t, "read", perm.Name)
		assert.Equal(t, permID, perm.ID)
		permRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		permRepo := new(mockPermRepo)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: permRepo,
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		permID := uuid.New()
		permRepo.On("GetByID", mock.Anything, permID).Return(nil, coredomain.ErrNotFound)

		perm, err := svc.GetPermission(context.Background(), permID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, coredomain.ErrNotFound)
		assert.Nil(t, perm)
		permRepo.AssertExpectations(t)
	})
}

func TestUpdatePermission(t *testing.T) {
	t.Run("success with all fields", func(t *testing.T) {
		permRepo := new(mockPermRepo)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: permRepo,
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		permID := uuid.New()
		existing := &entity.Permission{
			Entity:      coredomain.Entity{ID: permID},
			Name:        "old",
			Description: "old desc",
			Resource:    "old-res",
			Action:      "old-act",
		}
		permRepo.On("GetByID", mock.Anything, permID).Return(existing, nil)
		permRepo.On("Update", mock.Anything, mock.MatchedBy(func(p *entity.Permission) bool {
			return p.Name == "new" && p.Resource == "new-res" && p.Action == "new-act"
		})).Return(nil)

		perm, err := svc.UpdatePermission(context.Background(), permID, "new", "new desc", "new-res", "new-act")

		assert.NoError(t, err)
		assert.Equal(t, "new", perm.Name)
		assert.Equal(t, "new-res", perm.Resource)
		assert.Equal(t, "new-act", perm.Action)
		permRepo.AssertExpectations(t)
	})
}

func TestDeletePermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		permRepo := new(mockPermRepo)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: permRepo,
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		permID := uuid.New()
		permRepo.On("Delete", mock.Anything, permID).Return(nil)

		err := svc.DeletePermission(context.Background(), permID)

		assert.NoError(t, err)
		permRepo.AssertExpectations(t)
	})
}

func TestAssignRoleToUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		roleRepo := new(mockRoleRepo)
		userRoleRepo := new(mockUserRoleRepo)
		enf := new(mockEnforcer)
		svc := &AuthorizationService{
			roleRepo: roleRepo,
			permRepo: new(mockPermRepo),
			userRoleRepo: userRoleRepo,
			rolePermRepo: new(mockRolePermRepo),
			enforcer: enf,
		}

		userID := uuid.New()
		roleID := uuid.New()
		roleRepo.On("GetByID", mock.Anything, roleID).Return(&entity.Role{
			Entity: coredomain.Entity{ID: roleID},
			Name:   "admin",
		}, nil)
		userRoleRepo.On("Assign", mock.Anything, mock.MatchedBy(func(ur entity.UserRole) bool {
			return ur.UserID == userID && ur.RoleID == roleID
		})).Return(nil)
		enf.On("ReloadUserPolicies", mock.Anything, userID).Return(nil)

		err := svc.AssignRoleToUser(context.Background(), userID, roleID)

		assert.NoError(t, err)
		roleRepo.AssertExpectations(t)
		userRoleRepo.AssertExpectations(t)
		enf.AssertExpectations(t)
	})
}

func TestRemoveRoleFromUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		userRoleRepo := new(mockUserRoleRepo)
		enf := new(mockEnforcer)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: new(mockPermRepo),
			userRoleRepo: userRoleRepo,
			rolePermRepo: new(mockRolePermRepo),
			enforcer: enf,
		}

		userID := uuid.New()
		roleID := uuid.New()
		userRoleRepo.On("Remove", mock.Anything, userID, roleID).Return(nil)
		enf.On("ReloadUserPolicies", mock.Anything, userID).Return(nil)

		err := svc.RemoveRoleFromUser(context.Background(), userID, roleID)

		assert.NoError(t, err)
		userRoleRepo.AssertExpectations(t)
		enf.AssertExpectations(t)
	})
}

func TestGetUserRoles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		userRoleRepo := new(mockUserRoleRepo)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: new(mockPermRepo),
			userRoleRepo: userRoleRepo,
			rolePermRepo: new(mockRolePermRepo),
			enforcer: new(mockEnforcer),
		}

		userID := uuid.New()
		expected := []*entity.Role{
			{Name: "admin"},
			{Name: "user"},
		}
		userRoleRepo.On("GetRolesByUserID", mock.Anything, userID).Return(expected, nil)

		roles, err := svc.GetUserRoles(context.Background(), userID)

		assert.NoError(t, err)
		assert.Len(t, roles, 2)
		userRoleRepo.AssertExpectations(t)
	})
}

func TestAssignPermissionToRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		roleRepo := new(mockRoleRepo)
		permRepo := new(mockPermRepo)
		rolePermRepo := new(mockRolePermRepo)
		enf := new(mockEnforcer)
		svc := &AuthorizationService{
			roleRepo: roleRepo,
			permRepo: permRepo,
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: rolePermRepo,
			enforcer: enf,
		}

		roleID := uuid.New()
		permID := uuid.New()
		roleRepo.On("GetByID", mock.Anything, roleID).Return(&entity.Role{
			Entity: coredomain.Entity{ID: roleID},
			Name:   "admin",
		}, nil)
		permRepo.On("GetByID", mock.Anything, permID).Return(&entity.Permission{
			Entity: coredomain.Entity{ID: permID},
			Name:   "read",
		}, nil)
		rolePermRepo.On("Assign", mock.Anything, mock.MatchedBy(func(rp entity.RolePermission) bool {
			return rp.RoleID == roleID && rp.PermissionID == permID
		})).Return(nil)
		enf.On("ReloadPolicies", mock.Anything).Return(nil)

		err := svc.AssignPermissionToRole(context.Background(), roleID, permID)

		assert.NoError(t, err)
		roleRepo.AssertExpectations(t)
		permRepo.AssertExpectations(t)
		rolePermRepo.AssertExpectations(t)
		enf.AssertExpectations(t)
	})
}

func TestRemovePermissionFromRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		rolePermRepo := new(mockRolePermRepo)
		enf := new(mockEnforcer)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: rolePermRepo,
			enforcer: enf,
		}

		roleID := uuid.New()
		permID := uuid.New()
		rolePermRepo.On("Remove", mock.Anything, roleID, permID).Return(nil)
		enf.On("ReloadPolicies", mock.Anything).Return(nil)

		err := svc.RemovePermissionFromRole(context.Background(), roleID, permID)

		assert.NoError(t, err)
		rolePermRepo.AssertExpectations(t)
		enf.AssertExpectations(t)
	})
}

func TestGetRolePermissions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		rolePermRepo := new(mockRolePermRepo)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: rolePermRepo,
			enforcer: new(mockEnforcer),
		}

		roleID := uuid.New()
		expected := []*entity.Permission{
			{Name: "read", Resource: "users", Action: "read"},
		}
		rolePermRepo.On("GetPermissionsByRoleID", mock.Anything, roleID).Return(expected, nil)

		perms, err := svc.GetRolePermissions(context.Background(), roleID)

		assert.NoError(t, err)
		assert.Len(t, perms, 1)
		rolePermRepo.AssertExpectations(t)
	})
}

func TestCheckPermission(t *testing.T) {
	t.Run("allowed", func(t *testing.T) {
		enf := new(mockEnforcer)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: enf,
		}

		userID := uuid.New()
		enf.On("Enforce", userID, "users", "read").Return(true, nil)

		allowed, err := svc.CheckPermission(context.Background(), userID, "users", "read")

		assert.NoError(t, err)
		assert.True(t, allowed)
		enf.AssertExpectations(t)
	})

	t.Run("denied", func(t *testing.T) {
		enf := new(mockEnforcer)
		svc := &AuthorizationService{
			roleRepo: new(mockRoleRepo),
			permRepo: new(mockPermRepo),
			userRoleRepo: new(mockUserRoleRepo),
			rolePermRepo: new(mockRolePermRepo),
			enforcer: enf,
		}

		userID := uuid.New()
		enf.On("Enforce", userID, "users", "write").Return(false, nil)

		allowed, err := svc.CheckPermission(context.Background(), userID, "users", "write")

		assert.NoError(t, err)
		assert.False(t, allowed)
		enf.AssertExpectations(t)
	})
}

var (
	_ repository.RoleRepository         = (*mockRoleRepo)(nil)
	_ repository.PermissionRepository   = (*mockPermRepo)(nil)
	_ repository.UserRoleRepository     = (*mockUserRoleRepo)(nil)
	_ repository.RolePermissionRepository = (*mockRolePermRepo)(nil)
)
