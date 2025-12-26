package dbtesting

import (
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"
)

func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Run goose migrations
	goose.SetBaseFS(nil) // Use filesystem
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("Failed to set goose dialect: %v", err)
	}

	if err := goose.Up(sqlDB, "../../db/migrations"); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return sqlDB
}

func CreateTestUser(t *testing.T, sqlDB *sql.DB) int64 {
	t.Helper()

	result, err := sqlDB.Exec(`
		insert into users (email, name) VALUES (?, ?)
	`, "john@example.com", "John")
	if err != nil {
		t.Fatalf("Failed to create test user John: %v", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get test user ID: %v", err)
	}

	return userID
}
