package app

import (
	"path/filepath"
	"testing"

	appdb "github.com/michaelwinser/appbase/db"
)

func testDB(t *testing.T) *appdb.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	d, err := appdb.New(appdb.DBConfig{StoreType: "sqlite", SQLitePath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestMigrateSchema_AddsMissingColumns(t *testing.T) {
	d := testDB(t)

	// Create a trips table missing key, start_date, end_date, status
	_, err := d.Exec(`CREATE TABLE trips (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		color TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatal(err)
	}

	// Create an activities table missing several columns
	_, err = d.Exec(`CREATE TABLE activities (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		title TEXT NOT NULL,
		type TEXT NOT NULL,
		start_date TEXT NOT NULL,
		end_date TEXT NOT NULL,
		location TEXT NOT NULL,
		notes TEXT NOT NULL,
		trip_id TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatal(err)
	}

	// Run migration
	if err := MigrateSchema(d); err != nil {
		t.Fatalf("MigrateSchema failed: %v", err)
	}

	// Verify trips now has the missing columns
	for _, col := range []string{"key", "start_date", "end_date", "status"} {
		if !columnExists(t, d, "trips", col) {
			t.Errorf("trips: expected column %q after migration", col)
		}
	}

	// Verify activities has the missing columns
	for _, col := range []string{"place_id", "origin_place_id", "destination_place_id", "key", "source"} {
		if !columnExists(t, d, "activities", col) {
			t.Errorf("activities: expected column %q after migration", col)
		}
	}

	// Verify running migration again is safe (idempotent)
	if err := MigrateSchema(d); err != nil {
		t.Fatalf("second MigrateSchema failed: %v", err)
	}
}

func TestMigrateSchema_NoTableNoError(t *testing.T) {
	d := testDB(t)
	if err := MigrateSchema(d); err != nil {
		t.Fatalf("MigrateSchema on empty DB failed: %v", err)
	}
}

func columnExists(t *testing.T, d *appdb.DB, table, column string) bool {
	t.Helper()
	rows, err := d.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt *string
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		if name == column {
			return true
		}
	}
	return false
}
