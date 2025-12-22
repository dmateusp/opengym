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
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("Failed to set goose dialect: %v", err)
	}

	if err := goose.UpContext(t.Context(), sqlDB, "../../db/migrations"); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return sqlDB
}

func UpsertTestUser(t *testing.T, sqlDB *sql.DB, email string) int64 {
	t.Helper()

	// Idempotent insert by email; returns the existing or newly inserted row id.
	row := sqlDB.QueryRow(`
		insert into users (email, name) values (?, ?)
		on conflict(email) do update set name = excluded.name
		returning id
	`, email, "Test User")

	var userID int64
	if err := row.Scan(&userID); err != nil {
		t.Fatalf("Failed to upsert test user %s: %v", email, err)
	}

	return userID
}
