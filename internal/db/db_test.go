package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if db.Path() != dbPath {
		t.Errorf("Expected path %s, got %s", dbPath, db.Path())
	}

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestNew_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "nested", "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database with nested path: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(filepath.Dir(dbPath)); os.IsNotExist(err) {
		t.Error("Nested directories were not created")
	}
}

func TestSchema_TablesExist(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	tables := []string{
		"api_calls",
		"account_status",
		"session_events",
		"quota_snapshots",
		"quota_snapshots_agg",
	}

	for _, table := range tables {
		var name string
		err := db.QueryRowContext(context.Background(), "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s does not exist: %v", table, err)
		}
	}
}

func TestVacuum(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	if err := db.Vacuum(); err != nil {
		t.Errorf("Vacuum failed: %v", err)
	}
}

func TestClose(t *testing.T) {
	db := newTestDB(t)

	if err := db.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify database is closed by trying to query
	_, err := db.QueryContext(context.Background(), "SELECT 1")
	if err == nil {
		t.Error("Expected error querying closed database")
	}
}

// Helper to create a test database
func newTestDB(t *testing.T) *DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}
