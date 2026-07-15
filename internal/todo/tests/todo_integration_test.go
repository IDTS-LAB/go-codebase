package tests

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"github.com/IDTS-LAB/go-codebase/internal/shared/tenantfilter"
	"github.com/IDTS-LAB/go-codebase/internal/testhelper"
	"github.com/IDTS-LAB/go-codebase/internal/todo/domain/entity"
	todoPersistence "github.com/IDTS-LAB/go-codebase/internal/todo/infrastructure/persistence"
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

func TestTodoRepository_CreateAndGetByID(t *testing.T) {
	repo := todoPersistence.NewTodoRepository(db, &tenantfilter.Config{})

	todo := entity.NewTodo("Buy milk", "2% milk")
	err := repo.Create(context.Background(), todo)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(context.Background(), todo.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != todo.Title {
		t.Errorf("expected title %s, got %s", todo.Title, got.Title)
	}
	if got.Description != todo.Description {
		t.Errorf("expected description %s, got %s", todo.Description, got.Description)
	}
}

func TestTodoRepository_Update(t *testing.T) {
	repo := todoPersistence.NewTodoRepository(db, &tenantfilter.Config{})

	todo := entity.NewTodo("Old title", "Old desc")
	repo.Create(context.Background(), todo)

	todo.Update("New title", "New desc")
	err := repo.Update(context.Background(), todo)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(context.Background(), todo.ID)
	if got.Title != "New title" {
		t.Errorf("expected title 'New title', got %s", got.Title)
	}
}

func TestTodoRepository_Delete(t *testing.T) {
	repo := todoPersistence.NewTodoRepository(db, &tenantfilter.Config{})

	todo := entity.NewTodo("Delete me", "To be deleted")
	repo.Create(context.Background(), todo)

	err := repo.Delete(context.Background(), todo.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = repo.GetByID(context.Background(), todo.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestTodoRepository_GetAll(t *testing.T) {
	repo := todoPersistence.NewTodoRepository(db, &tenantfilter.Config{})

	repo.Create(context.Background(), entity.NewTodo("First", "First desc"))
	repo.Create(context.Background(), entity.NewTodo("Second", "Second desc"))

	todos, _, _, _, _, err := repo.GetAll(context.Background(), nil, 20)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(todos) < 2 {
		t.Errorf("expected at least 2 todos, got %d", len(todos))
	}
}

func TestTodoRepository_Search(t *testing.T) {
	repo := todoPersistence.NewTodoRepository(db, &tenantfilter.Config{})

	uniqueTitle := "SearchTest_" + uuid.New().String()
	repo.Create(context.Background(), entity.NewTodo(uniqueTitle, "Description"))

	results, _, _, _, _, err := repo.Search(context.Background(), uniqueTitle, nil, 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestTodoRepository_GetByID_NotFound(t *testing.T) {
	repo := todoPersistence.NewTodoRepository(db, &tenantfilter.Config{})
	_, err := repo.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error for nonexistent todo")
	}
}

func TestTodoRepository_Complete(t *testing.T) {
	repo := todoPersistence.NewTodoRepository(db, &tenantfilter.Config{})

	todo := entity.NewTodo("Complete me", "")
	repo.Create(context.Background(), todo)

	todo.Complete()
	err := repo.Update(context.Background(), todo)
	if err != nil {
		t.Fatalf("Update after complete: %v", err)
	}

	got, _ := repo.GetByID(context.Background(), todo.ID)
	if !got.Completed {
		t.Error("expected todo to be completed")
	}
}
