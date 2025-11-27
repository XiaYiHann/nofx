package testhelpers

import (
	"nofx/config"
	"os"
	"testing"
)

// SetupTestDB creates a temporary database for testing and returns the database instance and a cleanup function.
// It also creates a default user "test-user".
func SetupTestDB(t *testing.T) (*config.Database, func()) {
	t.Helper()

	// Create a temporary file for the database
	tmpFile, err := os.CreateTemp("", "nofx-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close() // Close the file handle so sqlite can open it

	db, err := config.NewDatabase(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("Failed to create database: %v", err)
	}

	// Create a default test user
	err = db.CreateUser(&config.User{
		ID:           "test-user",
		Email:        "test@example.com",
		PasswordHash: "hash",
	})
	if err != nil {
		db.Close()
		os.Remove(dbPath)
		t.Fatalf("Failed to create test user: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}
