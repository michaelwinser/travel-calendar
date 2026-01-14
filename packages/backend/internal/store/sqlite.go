// Package store provides database access layer using SQLite.
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

// Store provides database access methods.
type Store struct {
	db *sql.DB
}

// New creates a new Store with the given database path.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates the database schema.
func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS trips (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		purpose TEXT NOT NULL,
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
	`

	_, err := s.db.Exec(schema)
	return err
}

// Trip methods

// ListTrips returns all trips, optionally filtered.
func (s *Store) ListTrips(upcoming, past *bool, purpose *string) ([]entity.Trip, error) {
	query := "SELECT id, name, purpose, start_date, end_date, status, notes, created_at, updated_at FROM trips WHERE 1=1"
	args := []interface{}{}

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
func (s *Store) GetTrip(id uuid.UUID) (*entity.Trip, error) {
	query := "SELECT id, name, purpose, start_date, end_date, status, notes, created_at, updated_at FROM trips WHERE id = ?"
	row := s.db.QueryRow(query, id.String())
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
func (s *Store) CreateTrip(trip *entity.Trip) error {
	query := `INSERT INTO trips (id, name, purpose, start_date, end_date, status, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query,
		trip.ID.String(),
		trip.Name,
		trip.Purpose,
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
func (s *Store) UpdateTrip(trip *entity.Trip) error {
	query := `UPDATE trips SET name = ?, purpose = ?, start_date = ?, end_date = ?, status = ?, notes = ?, updated_at = ? WHERE id = ?`
	result, err := s.db.Exec(query,
		trip.Name,
		trip.Purpose,
		formatDatePtr(trip.StartDate),
		formatDatePtr(trip.EndDate),
		trip.Status,
		trip.Notes,
		trip.UpdatedAt.Format(time.RFC3339),
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
		return sql.ErrNoRows
	}
	return nil
}

// DeleteTrip deletes a trip by ID.
func (s *Store) DeleteTrip(id uuid.UUID) error {
	result, err := s.db.Exec("DELETE FROM trips WHERE id = ?", id.String())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SearchTrips searches trips by query string.
func (s *Store) SearchTrips(q string) ([]entity.Trip, error) {
	pattern := "%" + q + "%"
	query := `SELECT id, name, purpose, start_date, end_date, status, notes, created_at, updated_at FROM trips
		WHERE name LIKE ? OR notes LIKE ?
		ORDER BY COALESCE(start_date, '9999-12-31') ASC`

	rows, err := s.db.Query(query, pattern, pattern)
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
func (s *Store) ListItems(tripID uuid.UUID) ([]entity.Item, error) {
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
func (s *Store) GetItem(id uuid.UUID) (*entity.Item, error) {
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
func (s *Store) CreateItem(item *entity.Item) error {
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
func (s *Store) DeleteItem(id uuid.UUID) error {
	result, err := s.db.Exec("DELETE FROM items WHERE id = ?", id.String())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Document methods

// ListDocuments returns documents, optionally filtered by trip or unassociated.
func (s *Store) ListDocuments(tripID *uuid.UUID, unassociated *bool) ([]entity.Document, error) {
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
	var id, startDate, endDate, notes, createdAt, updatedAt sql.NullString
	err := rows.Scan(&id, &trip.Name, &trip.Purpose, &startDate, &endDate, &trip.Status, &notes, &createdAt, &updatedAt)
	if err != nil {
		return trip, err
	}
	trip.ID, _ = uuid.Parse(id.String)
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
	var id, startDate, endDate, notes, createdAt, updatedAt sql.NullString
	err := row.Scan(&id, &trip.Name, &trip.Purpose, &startDate, &endDate, &trip.Status, &notes, &createdAt, &updatedAt)
	if err != nil {
		return trip, err
	}
	trip.ID, _ = uuid.Parse(id.String)
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

// Silence the unused import warning
var _ = strings.TrimSpace
