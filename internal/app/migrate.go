package app

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"

	appdb "github.com/michaelwinser/appbase/db"
)

// tableDef describes a table and its expected Go struct type for migration.
type tableDef struct {
	name       string
	structType reflect.Type
}

// allTables returns the full list of tables and their struct types.
func allTables() []tableDef {
	return []tableDef{
		{"trips", reflect.TypeFor[Trip]()},
		{"activities", reflect.TypeFor[Activity]()},
		{"share_links", reflect.TypeFor[ShareLink]()},
		{"shares", reflect.TypeFor[Share]()},
		{"places", reflect.TypeFor[Place]()},
		{"public_profiles", reflect.TypeFor[PublicProfile]()},
		{"parse_history", reflect.TypeFor[ParseHistory]()},
		{"user_configs", reflect.TypeFor[UserConfig]()},
		{"sync_targets", reflect.TypeFor[SyncTarget]()},
		{"sync_records", reflect.TypeFor[SyncRecord]()},
		{"import_sources", reflect.TypeFor[ImportSource]()},
		{"staged_events", reflect.TypeFor[StagedEvent]()},
	}
}

// MigrateSchema ensures all tables have the columns declared by their struct tags.
// For any missing column, it runs ALTER TABLE ADD COLUMN with a type-appropriate default.
// This handles the common case of adding new fields to existing entities.
func MigrateSchema(d *appdb.DB) error {
	if !d.IsSQL() {
		return nil // Firestore is schemaless
	}
	rawDB := d.SQL()

	for _, td := range allTables() {
		if err := migrateTable(rawDB, td.name, td.structType); err != nil {
			return fmt.Errorf("migrate %s: %w", td.name, err)
		}
	}
	return nil
}

func migrateTable(db *sql.DB, table string, structType reflect.Type) error {
	// Check if table exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return nil // Table doesn't exist yet — NewCollection will create it
	}

	// Get existing columns
	existing := map[string]bool{}
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt *string
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return err
		}
		existing[name] = true
	}

	// Get expected columns from struct tags
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("store")
		if tag == "" || tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		colName := parts[0]

		if existing[colName] {
			continue
		}

		// Determine SQL type
		sqlType := goTypeToSQL(field.Type)
		defaultVal := defaultForType(sqlType)

		stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s NOT NULL DEFAULT %s",
			table, colName, sqlType, defaultVal)
		log.Printf("migrate: %s — adding column %s (%s)", table, colName, sqlType)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("ALTER TABLE %s ADD COLUMN %s: %w", table, colName, err)
		}
	}
	return nil
}

func goTypeToSQL(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Bool:
		return "INTEGER"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "INTEGER"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	default:
		return "TEXT"
	}
}

func defaultForType(sqlType string) string {
	switch sqlType {
	case "INTEGER":
		return "0"
	case "REAL":
		return "0.0"
	default:
		return "''"
	}
}
