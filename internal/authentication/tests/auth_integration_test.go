package tests

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/IDTS-LAB/go-codebase/internal/authentication/domain/entity"
	authRepo "github.com/IDTS-LAB/go-codebase/internal/authentication/infrastructure/persistence"
	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
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

func newTestUser(email, name string) *entity.User {
	now := domain.NewEntity()
	return &entity.User{
		Entity:   now,
		Email:    email,
		Password: "$2a$10$test_hash_value",
		Name:     name,
		IsActive: true,
	}
}

func TestAuthUserRepository_CreateAndGet(t *testing.T) {
	repo := authRepo.NewUserRepository(db)
	user := newTestUser("create-get-"+uuid.New().String()[:8]+"@example.com", "Test User")

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

func TestAuthUserRepository_GetByEmail(t *testing.T) {
	repo := authRepo.NewUserRepository(db)
	email := "find-by-email-" + uuid.New().String()[:8] + "@example.com"
	user := newTestUser(email, "Find Me")
	repo.Create(context.Background(), user)

	got, err := repo.GetByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("expected id %s, got %s", user.ID, got.ID)
	}
}

func TestAuthUserRepository_GetByEmail_NotFound(t *testing.T) {
	repo := authRepo.NewUserRepository(db)
	_, err := repo.GetByEmail(context.Background(), "nonexistent@example.com")
	if err == nil {
		t.Fatal("expected error for nonexistent email")
	}
}

func TestAuthUserRepository_Update(t *testing.T) {
	repo := authRepo.NewUserRepository(db)
	user := newTestUser("update-test-"+uuid.New().String()[:8]+"@example.com", "Before")
	repo.Create(context.Background(), user)

	user.Name = "After"
	user.Email = "updated-email-" + uuid.New().String()[:8] + "@example.com"
	user.Touch()
	err := repo.Update(context.Background(), user)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(context.Background(), user.ID)
	if got.Name != "After" {
		t.Errorf("expected name 'After', got %s", got.Name)
	}
}

func TestAuthRefreshTokenRepository_CreateAndGet(t *testing.T) {
	userRepo := authRepo.NewUserRepository(db)
	tokenRepo := authRepo.NewRefreshTokenRepository(db)

	user := newTestUser("token-user-"+uuid.New().String()[:8]+"@example.com", "Token User")
	userRepo.Create(context.Background(), user)

	token := entity.NewRefreshToken(user.ID, "token-hash-value-"+uuid.New().String(), time.Now().Add(30*24*time.Hour))

	err := tokenRepo.Create(context.Background(), token)
	if err != nil {
		t.Fatalf("Create token: %v", err)
	}

	got, err := tokenRepo.GetByToken(context.Background(), token.Token)
	if err != nil {
		t.Fatalf("GetByToken: %v", err)
	}
	if got.UserID != user.ID {
		t.Errorf("expected userID %s, got %s", user.ID, got.UserID)
	}
}

func TestAuthRefreshTokenRepository_Revoke(t *testing.T) {
	userRepo := authRepo.NewUserRepository(db)
	tokenRepo := authRepo.NewRefreshTokenRepository(db)

	user := newTestUser("revoke-test-"+uuid.New().String()[:8]+"@example.com", "Revoke User")
	userRepo.Create(context.Background(), user)

	token := entity.NewRefreshToken(user.ID, "revoke-token-hash-"+uuid.New().String(), time.Now().Add(30*24*time.Hour))
	tokenRepo.Create(context.Background(), token)

	err := tokenRepo.Revoke(context.Background(), token.Token)
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	got, _ := tokenRepo.GetByToken(context.Background(), token.Token)
	if !got.IsRevoked() {
		t.Error("expected token to be revoked")
	}
}

func TestAuthRefreshTokenRepository_RevokeAllByUserID(t *testing.T) {
	userRepo := authRepo.NewUserRepository(db)
	tokenRepo := authRepo.NewRefreshTokenRepository(db)

	user := newTestUser("revoke-all-test-"+uuid.New().String()[:8]+"@example.com", "Revoke All")
	userRepo.Create(context.Background(), user)

	token1 := entity.NewRefreshToken(user.ID, "token1-hash-"+uuid.New().String(), time.Now().Add(30*24*time.Hour))
	token2 := entity.NewRefreshToken(user.ID, "token2-hash-"+uuid.New().String(), time.Now().Add(30*24*time.Hour))
	tokenRepo.Create(context.Background(), token1)
	tokenRepo.Create(context.Background(), token2)

	err := tokenRepo.RevokeAllByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("RevokeAllByUserID: %v", err)
	}

	tokens, _ := tokenRepo.GetByUserID(context.Background(), user.ID)
	for _, tkn := range tokens {
		if !tkn.IsRevoked() {
			t.Error("expected all tokens to be revoked")
		}
	}
}

func TestAuthUserRepository_Create_DuplicateEmail(t *testing.T) {
	repo := authRepo.NewUserRepository(db)
	email := "dup-email-auth-" + uuid.New().String()[:8] + "@example.com"

	user1 := newTestUser(email, "First")
	if err := repo.Create(context.Background(), user1); err != nil {
		t.Fatalf("first create: %v", err)
	}

	user2 := newTestUser(email, "Second")
	if err := repo.Create(context.Background(), user2); err == nil {
		t.Error("expected error for duplicate email")
	}
}

func TestAuthUserRepository_GetByID_NotFound(t *testing.T) {
	repo := authRepo.NewUserRepository(db)
	_, err := repo.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}
