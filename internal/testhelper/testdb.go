package testhelper

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pressly/goose/v3"
)

func migrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "..", "..", "migrations")
}

func SetupTestDB(m *testing.M) (*sql.DB, func()) {
	db, err := sql.Open("postgres", dsnFromEnv())
	if err != nil {
		panic(fmt.Sprintf("connect to test DB: %v", err))
	}
	if err := db.Ping(); err != nil {
		panic(fmt.Sprintf("ping test DB: %v", err))
	}
	if err := runMigrations(db); err != nil {
		panic(fmt.Sprintf("run migrations: %v", err))
	}
	return db, func() { db.Close() }
}

func dsnFromEnv() string {
	host := envOrDefault("DB_HOST", "localhost")
	port := envOrDefault("DB_PORT", "5432")
	user := envOrDefault("DB_USER", "postgres")
	password := envOrDefault("DB_PASSWORD", "postgres")
	dbname := envOrDefault("DB_NAME", "codebase_testing")
	sslmode := envOrDefault("DB_SSLMODE", "disable")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func runMigrations(db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	mDir := migrationsDir()
	if _, err := os.Stat(mDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found at %s", mDir)
	}
	if err := goose.Up(db, mDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

func WithTx(t *testing.T, db *sql.DB, fn func(tx *sql.Tx)) {
	t.Helper()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	fn(tx)
}
