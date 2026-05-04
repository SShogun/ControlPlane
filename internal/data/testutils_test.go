package data

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	// 1. Setup: This runs ONCE before any tests start
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://testuser:testpass@localhost:5433/controlplane_test?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to test database: %v\n", err)
		os.Exit(1)
	}
	testPool = pool

	// Run migrations once
	setupMigrations(testPool)

	// 2. Execution: Run all tests in the package
	code := m.Run()

	// 3. Teardown: This runs ONCE after all tests finish
	testPool.Close()
	os.Exit(code)
}

func setupMigrations(pool *pgxpool.Pool) {
	migrationsDir := "../../migrations"
	files, _ := os.ReadDir(migrationsDir)

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" {
			path := filepath.Join(migrationsDir, file.Name())
			script, _ := os.ReadFile(path)
			_, err := pool.Exec(context.Background(), string(script))
			if err != nil {
				// We ignore "already exists" errors if we re-run migrations,
				// but since TestMain runs once, this is just a safeguard.
				fmt.Printf("Migration info: %s\n", err)
			}
		}
	}
}

// teardownTestDB now just clears the data, so tests stay isolated
func teardownTestDB(t *testing.T, pool *pgxpool.Pool) {
	tables := []string{"audit_logs", "audit_events", "notebook_revisions", "notebooks", "users"}
	for _, table := range tables {
		_, err := pool.Exec(context.Background(), "TRUNCATE TABLE "+table+" RESTART IDENTITY CASCADE")
		if err != nil {
			t.Errorf("failed to truncate table %s: %v", table, err)
		}
	}
}
