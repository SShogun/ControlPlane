package data

import (
	"context"
	"testing"
)

func TestCreateDraft(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	// Use the global testPool and teardown helper
	teardownTestDB(t, testPool)

	store := &PgxStore{DB: testPool}
	ctx := context.Background()
	params := CreateDraftParams{
		Title: "Test Notebook",
		Body:  "This is a test content",
	}
	id, err := store.CreateDraft(ctx, params)
	if err != nil {
		t.Fatalf("failed to create draft: %v", err)
	}
	// validation
	if id == 0 {
		t.Errorf("expected a non-zero ID, got %d", id)
	}
	// verify if in db
	var title string
	err = testPool.QueryRow(ctx, "SELECT title FROM notebooks WHERE id = $1", id).Scan(&title)
	if err != nil {
		t.Fatalf("failed to fetch created notebook: %v", err)
	}
	if title != params.Title {
		t.Errorf("expected title %q, got %q", params.Title, title)
	}
}

func TestInsertAuditLog(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	// Use the global testPool and teardown helper
	teardownTestDB(t, testPool)

	store := &PgxStore{DB: testPool}
	ctx := context.Background()
	// user to reference in the audit log
	var userID int
	err := testPool.QueryRow(ctx, "INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id", "test@example.com", []byte("hash")).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	params := InsertAuditLogParams{
		UserID:     userID,
		Action:     "create",
		EntityType: "notebook",
		EntityID:   1,
	}
	err = store.InsertAuditLog(ctx, params)
	if err != nil {
		t.Errorf("InsertAuditLog failed: %v", err)
	}
}
