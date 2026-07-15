package tests

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"

	authEntity "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/IDTS-LAB/go-codebase/internal/testhelper"
	userPersistence "github.com/IDTS-LAB/go-codebase/internal/user/infrastructure/persistence"
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

func newUser(email, name string) *authEntity.User {
	return &authEntity.User{
		Entity:   domain.NewEntity(),
		Email:    email,
		Password: "$2a$10$test_hash",
		Name:     name,
		IsActive: true,
	}
}

func TestUserRepository_CreateAndGetByID(t *testing.T) {
	repo := userPersistence.NewUserRepository(db, &tenantfilter.Config{})

	email := "create-get-user-" + uuid.New().String()[:8] + "@example.com"
	user := newUser(email, "Create Test")
	err := repo.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, got.Email)
	}
	if got.Name != user.Name {
		t.Errorf("expected name %s, got %s", user.Name, got.Name)
	}
}

func TestUserRepository_List(t *testing.T) {
	repo := userPersistence.NewUserRepository(db, &tenantfilter.Config{})

	repo.Create(context.Background(), newUser("list1-user-"+uuid.New().String()[:8]+"@example.com", "List One"))
	repo.Create(context.Background(), newUser("list2-user-"+uuid.New().String()[:8]+"@example.com", "List Two"))

	users, _, _, _, _, err := repo.List(context.Background(), nil, 20)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(users) < 2 {
		t.Errorf("expected at least 2 users, got %d", len(users))
	}
}

func TestUserRepository_Update(t *testing.T) {
	repo := userPersistence.NewUserRepository(db, &tenantfilter.Config{})

	email := "update-user-test-" + uuid.New().String()[:8] + "@example.com"
	user := newUser(email, "Before")
	repo.Create(context.Background(), user)

	user.Name = "After"
	user.Email = "after-update-" + uuid.New().String()[:8] + "@example.com"
	err := repo.Update(context.Background(), user)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(context.Background(), user.ID)
	if got.Name != "After" {
		t.Errorf("expected name 'After', got %s", got.Name)
	}
}

func TestUserRepository_Delete(t *testing.T) {
	repo := userPersistence.NewUserRepository(db, &tenantfilter.Config{})

	email := "delete-user-test-" + uuid.New().String()[:8] + "@example.com"
	user := newUser(email, "Delete Test")
	repo.Create(context.Background(), user)

	err := repo.Delete(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = repo.GetByID(context.Background(), user.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	repo := userPersistence.NewUserRepository(db, &tenantfilter.Config{})
	_, err := repo.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}
