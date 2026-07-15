package tests

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"github.com/IDTS-LAB/go-codebase/internal/authorization/domain/entity"
	authzPersistence "github.com/IDTS-LAB/go-codebase/internal/authorization/infrastructure/persistence"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/IDTS-LAB/go-codebase/internal/testhelper"
	"github.com/google/uuid"
)

var db *sql.DB

func TestMain(m *testing.M) {
	var cleanup func()
	db, cleanup = testhelper.SetupTestDB(m)
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func TestRoleRepository_CreateAndGetByID(t *testing.T) {
	repo := authzPersistence.NewRoleRepository(db, &tenantfilter.Config{})

	role := entity.NewRole(fmt.Sprintf("admin-role-%s", uuid.New().String()[:8]), "Administrator")
	err := repo.Create(context.Background(), role)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(context.Background(), role.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != role.Name {
		t.Errorf("expected name %s, got %s", role.Name, got.Name)
	}
	if got.Description != role.Description {
		t.Errorf("expected description %s, got %s", role.Description, got.Description)
	}
}

func TestRoleRepository_GetByName(t *testing.T) {
	repo := authzPersistence.NewRoleRepository(db, &tenantfilter.Config{})

	roleName := fmt.Sprintf("editor-role-%s", uuid.New().String()[:8])
	role := entity.NewRole(roleName, "Editor role")
	repo.Create(context.Background(), role)

	got, err := repo.GetByName(context.Background(), roleName)
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.ID != role.ID {
		t.Errorf("expected id %s, got %s", role.ID, got.ID)
	}
}

func TestRoleRepository_GetAll(t *testing.T) {
	repo := authzPersistence.NewRoleRepository(db, &tenantfilter.Config{})

	repo.Create(context.Background(), entity.NewRole(fmt.Sprintf("role-a-%s", uuid.New().String()[:8]), "A"))
	repo.Create(context.Background(), entity.NewRole(fmt.Sprintf("role-b-%s", uuid.New().String()[:8]), "B"))

	roles, _, _, _, _, err := repo.GetAll(context.Background(), nil, 20)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(roles) < 2 {
		t.Errorf("expected at least 2 roles, got %d", len(roles))
	}
}

func TestRoleRepository_Update(t *testing.T) {
	repo := authzPersistence.NewRoleRepository(db, &tenantfilter.Config{})

	beforeName := fmt.Sprintf("before-role-%s", uuid.New().String()[:8])
	afterName := fmt.Sprintf("after-role-%s", uuid.New().String()[:8])
	role := entity.NewRole(beforeName, "Before description")
	repo.Create(context.Background(), role)

	role.Name = afterName
	role.Description = "After description"
	role.Touch()
	err := repo.Update(context.Background(), role)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(context.Background(), role.ID)
	if got.Name != afterName {
		t.Errorf("expected name '%s', got %s", afterName, got.Name)
	}
}

func TestRoleRepository_Delete(t *testing.T) {
	repo := authzPersistence.NewRoleRepository(db, &tenantfilter.Config{})

	role := entity.NewRole(fmt.Sprintf("delete-role-%s", uuid.New().String()[:8]), "To be deleted")
	repo.Create(context.Background(), role)

	err := repo.Delete(context.Background(), role.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = repo.GetByID(context.Background(), role.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestPermissionRepository_CreateAndGetByID(t *testing.T) {
	repo := authzPersistence.NewPermissionRepository(db, &tenantfilter.Config{})

	perm := entity.NewPermission(fmt.Sprintf("read-users-%s", uuid.New().String()[:8]), "Read users", "users", "read")
	err := repo.Create(context.Background(), perm)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(context.Background(), perm.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != perm.Name {
		t.Errorf("expected name %s, got %s", perm.Name, got.Name)
	}
	if got.Resource != "users" {
		t.Errorf("expected resource 'users', got %s", got.Resource)
	}
}

func TestPermissionRepository_GetAll(t *testing.T) {
	repo := authzPersistence.NewPermissionRepository(db, &tenantfilter.Config{})

	repo.Create(context.Background(), entity.NewPermission(fmt.Sprintf("perm-a-%s", uuid.New().String()[:8]), "", "res", "read"))
	repo.Create(context.Background(), entity.NewPermission(fmt.Sprintf("perm-b-%s", uuid.New().String()[:8]), "", "res", "write"))

	perms, _, _, _, _, err := repo.GetAll(context.Background(), nil, 20)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(perms) < 2 {
		t.Errorf("expected at least 2 permissions, got %d", len(perms))
	}
}

func TestPermissionRepository_Delete(t *testing.T) {
	repo := authzPersistence.NewPermissionRepository(db, &tenantfilter.Config{})

	perm := entity.NewPermission(fmt.Sprintf("del-perm-%s", uuid.New().String()[:8]), "", "res", "read")
	repo.Create(context.Background(), perm)

	err := repo.Delete(context.Background(), perm.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = repo.GetByID(context.Background(), perm.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestUserRoleRepository_AssignAndGetByUserID(t *testing.T) {
	roleRepo := authzPersistence.NewRoleRepository(db, &tenantfilter.Config{})
	urRepo := authzPersistence.NewUserRoleRepository(db)

	role := entity.NewRole(fmt.Sprintf("user-role-test-%s", uuid.New().String()[:8]), "Test")
	roleRepo.Create(context.Background(), role)

	userID := uuid.New()
	err := urRepo.Assign(context.Background(), entity.NewUserRole(userID, role.ID))
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}

	roles, err := urRepo.GetRolesByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetRolesByUserID: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(roles))
	}
	if roles[0].ID != role.ID {
		t.Errorf("expected role id %s, got %s", role.ID, roles[0].ID)
	}
}

func TestUserRoleRepository_Remove(t *testing.T) {
	roleRepo := authzPersistence.NewRoleRepository(db, &tenantfilter.Config{})
	urRepo := authzPersistence.NewUserRoleRepository(db)

	role := entity.NewRole(fmt.Sprintf("remove-role-test-%s", uuid.New().String()[:8]), "To remove")
	roleRepo.Create(context.Background(), role)

	userID := uuid.New()
	urRepo.Assign(context.Background(), entity.NewUserRole(userID, role.ID))

	err := urRepo.Remove(context.Background(), userID, role.ID)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	roles, _ := urRepo.GetRolesByUserID(context.Background(), userID)
	if len(roles) != 0 {
		t.Error("expected no roles after remove")
	}
}

func TestRolePermissionRepository_AssignAndGetByRoleID(t *testing.T) {
	roleRepo := authzPersistence.NewRoleRepository(db, &tenantfilter.Config{})
	permRepo := authzPersistence.NewPermissionRepository(db, &tenantfilter.Config{})
	rpRepo := authzPersistence.NewRolePermissionRepository(db)

	role := entity.NewRole(fmt.Sprintf("rp-role-test-%s", uuid.New().String()[:8]), "Role for RP test")
	perm := entity.NewPermission(fmt.Sprintf("rp-perm-test-%s", uuid.New().String()[:8]), "", "res", "read")
	roleRepo.Create(context.Background(), role)
	permRepo.Create(context.Background(), perm)

	err := rpRepo.Assign(context.Background(), entity.NewRolePermission(role.ID, perm.ID))
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}

	perms, err := rpRepo.GetPermissionsByRoleID(context.Background(), role.ID)
	if err != nil {
		t.Fatalf("GetPermissionsByRoleID: %v", err)
	}
	if len(perms) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(perms))
	}
}

func TestRolePermissionRepository_Remove(t *testing.T) {
	roleRepo := authzPersistence.NewRoleRepository(db, &tenantfilter.Config{})
	permRepo := authzPersistence.NewPermissionRepository(db, &tenantfilter.Config{})
	rpRepo := authzPersistence.NewRolePermissionRepository(db)

	role := entity.NewRole(fmt.Sprintf("rp-remove-test-%s", uuid.New().String()[:8]), "RP remove")
	perm := entity.NewPermission(fmt.Sprintf("rp-remove-perm-%s", uuid.New().String()[:8]), "", "res", "read")
	roleRepo.Create(context.Background(), role)
	permRepo.Create(context.Background(), perm)

	rpRepo.Assign(context.Background(), entity.NewRolePermission(role.ID, perm.ID))

	err := rpRepo.Remove(context.Background(), role.ID, perm.ID)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	perms, _ := rpRepo.GetPermissionsByRoleID(context.Background(), role.ID)
	if len(perms) != 0 {
		t.Error("expected no permissions after remove")
	}
}

func TestRoleRepository_Create_DuplicateName(t *testing.T) {
	repo := authzPersistence.NewRoleRepository(db, &tenantfilter.Config{})
	name := fmt.Sprintf("unique-role-name-test-%s", uuid.New().String()[:8])

	r1 := entity.NewRole(name, "First")
	if err := repo.Create(context.Background(), r1); err != nil {
		t.Fatalf("first create: %v", err)
	}

	r2 := entity.NewRole(name, "Second")
	if err := repo.Create(context.Background(), r2); err == nil {
		t.Error("expected error for duplicate role name")
	}
}
