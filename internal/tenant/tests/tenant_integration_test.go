package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/IDTS-LAB/go-codebase/internal/tenant/domain/entity"
	tenantPersistence "github.com/IDTS-LAB/go-codebase/internal/tenant/infrastructure/persistence"
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

func newTenant(name, slug string) *entity.Tenant {
	now := time.Now().UTC()
	return &entity.Tenant{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		Settings:  json.RawMessage("{}"),
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestTenantRepository_CreateAndGetByID(t *testing.T) {
	repo := tenantPersistence.NewTenantRepository(db)

	slug := "test-corp-" + uuid.New().String()[:8]
	tenant := newTenant("Test Corp", slug)
	err := repo.Create(context.Background(), tenant)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != tenant.Name {
		t.Errorf("expected name %s, got %s", tenant.Name, got.Name)
	}
	if got.Slug != tenant.Slug {
		t.Errorf("expected slug %s, got %s", tenant.Slug, got.Slug)
	}
}

func TestTenantRepository_GetBySlug(t *testing.T) {
	repo := tenantPersistence.NewTenantRepository(db)

	slug := "unique-slug-test-" + uuid.New().String()[:8]
	tenant := newTenant("Slug Corp", slug)
	repo.Create(context.Background(), tenant)

	got, err := repo.GetBySlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.ID != tenant.ID {
		t.Errorf("expected id %s, got %s", tenant.ID, got.ID)
	}
}

func TestTenantRepository_List(t *testing.T) {
	repo := tenantPersistence.NewTenantRepository(db)

	repo.Create(context.Background(), newTenant("List Corp A", "list-a-"+uuid.New().String()[:8]))
	repo.Create(context.Background(), newTenant("List Corp B", "list-b-"+uuid.New().String()[:8]))

	tenants, _, _, _, _, err := repo.List(context.Background(), nil, 20)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tenants) < 2 {
		t.Errorf("expected at least 2 tenants, got %d", len(tenants))
	}
}

func TestTenantRepository_Update(t *testing.T) {
	repo := tenantPersistence.NewTenantRepository(db)

	tenant := newTenant("Before Corp", "before-slug-"+uuid.New().String()[:8])
	repo.Create(context.Background(), tenant)

	tenant.Name = "After Corp"
	tenant.UpdatedAt = time.Now().UTC()
	err := repo.Update(context.Background(), tenant)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(context.Background(), tenant.ID)
	if got.Name != "After Corp" {
		t.Errorf("expected name 'After Corp', got %s", got.Name)
	}
}

func TestTenantRepository_Delete(t *testing.T) {
	repo := tenantPersistence.NewTenantRepository(db)

	tenant := newTenant("Delete Corp", "delete-corp-"+uuid.New().String()[:8])
	repo.Create(context.Background(), tenant)

	err := repo.Delete(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = repo.GetByID(context.Background(), tenant.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestTenantRepository_GetByID_NotFound(t *testing.T) {
	repo := tenantPersistence.NewTenantRepository(db)
	_, err := repo.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error for nonexistent tenant")
	}
}

func TestTenantRepository_Create_DuplicateSlug(t *testing.T) {
	repo := tenantPersistence.NewTenantRepository(db)
	slug := "dup-slug-test-" + uuid.New().String()[:8]

	t1 := newTenant("First", slug)
	if err := repo.Create(context.Background(), t1); err != nil {
		t.Fatalf("first create: %v", err)
	}

	t2 := newTenant("Second", slug)
	if err := repo.Create(context.Background(), t2); err == nil {
		t.Error("expected error for duplicate slug")
	}
}
