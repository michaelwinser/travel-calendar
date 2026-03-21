// Package store provides database access layer implementations.
package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/user/travel-calendar/backend/internal/entity"
)

// Compile-time check that SQLiteStore implements StoreInterface.
var _ StoreInterface = (*SQLiteStore)(nil)

// SQLiteStore provides database access methods using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLite creates a new SQLiteStore with the given database path.
func NewSQLite(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// migrate creates the database schema.
func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS trips (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL DEFAULT '',
		name TEXT NOT NULL,
		purpose TEXT NOT NULL,
		location TEXT,
		start_date TEXT,
		end_date TEXT,
		status TEXT NOT NULL DEFAULT 'planning',
		notes TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS items (
		id TEXT PRIMARY KEY,
		trip_id TEXT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
		type TEXT NOT NULL,
		date TEXT,
		time TEXT,
		confirmation TEXT,
		notes TEXT,
		from_location TEXT,
		to_location TEXT,
		carrier TEXT,
		flight_number TEXT,
		name TEXT,
		location TEXT,
		check_in TEXT,
		check_out TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS documents (
		id TEXT PRIMARY KEY,
		trip_id TEXT REFERENCES trips(id) ON DELETE SET NULL,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		url TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_items_trip_id ON items(trip_id);
	CREATE INDEX IF NOT EXISTS idx_documents_trip_id ON documents(trip_id);
	CREATE INDEX IF NOT EXISTS idx_trips_start_date ON trips(start_date);
	CREATE INDEX IF NOT EXISTS idx_trips_purpose ON trips(purpose);
	CREATE INDEX IF NOT EXISTS idx_trips_user_id ON trips(user_id);

	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS trip_locations (
		id TEXT PRIMARY KEY,
		trip_id TEXT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
		date TEXT NOT NULL,
		location TEXT NOT NULL,
		created_at TEXT NOT NULL,
		UNIQUE(trip_id, date, location)
	);

	CREATE INDEX IF NOT EXISTS idx_trip_locations_trip_id ON trip_locations(trip_id);
	CREATE INDEX IF NOT EXISTS idx_trip_locations_date ON trip_locations(date);

	-- Google Calendar OAuth credentials (multi-user ready)
	CREATE TABLE IF NOT EXISTS google_credentials (
		user_id TEXT PRIMARY KEY DEFAULT 'default',
		access_token TEXT NOT NULL,
		refresh_token TEXT NOT NULL,
		token_type TEXT NOT NULL DEFAULT 'Bearer',
		expires_at TEXT NOT NULL,
		scopes TEXT NOT NULL,
		email TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	-- User's selected calendars for monitoring
	CREATE TABLE IF NOT EXISTS user_calendars (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL DEFAULT 'default',
		calendar_id TEXT NOT NULL,
		name TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		UNIQUE(user_id, calendar_id)
	);

	CREATE INDEX IF NOT EXISTS idx_user_calendars_user ON user_calendars(user_id);

	-- Trip-to-calendar event sync tracking
	CREATE TABLE IF NOT EXISTS calendar_links (
		id TEXT PRIMARY KEY,
		trip_id TEXT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
		item_id TEXT REFERENCES items(id) ON DELETE CASCADE,
		calendar_id TEXT NOT NULL,
		event_id TEXT NOT NULL,
		synced_at TEXT NOT NULL,
		UNIQUE(trip_id, calendar_id, event_id)
	);

	CREATE INDEX IF NOT EXISTS idx_calendar_links_trip ON calendar_links(trip_id);
	CREATE INDEX IF NOT EXISTS idx_calendar_links_event ON calendar_links(calendar_id, event_id);

	-- Processed calendar events tracking (for suggestion deduplication)
	CREATE TABLE IF NOT EXISTS processed_calendar_events (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL DEFAULT '',
		calendar_event_id TEXT NOT NULL,
		calendar_id TEXT NOT NULL,
		action TEXT NOT NULL,
		trip_id TEXT,
		item_id TEXT,
		processed_at TEXT NOT NULL,
		UNIQUE(calendar_id, calendar_event_id)
	);

	CREATE INDEX IF NOT EXISTS idx_processed_events_calendar ON processed_calendar_events(calendar_id);
	CREATE INDEX IF NOT EXISTS idx_processed_events_user ON processed_calendar_events(user_id);

	-- User sessions for authentication
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		email TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		created_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Trip methods

// ListTrips returns all trips, optionally filtered.
func (s *SQLiteStore) ListTrips(userID string, upcoming, past *bool, purpose *string) ([]entity.Trip, error) {
	query := "SELECT id, user_id, name, purpose, location, start_date, end_date, status, notes, created_at, updated_at FROM trips WHERE user_id = ?"
	args := []interface{}{userID}

	now := time.Now().Format("2006-01-02")
	if upcoming != nil && *upcoming {
		query += " AND (start_date IS NULL OR start_date >= ?)"
		args = append(args, now)
	}
	if past != nil && *past {
		query += " AND end_date IS NOT NULL AND end_date < ?"
		args = append(args, now)
	}
	if purpose != nil {
		query += " AND purpose = ?"
		args = append(args, *purpose)
	}

	query += " ORDER BY COALESCE(start_date, '9999-12-31') ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []entity.Trip
	for rows.Next() {
		trip, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		trips = append(trips, trip)
	}
	return trips, rows.Err()
}

// GetTrip returns a single trip by ID.
func (s *SQLiteStore) GetTrip(userID string, id uuid.UUID) (*entity.Trip, error) {
	query := "SELECT id, user_id, name, purpose, location, start_date, end_date, status, notes, created_at, updated_at FROM trips WHERE user_id = ? AND id = ?"
	row := s.db.QueryRow(query, userID, id.String())
	trip, err := scanTripRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &trip, nil
}

// CreateTrip inserts a new trip.
func (s *SQLiteStore) CreateTrip(trip *entity.Trip) error {
	query := `INSERT INTO trips (id, user_id, name, purpose, location, start_date, end_date, status, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query,
		trip.ID.String(),
		trip.UserID,
		trip.Name,
		trip.Purpose,
		trip.Location,
		formatDatePtr(trip.StartDate),
		formatDatePtr(trip.EndDate),
		trip.Status,
		trip.Notes,
		trip.CreatedAt.Format(time.RFC3339),
		trip.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// UpdateTrip updates an existing trip.
func (s *SQLiteStore) UpdateTrip(userID string, trip *entity.Trip) error {
	query := `UPDATE trips SET name = ?, purpose = ?, location = ?, start_date = ?, end_date = ?, status = ?, notes = ?, updated_at = ? WHERE user_id = ? AND id = ?`
	result, err := s.db.Exec(query,
		trip.Name,
		trip.Purpose,
		trip.Location,
		formatDatePtr(trip.StartDate),
		formatDatePtr(trip.EndDate),
		trip.Status,
		trip.Notes,
		trip.UpdatedAt.Format(time.RFC3339),
		userID,
		trip.ID.String(),
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteTrip deletes a trip by ID.
func (s *SQLiteStore) DeleteTrip(userID string, id uuid.UUID) error {
	result, err := s.db.Exec("DELETE FROM trips WHERE user_id = ? AND id = ?", userID, id.String())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// SearchTrips searches trips by query string.
func (s *SQLiteStore) SearchTrips(userID string, q string) ([]entity.Trip, error) {
	pattern := "%" + q + "%"
	query := `SELECT id, user_id, name, purpose, location, start_date, end_date, status, notes, created_at, updated_at FROM trips
		WHERE user_id = ? AND (name LIKE ? OR notes LIKE ?)
		ORDER BY COALESCE(start_date, '9999-12-31') ASC`

	rows, err := s.db.Query(query, userID, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []entity.Trip
	for rows.Next() {
		trip, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		trips = append(trips, trip)
	}
	return trips, rows.Err()
}

// Item methods

// ListItems returns all items for a trip.
func (s *SQLiteStore) ListItems(tripID uuid.UUID) ([]entity.Item, error) {
	query := `SELECT id, trip_id, type, date, time, confirmation, notes, from_location, to_location, carrier, flight_number, name, location, check_in, check_out, created_at, updated_at
		FROM items WHERE trip_id = ? ORDER BY COALESCE(date, '9999-12-31') ASC, time ASC`

	rows, err := s.db.Query(query, tripID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []entity.Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetItem returns a single item by ID.
func (s *SQLiteStore) GetItem(id uuid.UUID) (*entity.Item, error) {
	query := `SELECT id, trip_id, type, date, time, confirmation, notes, from_location, to_location, carrier, flight_number, name, location, check_in, check_out, created_at, updated_at
		FROM items WHERE id = ?`
	row := s.db.QueryRow(query, id.String())
	item, err := scanItemRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateItem inserts a new item.
func (s *SQLiteStore) CreateItem(item *entity.Item) error {
	query := `INSERT INTO items (id, trip_id, type, date, time, confirmation, notes, from_location, to_location, carrier, flight_number, name, location, check_in, check_out, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query,
		item.ID.String(),
		item.TripID.String(),
		item.Type,
		formatDatePtr(item.Date),
		item.Time,
		item.Confirmation,
		item.Notes,
		item.From,
		item.To,
		item.Carrier,
		item.FlightNumber,
		item.Name,
		item.Location,
		formatDatePtr(item.CheckIn),
		formatDatePtr(item.CheckOut),
		item.CreatedAt.Format(time.RFC3339),
		item.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// DeleteItem deletes an item by ID.
func (s *SQLiteStore) DeleteItem(id uuid.UUID) error {
	result, err := s.db.Exec("DELETE FROM items WHERE id = ?", id.String())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateItemTrip updates an item's trip assignment.
func (s *SQLiteStore) UpdateItemTrip(itemID, newTripID uuid.UUID) error {
	query := `UPDATE items SET trip_id = ?, updated_at = ? WHERE id = ?`
	result, err := s.db.Exec(query, newTripID.String(), time.Now().Format(time.RFC3339), itemID.String())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// MergeTripsTransaction executes a complete trip merge within a transaction.
// Moves all items from source to target, merges locations, and deletes source.
func (s *SQLiteStore) MergeTripsTransaction(sourceID, targetID uuid.UUID) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)

	// Move all items from source to target
	_, err = tx.Exec(
		`UPDATE items SET trip_id = ?, updated_at = ? WHERE trip_id = ?`,
		targetID.String(),
		now,
		sourceID.String(),
	)
	if err != nil {
		return fmt.Errorf("moving items: %w", err)
	}

	// Copy locations from source to target for dates not already in target.
	// First get target dates to exclude.
	rows, err := tx.Query(
		`SELECT date FROM trip_locations WHERE trip_id = ?`,
		targetID.String(),
	)
	if err != nil {
		return fmt.Errorf("getting target dates: %w", err)
	}
	targetDates := make(map[string]bool)
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			rows.Close()
			return fmt.Errorf("scanning target date: %w", err)
		}
		targetDates[date] = true
	}
	rows.Close()

	// Get source locations and insert those for dates not in target.
	rows, err = tx.Query(
		`SELECT date, location FROM trip_locations WHERE trip_id = ?`,
		sourceID.String(),
	)
	if err != nil {
		return fmt.Errorf("getting source locations: %w", err)
	}

	insertStmt, err := tx.Prepare(
		`INSERT INTO trip_locations (id, trip_id, date, location, created_at) VALUES (?, ?, ?, ?, ?)`,
	)
	if err != nil {
		rows.Close()
		return fmt.Errorf("preparing insert: %w", err)
	}
	defer insertStmt.Close()

	for rows.Next() {
		var date, location string
		if err := rows.Scan(&date, &location); err != nil {
			rows.Close()
			return fmt.Errorf("scanning source location: %w", err)
		}
		// Only insert if target doesn't have this date
		if !targetDates[date] {
			newID := uuid.New()
			_, err := insertStmt.Exec(newID.String(), targetID.String(), date, location, now)
			if err != nil {
				rows.Close()
				return fmt.Errorf("inserting location: %w", err)
			}
			targetDates[date] = true // Avoid duplicates if source has multiple locations per date
		}
	}
	rows.Close()

	// Delete source trip (cascade will delete orphaned locations)
	_, err = tx.Exec("DELETE FROM trips WHERE id = ?", sourceID.String())
	if err != nil {
		return fmt.Errorf("deleting source trip: %w", err)
	}

	return tx.Commit()
}

// Document methods

// ListDocuments returns documents, optionally filtered by trip or unassociated.
func (s *SQLiteStore) ListDocuments(tripID *uuid.UUID, unassociated *bool) ([]entity.Document, error) {
	query := "SELECT id, trip_id, name, type, url, created_at, updated_at FROM documents WHERE 1=1"
	args := []interface{}{}

	if tripID != nil {
		query += " AND trip_id = ?"
		args = append(args, tripID.String())
	}
	if unassociated != nil && *unassociated {
		query += " AND trip_id IS NULL"
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []entity.Document
	for rows.Next() {
		doc, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

// Helper functions

func formatDatePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

func parseDatePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	return &t
}

func parseTimePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTrip(rows *sql.Rows) (entity.Trip, error) {
	var trip entity.Trip
	var id, location, startDate, endDate, notes, createdAt, updatedAt sql.NullString
	err := rows.Scan(&id, &trip.UserID, &trip.Name, &trip.Purpose, &location, &startDate, &endDate, &trip.Status, &notes, &createdAt, &updatedAt)
	if err != nil {
		return trip, err
	}
	trip.ID, _ = uuid.Parse(id.String)
	trip.Location = nullToPtr(location)
	trip.StartDate = parseDatePtr(nullToPtr(startDate))
	trip.EndDate = parseDatePtr(nullToPtr(endDate))
	trip.Notes = nullToPtr(notes)
	if createdAt.Valid {
		trip.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if updatedAt.Valid {
		trip.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	}
	return trip, nil
}

func scanTripRow(row *sql.Row) (entity.Trip, error) {
	var trip entity.Trip
	var id, location, startDate, endDate, notes, createdAt, updatedAt sql.NullString
	err := row.Scan(&id, &trip.UserID, &trip.Name, &trip.Purpose, &location, &startDate, &endDate, &trip.Status, &notes, &createdAt, &updatedAt)
	if err != nil {
		return trip, err
	}
	trip.ID, _ = uuid.Parse(id.String)
	trip.Location = nullToPtr(location)
	trip.StartDate = parseDatePtr(nullToPtr(startDate))
	trip.EndDate = parseDatePtr(nullToPtr(endDate))
	trip.Notes = nullToPtr(notes)
	if createdAt.Valid {
		trip.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if updatedAt.Valid {
		trip.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	}
	return trip, nil
}

func scanItem(rows *sql.Rows) (entity.Item, error) {
	var item entity.Item
	var id, tripID, date, itemTime, confirmation, notes, from, to, carrier, flightNumber, name, location, checkIn, checkOut, createdAt, updatedAt sql.NullString
	err := rows.Scan(&id, &tripID, &item.Type, &date, &itemTime, &confirmation, &notes, &from, &to, &carrier, &flightNumber, &name, &location, &checkIn, &checkOut, &createdAt, &updatedAt)
	if err != nil {
		return item, err
	}
	item.ID, _ = uuid.Parse(id.String)
	item.TripID, _ = uuid.Parse(tripID.String)
	item.Date = parseDatePtr(nullToPtr(date))
	item.Time = nullToPtr(itemTime)
	item.Confirmation = nullToPtr(confirmation)
	item.Notes = nullToPtr(notes)
	item.From = nullToPtr(from)
	item.To = nullToPtr(to)
	item.Carrier = nullToPtr(carrier)
	item.FlightNumber = nullToPtr(flightNumber)
	item.Name = nullToPtr(name)
	item.Location = nullToPtr(location)
	item.CheckIn = parseDatePtr(nullToPtr(checkIn))
	item.CheckOut = parseDatePtr(nullToPtr(checkOut))
	if createdAt.Valid {
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if updatedAt.Valid {
		item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	}
	return item, nil
}

func scanItemRow(row *sql.Row) (entity.Item, error) {
	var item entity.Item
	var id, tripID, date, itemTime, confirmation, notes, from, to, carrier, flightNumber, name, location, checkIn, checkOut, createdAt, updatedAt sql.NullString
	err := row.Scan(&id, &tripID, &item.Type, &date, &itemTime, &confirmation, &notes, &from, &to, &carrier, &flightNumber, &name, &location, &checkIn, &checkOut, &createdAt, &updatedAt)
	if err != nil {
		return item, err
	}
	item.ID, _ = uuid.Parse(id.String)
	item.TripID, _ = uuid.Parse(tripID.String)
	item.Date = parseDatePtr(nullToPtr(date))
	item.Time = nullToPtr(itemTime)
	item.Confirmation = nullToPtr(confirmation)
	item.Notes = nullToPtr(notes)
	item.From = nullToPtr(from)
	item.To = nullToPtr(to)
	item.Carrier = nullToPtr(carrier)
	item.FlightNumber = nullToPtr(flightNumber)
	item.Name = nullToPtr(name)
	item.Location = nullToPtr(location)
	item.CheckIn = parseDatePtr(nullToPtr(checkIn))
	item.CheckOut = parseDatePtr(nullToPtr(checkOut))
	if createdAt.Valid {
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if updatedAt.Valid {
		item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	}
	return item, nil
}

func scanDocument(rows *sql.Rows) (entity.Document, error) {
	var doc entity.Document
	var id, tripID, url, createdAt, updatedAt sql.NullString
	err := rows.Scan(&id, &tripID, &doc.Name, &doc.Type, &url, &createdAt, &updatedAt)
	if err != nil {
		return doc, err
	}
	doc.ID, _ = uuid.Parse(id.String)
	if tripID.Valid && tripID.String != "" {
		tid, _ := uuid.Parse(tripID.String)
		doc.TripID = &tid
	}
	doc.URL = nullToPtr(url)
	if createdAt.Valid {
		doc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if updatedAt.Valid {
		doc.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	}
	return doc, nil
}

func nullToPtr(ns sql.NullString) *string {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	return &ns.String
}

// Config methods

// GetConfig retrieves a config value by key.
func (s *SQLiteStore) GetConfig(key string) (*string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// SetConfig sets a config value.
func (s *SQLiteStore) SetConfig(key, value string) error {
	query := `INSERT INTO config (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`
	_, err := s.db.Exec(query, key, value, time.Now().Format(time.RFC3339))
	return err
}

// DeleteConfig removes a config value.
func (s *SQLiteStore) DeleteConfig(key string) error {
	_, err := s.db.Exec("DELETE FROM config WHERE key = ?", key)
	return err
}

// Trip Location methods

// GetTripLocations returns all locations for a trip.
func (s *SQLiteStore) GetTripLocations(tripID uuid.UUID) ([]entity.TripLocation, error) {
	query := `SELECT id, trip_id, date, location, created_at FROM trip_locations
		WHERE trip_id = ? ORDER BY date ASC, location ASC`
	rows, err := s.db.Query(query, tripID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []entity.TripLocation
	for rows.Next() {
		var loc entity.TripLocation
		var id, tripIDStr, date, createdAt string
		if err := rows.Scan(&id, &tripIDStr, &date, &loc.Location, &createdAt); err != nil {
			return nil, err
		}
		loc.ID, _ = uuid.Parse(id)
		loc.TripID, _ = uuid.Parse(tripIDStr)
		loc.Date, _ = time.Parse("2006-01-02", date)
		loc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		locations = append(locations, loc)
	}
	return locations, rows.Err()
}

// SetTripLocations replaces all locations for a trip.
func (s *SQLiteStore) SetTripLocations(tripID uuid.UUID, locations []entity.TripLocation) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing locations
	if _, err := tx.Exec("DELETE FROM trip_locations WHERE trip_id = ?", tripID.String()); err != nil {
		return err
	}

	// Insert new locations
	stmt, err := tx.Prepare(`INSERT INTO trip_locations (id, trip_id, date, location, created_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, loc := range locations {
		_, err := stmt.Exec(
			loc.ID.String(),
			loc.TripID.String(),
			loc.Date.Format("2006-01-02"),
			loc.Location,
			loc.CreatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetTripsForDateRange returns trips that overlap with the given date range.
func (s *SQLiteStore) GetTripsForDateRange(userID string, from, to time.Time) ([]entity.Trip, error) {
	query := `SELECT id, user_id, name, purpose, location, start_date, end_date, status, notes, created_at, updated_at FROM trips
		WHERE user_id = ? AND start_date IS NOT NULL AND end_date IS NOT NULL
		AND start_date <= ? AND end_date >= ?
		ORDER BY start_date ASC`

	rows, err := s.db.Query(query, userID, to.Format("2006-01-02"), from.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []entity.Trip
	for rows.Next() {
		trip, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		trips = append(trips, trip)
	}
	return trips, rows.Err()
}

// GetTripLocationsForDateRange returns locations for a trip within a date range.
func (s *SQLiteStore) GetTripLocationsForDateRange(tripID uuid.UUID, from, to time.Time) ([]entity.TripLocation, error) {
	query := `SELECT id, trip_id, date, location, created_at FROM trip_locations
		WHERE trip_id = ? AND date >= ? AND date <= ?
		ORDER BY date ASC, location ASC`

	rows, err := s.db.Query(query, tripID.String(), from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []entity.TripLocation
	for rows.Next() {
		var loc entity.TripLocation
		var id, tripIDStr, date, createdAt string
		if err := rows.Scan(&id, &tripIDStr, &date, &loc.Location, &createdAt); err != nil {
			return nil, err
		}
		loc.ID, _ = uuid.Parse(id)
		loc.TripID, _ = uuid.Parse(tripIDStr)
		loc.Date, _ = time.Parse("2006-01-02", date)
		loc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		locations = append(locations, loc)
	}
	return locations, rows.Err()
}

// Google Credentials methods

// GetGoogleCredentials retrieves OAuth credentials for a user.
func (s *SQLiteStore) GetGoogleCredentials(userID string) (*entity.GoogleCredentials, error) {
	query := `SELECT user_id, access_token, refresh_token, token_type, expires_at, scopes, email, created_at, updated_at
		FROM google_credentials WHERE user_id = ?`
	row := s.db.QueryRow(query, userID)

	var creds entity.GoogleCredentials
	var expiresAt, createdAt, updatedAt string
	var email sql.NullString

	err := row.Scan(
		&creds.UserID, &creds.AccessToken, &creds.RefreshToken, &creds.TokenType,
		&expiresAt, &creds.Scopes, &email, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	creds.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	creds.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	creds.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	creds.Email = nullToPtr(email)

	return &creds, nil
}

// SaveGoogleCredentials inserts or updates OAuth credentials.
func (s *SQLiteStore) SaveGoogleCredentials(creds *entity.GoogleCredentials) error {
	query := `INSERT INTO google_credentials (user_id, access_token, refresh_token, token_type, expires_at, scopes, email, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			token_type = excluded.token_type,
			expires_at = excluded.expires_at,
			scopes = excluded.scopes,
			email = excluded.email,
			updated_at = excluded.updated_at`

	_, err := s.db.Exec(query,
		creds.UserID,
		creds.AccessToken,
		creds.RefreshToken,
		creds.TokenType,
		creds.ExpiresAt.Format(time.RFC3339),
		creds.Scopes,
		creds.Email,
		creds.CreatedAt.Format(time.RFC3339),
		creds.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// DeleteGoogleCredentials removes OAuth credentials for a user.
func (s *SQLiteStore) DeleteGoogleCredentials(userID string) error {
	result, err := s.db.Exec("DELETE FROM google_credentials WHERE user_id = ?", userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// User Calendar methods

// ListUserCalendars returns all selected calendars for a user.
func (s *SQLiteStore) ListUserCalendars(userID string) ([]entity.UserCalendar, error) {
	query := `SELECT id, user_id, calendar_id, name, enabled, created_at, updated_at
		FROM user_calendars WHERE user_id = ? ORDER BY name ASC`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calendars []entity.UserCalendar
	for rows.Next() {
		cal, err := scanUserCalendar(rows)
		if err != nil {
			return nil, err
		}
		calendars = append(calendars, cal)
	}
	return calendars, rows.Err()
}

// GetUserCalendar retrieves a specific user calendar by ID.
func (s *SQLiteStore) GetUserCalendar(id uuid.UUID) (*entity.UserCalendar, error) {
	query := `SELECT id, user_id, calendar_id, name, enabled, created_at, updated_at
		FROM user_calendars WHERE id = ?`
	row := s.db.QueryRow(query, id.String())

	cal, err := scanUserCalendarRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cal, nil
}

// GetUserCalendarByCalendarID retrieves a user calendar by Google Calendar ID.
func (s *SQLiteStore) GetUserCalendarByCalendarID(userID, calendarID string) (*entity.UserCalendar, error) {
	query := `SELECT id, user_id, calendar_id, name, enabled, created_at, updated_at
		FROM user_calendars WHERE user_id = ? AND calendar_id = ?`
	row := s.db.QueryRow(query, userID, calendarID)

	cal, err := scanUserCalendarRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cal, nil
}

// SaveUserCalendar inserts or updates a user calendar.
func (s *SQLiteStore) SaveUserCalendar(cal *entity.UserCalendar) error {
	query := `INSERT INTO user_calendars (id, user_id, calendar_id, name, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, calendar_id) DO UPDATE SET
			name = excluded.name,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at`

	_, err := s.db.Exec(query,
		cal.ID.String(),
		cal.UserID,
		cal.CalendarID,
		cal.Name,
		cal.Enabled,
		cal.CreatedAt.Format(time.RFC3339),
		cal.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// DeleteUserCalendar removes a user calendar.
func (s *SQLiteStore) DeleteUserCalendar(id uuid.UUID) error {
	result, err := s.db.Exec("DELETE FROM user_calendars WHERE id = ?", id.String())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteUserCalendarsByUser removes all calendars for a user.
func (s *SQLiteStore) DeleteUserCalendarsByUser(userID string) error {
	_, err := s.db.Exec("DELETE FROM user_calendars WHERE user_id = ?", userID)
	return err
}

// SetUserCalendars replaces all calendars for a user.
func (s *SQLiteStore) SetUserCalendars(userID string, calendars []entity.UserCalendar) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing calendars for user
	if _, err := tx.Exec("DELETE FROM user_calendars WHERE user_id = ?", userID); err != nil {
		return err
	}

	// Insert new calendars
	stmt, err := tx.Prepare(`INSERT INTO user_calendars (id, user_id, calendar_id, name, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, cal := range calendars {
		_, err := stmt.Exec(
			cal.ID.String(),
			cal.UserID,
			cal.CalendarID,
			cal.Name,
			cal.Enabled,
			cal.CreatedAt.Format(time.RFC3339),
			cal.UpdatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func scanUserCalendar(rows *sql.Rows) (entity.UserCalendar, error) {
	var cal entity.UserCalendar
	var id, createdAt, updatedAt string
	err := rows.Scan(&id, &cal.UserID, &cal.CalendarID, &cal.Name, &cal.Enabled, &createdAt, &updatedAt)
	if err != nil {
		return cal, err
	}
	cal.ID, _ = uuid.Parse(id)
	cal.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	cal.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return cal, nil
}

func scanUserCalendarRow(row *sql.Row) (entity.UserCalendar, error) {
	var cal entity.UserCalendar
	var id, createdAt, updatedAt string
	err := row.Scan(&id, &cal.UserID, &cal.CalendarID, &cal.Name, &cal.Enabled, &createdAt, &updatedAt)
	if err != nil {
		return cal, err
	}
	cal.ID, _ = uuid.Parse(id)
	cal.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	cal.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return cal, nil
}

// Calendar Link methods

// ListCalendarLinks returns all calendar links for a trip.
func (s *SQLiteStore) ListCalendarLinks(tripID uuid.UUID) ([]entity.CalendarLink, error) {
	query := `SELECT id, trip_id, item_id, calendar_id, event_id, synced_at
		FROM calendar_links WHERE trip_id = ? ORDER BY synced_at DESC`

	rows, err := s.db.Query(query, tripID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []entity.CalendarLink
	for rows.Next() {
		link, err := scanCalendarLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

// GetCalendarLink retrieves a specific calendar link.
func (s *SQLiteStore) GetCalendarLink(id uuid.UUID) (*entity.CalendarLink, error) {
	query := `SELECT id, trip_id, item_id, calendar_id, event_id, synced_at
		FROM calendar_links WHERE id = ?`
	row := s.db.QueryRow(query, id.String())

	link, err := scanCalendarLinkRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// GetCalendarLinkByEvent retrieves a calendar link by trip and event ID.
func (s *SQLiteStore) GetCalendarLinkByEvent(tripID uuid.UUID, calendarID, eventID string) (*entity.CalendarLink, error) {
	query := `SELECT id, trip_id, item_id, calendar_id, event_id, synced_at
		FROM calendar_links WHERE trip_id = ? AND calendar_id = ? AND event_id = ?`
	row := s.db.QueryRow(query, tripID.String(), calendarID, eventID)

	link, err := scanCalendarLinkRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// CreateCalendarLink inserts a new calendar link.
func (s *SQLiteStore) CreateCalendarLink(link *entity.CalendarLink) error {
	query := `INSERT INTO calendar_links (id, trip_id, item_id, calendar_id, event_id, synced_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	var itemID interface{}
	if link.ItemID != nil {
		itemID = link.ItemID.String()
	}

	_, err := s.db.Exec(query,
		link.ID.String(),
		link.TripID.String(),
		itemID,
		link.CalendarID,
		link.EventID,
		link.SyncedAt.Format(time.RFC3339),
	)
	return err
}

// DeleteCalendarLink removes a calendar link.
func (s *SQLiteStore) DeleteCalendarLink(id uuid.UUID) error {
	result, err := s.db.Exec("DELETE FROM calendar_links WHERE id = ?", id.String())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCalendarLinksByTrip removes all calendar links for a trip.
func (s *SQLiteStore) DeleteCalendarLinksByTrip(tripID uuid.UUID) error {
	_, err := s.db.Exec("DELETE FROM calendar_links WHERE trip_id = ?", tripID.String())
	return err
}

func scanCalendarLink(rows *sql.Rows) (entity.CalendarLink, error) {
	var link entity.CalendarLink
	var id, tripID, syncedAt string
	var itemID sql.NullString

	err := rows.Scan(&id, &tripID, &itemID, &link.CalendarID, &link.EventID, &syncedAt)
	if err != nil {
		return link, err
	}

	link.ID, _ = uuid.Parse(id)
	link.TripID, _ = uuid.Parse(tripID)
	link.SyncedAt, _ = time.Parse(time.RFC3339, syncedAt)

	if itemID.Valid && itemID.String != "" {
		iid, _ := uuid.Parse(itemID.String)
		link.ItemID = &iid
	}

	return link, nil
}

func scanCalendarLinkRow(row *sql.Row) (entity.CalendarLink, error) {
	var link entity.CalendarLink
	var id, tripID, syncedAt string
	var itemID sql.NullString

	err := row.Scan(&id, &tripID, &itemID, &link.CalendarID, &link.EventID, &syncedAt)
	if err != nil {
		return link, err
	}

	link.ID, _ = uuid.Parse(id)
	link.TripID, _ = uuid.Parse(tripID)
	link.SyncedAt, _ = time.Parse(time.RFC3339, syncedAt)

	if itemID.Valid && itemID.String != "" {
		iid, _ := uuid.Parse(itemID.String)
		link.ItemID = &iid
	}

	return link, nil
}

// Processed Calendar Events methods

// CreateProcessedEvent saves a record of a processed calendar event.
func (s *SQLiteStore) CreateProcessedEvent(event *entity.ProcessedCalendarEvent) error {
	query := `INSERT INTO processed_calendar_events
		(id, user_id, calendar_event_id, calendar_id, action, trip_id, item_id, processed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	var tripID, itemID sql.NullString
	if event.TripID != nil {
		tripID = sql.NullString{String: event.TripID.String(), Valid: true}
	}
	if event.ItemID != nil {
		itemID = sql.NullString{String: event.ItemID.String(), Valid: true}
	}

	_, err := s.db.Exec(query,
		event.ID.String(),
		event.UserID,
		event.CalendarEventID,
		event.CalendarID,
		event.Action,
		tripID,
		itemID,
		event.ProcessedAt.Format(time.RFC3339),
	)
	return err
}

// GetProcessedEventByCalendarEvent retrieves a processed event by its calendar event ID.
func (s *SQLiteStore) GetProcessedEventByCalendarEvent(userID string, calendarID, eventID string) (*entity.ProcessedCalendarEvent, error) {
	query := `SELECT id, user_id, calendar_event_id, calendar_id, action, trip_id, item_id, processed_at
		FROM processed_calendar_events WHERE user_id = ? AND calendar_id = ? AND calendar_event_id = ?`

	row := s.db.QueryRow(query, userID, calendarID, eventID)
	event, err := scanProcessedEventRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// IsEventProcessed checks if a calendar event has already been processed.
func (s *SQLiteStore) IsEventProcessed(userID string, calendarID, eventID string) (bool, error) {
	query := `SELECT 1 FROM processed_calendar_events WHERE user_id = ? AND calendar_id = ? AND calendar_event_id = ? LIMIT 1`
	var exists int
	err := s.db.QueryRow(query, userID, calendarID, eventID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListProcessedEvents returns all processed events for a calendar.
func (s *SQLiteStore) ListProcessedEvents(userID string, calendarID string) ([]entity.ProcessedCalendarEvent, error) {
	query := `SELECT id, user_id, calendar_event_id, calendar_id, action, trip_id, item_id, processed_at
		FROM processed_calendar_events WHERE user_id = ? AND calendar_id = ? ORDER BY processed_at DESC`

	rows, err := s.db.Query(query, userID, calendarID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []entity.ProcessedCalendarEvent
	for rows.Next() {
		event, err := scanProcessedEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// DeleteAllProcessedEvents removes all processed event records for a user.
func (s *SQLiteStore) DeleteAllProcessedEvents(userID string) error {
	_, err := s.db.Exec("DELETE FROM processed_calendar_events WHERE user_id = ?", userID)
	return err
}

func scanProcessedEvent(rows *sql.Rows) (entity.ProcessedCalendarEvent, error) {
	var event entity.ProcessedCalendarEvent
	var id, processedAt string
	var tripID, itemID sql.NullString

	err := rows.Scan(&id, &event.UserID, &event.CalendarEventID, &event.CalendarID, &event.Action, &tripID, &itemID, &processedAt)
	if err != nil {
		return event, err
	}

	event.ID, _ = uuid.Parse(id)
	event.ProcessedAt, _ = time.Parse(time.RFC3339, processedAt)

	if tripID.Valid && tripID.String != "" {
		tid, _ := uuid.Parse(tripID.String)
		event.TripID = &tid
	}
	if itemID.Valid && itemID.String != "" {
		iid, _ := uuid.Parse(itemID.String)
		event.ItemID = &iid
	}

	return event, nil
}

func scanProcessedEventRow(row *sql.Row) (entity.ProcessedCalendarEvent, error) {
	var event entity.ProcessedCalendarEvent
	var id, processedAt string
	var tripID, itemID sql.NullString

	err := row.Scan(&id, &event.UserID, &event.CalendarEventID, &event.CalendarID, &event.Action, &tripID, &itemID, &processedAt)
	if err != nil {
		return event, err
	}

	event.ID, _ = uuid.Parse(id)
	event.ProcessedAt, _ = time.Parse(time.RFC3339, processedAt)

	if tripID.Valid && tripID.String != "" {
		tid, _ := uuid.Parse(tripID.String)
		event.TripID = &tid
	}
	if itemID.Valid && itemID.String != "" {
		iid, _ := uuid.Parse(itemID.String)
		event.ItemID = &iid
	}

	return event, nil
}

// Session methods

// CreateSession inserts a new session.
func (s *SQLiteStore) CreateSession(session *entity.Session) error {
	query := `INSERT INTO sessions (id, user_id, email, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query,
		session.ID,
		session.UserID,
		session.Email,
		session.ExpiresAt.Format(time.RFC3339),
		session.CreatedAt.Format(time.RFC3339),
	)
	return err
}

// GetSession retrieves a session by ID.
func (s *SQLiteStore) GetSession(id string) (*entity.Session, error) {
	query := `SELECT id, user_id, email, expires_at, created_at FROM sessions WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var session entity.Session
	var expiresAt, createdAt string
	err := row.Scan(&session.ID, &session.UserID, &session.Email, &expiresAt, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	session.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	session.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &session, nil
}

// DeleteSession removes a session by ID.
func (s *SQLiteStore) DeleteSession(id string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

// DeleteExpiredSessions removes all expired sessions.
func (s *SQLiteStore) DeleteExpiredSessions() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().Format(time.RFC3339))
	return err
}

// Silence the unused import warning
var _ = strings.TrimSpace
