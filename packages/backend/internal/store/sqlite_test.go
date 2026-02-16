package store

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/travel-calendar/backend/internal/entity"
)

// setupTestDB creates an in-memory SQLite store for testing.
func setupTestDB(t *testing.T) *SQLiteStore {
	store, err := NewSQLite(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}

// Helper to create a test trip
func createTestTrip(t *testing.T, store *SQLiteStore, name, purpose string, startDate, endDate *time.Time) entity.Trip {
	trip := entity.Trip{
		ID:        uuid.New(),
		Name:      name,
		Purpose:   purpose,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    "planning",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := store.CreateTrip(&trip)
	require.NoError(t, err)
	return trip
}

// Helper to create a pointer to time
func timePtr(t time.Time) *time.Time {
	return &t
}

// Helper to create a pointer to string
func strPtr(s string) *string {
	return &s
}

// Helper to create a pointer to bool
func boolPtr(b bool) *bool {
	return &b
}

// =============================================================================
// Trip Tests
// =============================================================================

func TestListTrips_Empty(t *testing.T) {
	store := setupTestDB(t)

	trips, err := store.ListTrips(nil, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, trips)
}

func TestListTrips_ReturnsAll(t *testing.T) {
	store := setupTestDB(t)

	// Create two trips
	createTestTrip(t, store, "Trip 1", "vacation", nil, nil)
	createTestTrip(t, store, "Trip 2", "business", nil, nil)

	trips, err := store.ListTrips(nil, nil, nil)
	require.NoError(t, err)
	assert.Len(t, trips, 2)
}

func TestListTrips_Upcoming(t *testing.T) {
	store := setupTestDB(t)

	// Create a future trip
	futureDate := time.Now().AddDate(0, 1, 0) // 1 month from now
	createTestTrip(t, store, "Future Trip", "vacation", timePtr(futureDate), nil)

	// Create a past trip
	pastDate := time.Now().AddDate(0, -1, 0) // 1 month ago
	pastEndDate := time.Now().AddDate(0, 0, -15)
	createTestTrip(t, store, "Past Trip", "vacation", timePtr(pastDate), timePtr(pastEndDate))

	trips, err := store.ListTrips(boolPtr(true), nil, nil)
	require.NoError(t, err)
	assert.Len(t, trips, 1)
	assert.Equal(t, "Future Trip", trips[0].Name)
}

func TestListTrips_Past(t *testing.T) {
	store := setupTestDB(t)

	// Create a future trip
	futureDate := time.Now().AddDate(0, 1, 0)
	futureEndDate := time.Now().AddDate(0, 1, 5)
	createTestTrip(t, store, "Future Trip", "vacation", timePtr(futureDate), timePtr(futureEndDate))

	// Create a past trip
	pastDate := time.Now().AddDate(0, -1, 0)
	pastEndDate := time.Now().AddDate(0, 0, -15)
	createTestTrip(t, store, "Past Trip", "vacation", timePtr(pastDate), timePtr(pastEndDate))

	trips, err := store.ListTrips(nil, boolPtr(true), nil)
	require.NoError(t, err)
	assert.Len(t, trips, 1)
	assert.Equal(t, "Past Trip", trips[0].Name)
}

func TestListTrips_ByPurpose(t *testing.T) {
	store := setupTestDB(t)

	createTestTrip(t, store, "Vacation Trip", "vacation", nil, nil)
	createTestTrip(t, store, "Business Trip", "business", nil, nil)

	trips, err := store.ListTrips(nil, nil, strPtr("vacation"))
	require.NoError(t, err)
	assert.Len(t, trips, 1)
	assert.Equal(t, "Vacation Trip", trips[0].Name)
}

func TestListTrips_SortedByStartDate(t *testing.T) {
	store := setupTestDB(t)

	// Create trips out of order
	date3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	date1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)

	createTestTrip(t, store, "March Trip", "vacation", timePtr(date3), nil)
	createTestTrip(t, store, "January Trip", "vacation", timePtr(date1), nil)
	createTestTrip(t, store, "February Trip", "vacation", timePtr(date2), nil)

	trips, err := store.ListTrips(nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, trips, 3)

	// Should be sorted by start date ascending
	assert.Equal(t, "January Trip", trips[0].Name)
	assert.Equal(t, "February Trip", trips[1].Name)
	assert.Equal(t, "March Trip", trips[2].Name)
}

func TestListTrips_NullDatesLast(t *testing.T) {
	store := setupTestDB(t)

	// Create trip without date
	createTestTrip(t, store, "Unscheduled Trip", "vacation", nil, nil)

	// Create trip with date
	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	createTestTrip(t, store, "Scheduled Trip", "vacation", timePtr(date), nil)

	trips, err := store.ListTrips(nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, trips, 2)

	// Scheduled trips come first, unscheduled last
	assert.Equal(t, "Scheduled Trip", trips[0].Name)
	assert.Equal(t, "Unscheduled Trip", trips[1].Name)
}

func TestGetTrip_Found(t *testing.T) {
	store := setupTestDB(t)

	created := createTestTrip(t, store, "Test Trip", "vacation", nil, nil)

	trip, err := store.GetTrip(created.ID)
	require.NoError(t, err)
	require.NotNil(t, trip)
	assert.Equal(t, "Test Trip", trip.Name)
	assert.Equal(t, "vacation", trip.Purpose)
}

func TestGetTrip_NotFound(t *testing.T) {
	store := setupTestDB(t)

	trip, err := store.GetTrip(uuid.New())
	require.NoError(t, err)
	assert.Nil(t, trip)
}

func TestCreateTrip_Success(t *testing.T) {
	store := setupTestDB(t)

	startDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	notes := "Test notes"

	trip := entity.Trip{
		ID:        uuid.New(),
		Name:      "Summer Vacation",
		Purpose:   "vacation",
		StartDate: &startDate,
		EndDate:   &endDate,
		Status:    "planning",
		Notes:     &notes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.CreateTrip(&trip)
	require.NoError(t, err)

	// Verify by reading back
	retrieved, err := store.GetTrip(trip.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "Summer Vacation", retrieved.Name)
	assert.Equal(t, "vacation", retrieved.Purpose)
	assert.NotNil(t, retrieved.StartDate)
	assert.NotNil(t, retrieved.EndDate)
	assert.NotNil(t, retrieved.Notes)
	assert.Equal(t, "Test notes", *retrieved.Notes)
}

func TestCreateTrip_SetsTimestamps(t *testing.T) {
	store := setupTestDB(t)

	now := time.Now()
	trip := entity.Trip{
		ID:        uuid.New(),
		Name:      "Test Trip",
		Purpose:   "vacation",
		Status:    "planning",
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := store.CreateTrip(&trip)
	require.NoError(t, err)

	retrieved, err := store.GetTrip(trip.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.CreatedAt.IsZero())
	assert.False(t, retrieved.UpdatedAt.IsZero())
}

func TestUpdateTrip_Success(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTrip(t, store, "Original Name", "vacation", nil, nil)

	// Update the trip
	trip.Name = "Updated Name"
	trip.Purpose = "business"
	trip.UpdatedAt = time.Now()

	err := store.UpdateTrip(&trip)
	require.NoError(t, err)

	// Verify
	retrieved, err := store.GetTrip(trip.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, "business", retrieved.Purpose)
}

func TestUpdateTrip_NotFound(t *testing.T) {
	store := setupTestDB(t)

	trip := entity.Trip{
		ID:        uuid.New(),
		Name:      "Nonexistent",
		Purpose:   "vacation",
		Status:    "planning",
		UpdatedAt: time.Now(),
	}

	err := store.UpdateTrip(&trip)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteTrip_Success(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTrip(t, store, "To Delete", "vacation", nil, nil)

	err := store.DeleteTrip(trip.ID)
	require.NoError(t, err)

	// Verify deleted
	retrieved, err := store.GetTrip(trip.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestDeleteTrip_NotFound(t *testing.T) {
	store := setupTestDB(t)

	err := store.DeleteTrip(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteTrip_CascadesItems(t *testing.T) {
	store := setupTestDB(t)

	// Create trip with item
	trip := createTestTrip(t, store, "Trip with Items", "vacation", nil, nil)

	item := entity.Item{
		ID:        uuid.New(),
		TripID:    trip.ID,
		Type:      "flight",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := store.CreateItem(&item)
	require.NoError(t, err)

	// Verify item exists
	items, err := store.ListItems(trip.ID)
	require.NoError(t, err)
	assert.Len(t, items, 1)

	// Delete trip
	err = store.DeleteTrip(trip.ID)
	require.NoError(t, err)

	// Items should be cascade deleted
	items, err = store.ListItems(trip.ID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestSearchTrips_ByName(t *testing.T) {
	store := setupTestDB(t)

	createTestTrip(t, store, "FOSDEM Conference", "conference", nil, nil)
	createTestTrip(t, store, "Beach Vacation", "vacation", nil, nil)

	trips, err := store.SearchTrips("FOSDEM")
	require.NoError(t, err)
	assert.Len(t, trips, 1)
	assert.Equal(t, "FOSDEM Conference", trips[0].Name)
}

func TestSearchTrips_ByNotes(t *testing.T) {
	store := setupTestDB(t)

	notes := "Meeting with important clients"
	trip := entity.Trip{
		ID:        uuid.New(),
		Name:      "Business Trip",
		Purpose:   "business",
		Status:    "planning",
		Notes:     &notes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := store.CreateTrip(&trip)
	require.NoError(t, err)

	createTestTrip(t, store, "Other Trip", "vacation", nil, nil)

	trips, err := store.SearchTrips("clients")
	require.NoError(t, err)
	assert.Len(t, trips, 1)
	assert.Equal(t, "Business Trip", trips[0].Name)
}

func TestSearchTrips_CaseInsensitive(t *testing.T) {
	store := setupTestDB(t)

	createTestTrip(t, store, "FOSDEM Conference", "conference", nil, nil)

	// Search with different case
	trips, err := store.SearchTrips("fosdem")
	require.NoError(t, err)
	assert.Len(t, trips, 1)
	assert.Equal(t, "FOSDEM Conference", trips[0].Name)
}

func TestSearchTrips_PartialMatch(t *testing.T) {
	store := setupTestDB(t)

	createTestTrip(t, store, "Summer Vacation 2025", "vacation", nil, nil)

	trips, err := store.SearchTrips("Vacat")
	require.NoError(t, err)
	assert.Len(t, trips, 1)
	assert.Equal(t, "Summer Vacation 2025", trips[0].Name)
}

// =============================================================================
// Item Tests
// =============================================================================

func TestListItems_Empty(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTrip(t, store, "Empty Trip", "vacation", nil, nil)

	items, err := store.ListItems(trip.ID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestListItems_SortedByDateTime(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTrip(t, store, "Trip", "vacation", nil, nil)

	// Create items out of order
	date2 := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
	date1 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	item2 := entity.Item{
		ID:        uuid.New(),
		TripID:    trip.ID,
		Type:      "event",
		Date:      &date2,
		Time:      strPtr("10:00"),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := store.CreateItem(&item2)
	require.NoError(t, err)

	item1 := entity.Item{
		ID:        uuid.New(),
		TripID:    trip.ID,
		Type:      "flight",
		Date:      &date1,
		Time:      strPtr("08:00"),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = store.CreateItem(&item1)
	require.NoError(t, err)

	items, err := store.ListItems(trip.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)

	// Should be sorted by date, then time
	assert.Equal(t, "flight", items[0].Type)
	assert.Equal(t, "event", items[1].Type)
}

func TestGetItem_Found(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTrip(t, store, "Trip", "vacation", nil, nil)

	item := entity.Item{
		ID:        uuid.New(),
		TripID:    trip.ID,
		Type:      "flight",
		From:      strPtr("JFK"),
		To:        strPtr("LAX"),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := store.CreateItem(&item)
	require.NoError(t, err)

	retrieved, err := store.GetItem(item.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "flight", retrieved.Type)
	assert.Equal(t, "JFK", *retrieved.From)
	assert.Equal(t, "LAX", *retrieved.To)
}

func TestGetItem_NotFound(t *testing.T) {
	store := setupTestDB(t)

	item, err := store.GetItem(uuid.New())
	require.NoError(t, err)
	assert.Nil(t, item)
}

func TestCreateItem_Success(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTrip(t, store, "Trip", "vacation", nil, nil)
	date := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	item := entity.Item{
		ID:           uuid.New(),
		TripID:       trip.ID,
		Type:         "flight",
		Date:         &date,
		Time:         strPtr("08:00"),
		From:         strPtr("JFK"),
		To:           strPtr("LAX"),
		Carrier:      strPtr("Delta"),
		FlightNumber: strPtr("DL123"),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err := store.CreateItem(&item)
	require.NoError(t, err)

	// Verify
	retrieved, err := store.GetItem(item.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "flight", retrieved.Type)
	assert.Equal(t, "JFK", *retrieved.From)
	assert.Equal(t, "LAX", *retrieved.To)
	assert.Equal(t, "Delta", *retrieved.Carrier)
	assert.Equal(t, "DL123", *retrieved.FlightNumber)
}

func TestCreateItem_OptionalFieldsNull(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTrip(t, store, "Trip", "vacation", nil, nil)

	// Create item with minimal fields
	item := entity.Item{
		ID:        uuid.New(),
		TripID:    trip.ID,
		Type:      "event",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := store.CreateItem(&item)
	require.NoError(t, err)

	retrieved, err := store.GetItem(item.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "event", retrieved.Type)
	assert.Nil(t, retrieved.Date)
	assert.Nil(t, retrieved.Time)
	assert.Nil(t, retrieved.From)
	assert.Nil(t, retrieved.To)
}

func TestDeleteItem_Success(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTrip(t, store, "Trip", "vacation", nil, nil)

	item := entity.Item{
		ID:        uuid.New(),
		TripID:    trip.ID,
		Type:      "flight",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := store.CreateItem(&item)
	require.NoError(t, err)

	err = store.DeleteItem(item.ID)
	require.NoError(t, err)

	// Verify deleted
	retrieved, err := store.GetItem(item.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestDeleteItem_NotFound(t *testing.T) {
	store := setupTestDB(t)

	err := store.DeleteItem(uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

// =============================================================================
// Document Tests
// =============================================================================

func TestListDocuments_ByTripID(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTrip(t, store, "Trip", "vacation", nil, nil)

	// Create document for this trip
	doc := createTestDocument(t, store, "Doc 1", "pdf", &trip.ID)

	// Create document for another trip
	otherTrip := createTestTrip(t, store, "Other Trip", "business", nil, nil)
	createTestDocument(t, store, "Doc 2", "pdf", &otherTrip.ID)

	docs, err := store.ListDocuments(&trip.ID, nil)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "Doc 1", docs[0].Name)
	assert.Equal(t, doc.ID, docs[0].ID)
}

func TestListDocuments_Unassociated(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTrip(t, store, "Trip", "vacation", nil, nil)

	// Create associated document
	createTestDocument(t, store, "Associated Doc", "pdf", &trip.ID)

	// Create unassociated document
	createTestDocument(t, store, "Unassociated Doc", "pdf", nil)

	docs, err := store.ListDocuments(nil, boolPtr(true))
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "Unassociated Doc", docs[0].Name)
}

// Helper to create test document
func createTestDocument(t *testing.T, store *SQLiteStore, name, docType string, tripID *uuid.UUID) entity.Document {
	doc := entity.Document{
		ID:        uuid.New(),
		TripID:    tripID,
		Name:      name,
		Type:      docType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// Direct insert since we don't have CreateDocument method
	query := `INSERT INTO documents (id, trip_id, name, type, url, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	var tripIDStr interface{}
	if tripID != nil {
		tripIDStr = tripID.String()
	}
	_, err := store.db.Exec(query, doc.ID.String(), tripIDStr, doc.Name, doc.Type, nil, doc.CreatedAt.Format(time.RFC3339), doc.UpdatedAt.Format(time.RFC3339))
	require.NoError(t, err)
	return doc
}

func TestDeleteTrip_PreservesDocuments(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTrip(t, store, "Trip", "vacation", nil, nil)

	// Create document associated with trip
	doc := createTestDocument(t, store, "Trip Doc", "pdf", &trip.ID)

	// Delete the trip
	err := store.DeleteTrip(trip.ID)
	require.NoError(t, err)

	// Document should still exist but be unassociated
	docs, err := store.ListDocuments(nil, boolPtr(true))
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, doc.ID, docs[0].ID)
	assert.Nil(t, docs[0].TripID) // Should be NULL after cascade
}
