package store

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/travel-calendar/backend/internal/entity"
)

// =============================================================================
// SQLite-specific test helpers (use raw SQL, can't go through StoreInterface)
// =============================================================================

// setupTestDB creates an in-memory SQLite store for testing.
func setupTestDB(t *testing.T) *SQLiteStore {
	store, err := NewSQLite(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}

// createTestDocument inserts a document via raw SQL since there's no CreateDocument
// in the StoreInterface. This is SQLite-specific.
func createTestDocument(t *testing.T, store *SQLiteStore, name, docType string, tripID *uuid.UUID) entity.Document {
	doc := entity.Document{
		ID:        uuid.New(),
		TripID:    tripID,
		Name:      name,
		Type:      docType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	query := `INSERT INTO documents (id, trip_id, name, type, url, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	var tripIDStr interface{}
	if tripID != nil {
		tripIDStr = tripID.String()
	}
	_, err := store.db.Exec(query, doc.ID.String(), tripIDStr, doc.Name, doc.Type, nil, doc.CreatedAt.Format(time.RFC3339), doc.UpdatedAt.Format(time.RFC3339))
	require.NoError(t, err)
	return doc
}

// createTestTripSQLite creates a trip using the concrete SQLiteStore.
func createTestTripSQLite(t *testing.T, store *SQLiteStore, name, purpose string, startDate, endDate *time.Time) entity.Trip {
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

// =============================================================================
// SQLite-specific Document Tests (require raw SQL for setup)
// =============================================================================

func TestListDocuments_ByTripID(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTripSQLite(t, store, "Trip", "vacation", nil, nil)
	doc := createTestDocument(t, store, "Doc 1", "pdf", &trip.ID)

	otherTrip := createTestTripSQLite(t, store, "Other Trip", "business", nil, nil)
	createTestDocument(t, store, "Doc 2", "pdf", &otherTrip.ID)

	docs, err := store.ListDocuments(&trip.ID, nil)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "Doc 1", docs[0].Name)
	assert.Equal(t, doc.ID, docs[0].ID)
}

func TestListDocuments_Unassociated(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTripSQLite(t, store, "Trip", "vacation", nil, nil)
	createTestDocument(t, store, "Associated Doc", "pdf", &trip.ID)
	createTestDocument(t, store, "Unassociated Doc", "pdf", nil)

	docs, err := store.ListDocuments(nil, boolPtr(true))
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "Unassociated Doc", docs[0].Name)
}

func TestDeleteTrip_PreservesDocuments(t *testing.T) {
	store := setupTestDB(t)

	trip := createTestTripSQLite(t, store, "Trip", "vacation", nil, nil)
	doc := createTestDocument(t, store, "Trip Doc", "pdf", &trip.ID)

	err := store.DeleteTrip(trip.ID)
	require.NoError(t, err)

	// Document should still exist but be unassociated
	docs, err := store.ListDocuments(nil, boolPtr(true))
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, doc.ID, docs[0].ID)
	assert.Nil(t, docs[0].TripID)
}
