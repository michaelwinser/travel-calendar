package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/travel-calendar/backend/internal/api"
	"github.com/user/travel-calendar/backend/internal/store"
)

// setupTestService creates a service with an in-memory database for testing.
func setupTestService(t *testing.T) *Service {
	s, err := store.New(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return New(s)
}

// Helper to create API date
func apiDate(year, month, day int) *types.Date {
	return &types.Date{Time: time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)}
}

// Helper to create a test trip via service
func createTestTrip(t *testing.T, svc *Service, name string, purpose api.TripPurpose) *api.Trip {
	req := &api.CreateTripRequest{
		Name:    name,
		Purpose: purpose,
	}
	trip, err := svc.CreateTrip(req)
	require.NoError(t, err)
	require.NotNil(t, trip)
	return trip
}

// =============================================================================
// Trip Service Tests
// =============================================================================

func TestListTrips_Empty(t *testing.T) {
	svc := setupTestService(t)

	trips, err := svc.ListTrips(nil, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, trips)
}

func TestListTrips_ReturnsAll(t *testing.T) {
	svc := setupTestService(t)

	createTestTrip(t, svc, "Trip 1", api.TripPurposeVacation)
	createTestTrip(t, svc, "Trip 2", api.TripPurposeBusiness)

	trips, err := svc.ListTrips(nil, nil, nil)
	require.NoError(t, err)
	assert.Len(t, trips, 2)
}

func TestListTrips_ConvertsToAPI(t *testing.T) {
	svc := setupTestService(t)

	req := &api.CreateTripRequest{
		Name:      "Test Trip",
		Purpose:   api.TripPurposeConference,
		StartDate: apiDate(2025, 6, 1),
		EndDate:   apiDate(2025, 6, 10),
	}
	_, err := svc.CreateTrip(req)
	require.NoError(t, err)

	trips, err := svc.ListTrips(nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, trips, 1)

	// Verify API types
	trip := trips[0]
	assert.Equal(t, "Test Trip", trip.Name)
	assert.Equal(t, api.TripPurposeConference, trip.Purpose)
	assert.NotNil(t, trip.StartDate)
	assert.NotNil(t, trip.EndDate)
	assert.False(t, trip.CreatedAt.IsZero())
	assert.False(t, trip.UpdatedAt.IsZero())
}

func TestListTrips_FiltersByPurpose(t *testing.T) {
	svc := setupTestService(t)

	createTestTrip(t, svc, "Vacation Trip", api.TripPurposeVacation)
	createTestTrip(t, svc, "Business Trip", api.TripPurposeBusiness)

	purpose := api.TripPurposeVacation
	trips, err := svc.ListTrips(nil, nil, &purpose)
	require.NoError(t, err)
	assert.Len(t, trips, 1)
	assert.Equal(t, "Vacation Trip", trips[0].Name)
}

func TestGetTrip_Found(t *testing.T) {
	svc := setupTestService(t)

	created := createTestTrip(t, svc, "Test Trip", api.TripPurposeVacation)

	trip, err := svc.GetTrip(uuid.UUID(created.Id))
	require.NoError(t, err)
	require.NotNil(t, trip)
	assert.Equal(t, "Test Trip", trip.Name)
}

func TestGetTrip_NotFound(t *testing.T) {
	svc := setupTestService(t)

	trip, err := svc.GetTrip(uuid.New())
	require.NoError(t, err)
	assert.Nil(t, trip)
}

func TestGetTrip_IncludesItems(t *testing.T) {
	svc := setupTestService(t)

	// Create trip
	created := createTestTrip(t, svc, "Trip with Items", api.TripPurposeVacation)
	tripID := uuid.UUID(created.Id)

	// Add items
	itemReq := &api.CreateItemRequest{
		Type: api.Flight,
	}
	_, err := svc.CreateTripItem(tripID, itemReq)
	require.NoError(t, err)

	// Get trip should include items
	trip, err := svc.GetTrip(tripID)
	require.NoError(t, err)
	require.NotNil(t, trip)
	require.NotNil(t, trip.Items)
	assert.Len(t, *trip.Items, 1)
}

func TestCreateTrip_Success(t *testing.T) {
	svc := setupTestService(t)

	notes := "Test notes"
	req := &api.CreateTripRequest{
		Name:      "New Trip",
		Purpose:   api.TripPurposeFamily,
		StartDate: apiDate(2025, 7, 1),
		EndDate:   apiDate(2025, 7, 15),
		Notes:     &notes,
	}

	trip, err := svc.CreateTrip(req)
	require.NoError(t, err)
	require.NotNil(t, trip)

	assert.Equal(t, "New Trip", trip.Name)
	assert.Equal(t, api.TripPurposeFamily, trip.Purpose)
	assert.Equal(t, api.Planning, trip.Status) // Default status
	assert.NotNil(t, trip.StartDate)
	assert.NotNil(t, trip.EndDate)
	assert.NotNil(t, trip.Notes)
	assert.Equal(t, "Test notes", *trip.Notes)
}

func TestCreateTrip_DefaultStatus(t *testing.T) {
	svc := setupTestService(t)

	req := &api.CreateTripRequest{
		Name:    "Trip",
		Purpose: api.TripPurposeVacation,
	}

	trip, err := svc.CreateTrip(req)
	require.NoError(t, err)
	assert.Equal(t, api.Planning, trip.Status)
}

func TestCreateTrip_CustomStatus(t *testing.T) {
	svc := setupTestService(t)

	status := api.Confirmed
	req := &api.CreateTripRequest{
		Name:    "Trip",
		Purpose: api.TripPurposeVacation,
		Status:  &status,
	}

	trip, err := svc.CreateTrip(req)
	require.NoError(t, err)
	assert.Equal(t, api.Confirmed, trip.Status)
}

func TestUpdateTrip_Success(t *testing.T) {
	svc := setupTestService(t)

	created := createTestTrip(t, svc, "Original Name", api.TripPurposeVacation)
	tripID := uuid.UUID(created.Id)

	newName := "Updated Name"
	newPurpose := api.TripPurposeBusiness
	req := &api.UpdateTripRequest{
		Name:    &newName,
		Purpose: &newPurpose,
	}

	updated, err := svc.UpdateTrip(tripID, req)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, api.TripPurposeBusiness, updated.Purpose)
}

func TestUpdateTrip_PartialUpdate(t *testing.T) {
	svc := setupTestService(t)

	// Create with specific values
	notes := "Original notes"
	status := api.Confirmed
	req := &api.CreateTripRequest{
		Name:    "Original Trip",
		Purpose: api.TripPurposeConference,
		Notes:   &notes,
		Status:  &status,
	}
	created, err := svc.CreateTrip(req)
	require.NoError(t, err)
	tripID := uuid.UUID(created.Id)

	// Update only name
	newName := "Updated Name"
	updateReq := &api.UpdateTripRequest{
		Name: &newName,
	}

	updated, err := svc.UpdateTrip(tripID, updateReq)
	require.NoError(t, err)
	require.NotNil(t, updated)

	// Name changed
	assert.Equal(t, "Updated Name", updated.Name)
	// Other fields preserved
	assert.Equal(t, api.TripPurposeConference, updated.Purpose)
	assert.Equal(t, api.Confirmed, updated.Status)
	assert.NotNil(t, updated.Notes)
	assert.Equal(t, "Original notes", *updated.Notes)
}

func TestUpdateTrip_NotFound(t *testing.T) {
	svc := setupTestService(t)

	newName := "Updated"
	req := &api.UpdateTripRequest{
		Name: &newName,
	}

	updated, err := svc.UpdateTrip(uuid.New(), req)
	require.NoError(t, err)
	assert.Nil(t, updated)
}

func TestDeleteTrip_Success(t *testing.T) {
	svc := setupTestService(t)

	created := createTestTrip(t, svc, "To Delete", api.TripPurposeVacation)
	tripID := uuid.UUID(created.Id)

	err := svc.DeleteTrip(tripID)
	require.NoError(t, err)

	// Verify deleted
	trip, err := svc.GetTrip(tripID)
	require.NoError(t, err)
	assert.Nil(t, trip)
}

func TestSearchTrips_FindsByName(t *testing.T) {
	svc := setupTestService(t)

	createTestTrip(t, svc, "FOSDEM Conference", api.TripPurposeConference)
	createTestTrip(t, svc, "Beach Vacation", api.TripPurposeVacation)

	trips, err := svc.SearchTrips("FOSDEM")
	require.NoError(t, err)
	assert.Len(t, trips, 1)
	assert.Equal(t, "FOSDEM Conference", trips[0].Name)
}

func TestSearchTrips_CaseInsensitive(t *testing.T) {
	svc := setupTestService(t)

	createTestTrip(t, svc, "FOSDEM Conference", api.TripPurposeConference)

	trips, err := svc.SearchTrips("fosdem")
	require.NoError(t, err)
	assert.Len(t, trips, 1)
}

// =============================================================================
// Item Service Tests
// =============================================================================

func TestListTripItems_ReturnsItems(t *testing.T) {
	svc := setupTestService(t)

	trip := createTestTrip(t, svc, "Trip", api.TripPurposeVacation)
	tripID := uuid.UUID(trip.Id)

	// Create item
	itemReq := &api.CreateItemRequest{
		Type: api.Flight,
	}
	_, err := svc.CreateTripItem(tripID, itemReq)
	require.NoError(t, err)

	items, err := svc.ListTripItems(tripID)
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, api.Flight, items[0].Type)
}

func TestListTripItems_TripNotFound(t *testing.T) {
	svc := setupTestService(t)

	items, err := svc.ListTripItems(uuid.New())
	require.NoError(t, err)
	assert.Nil(t, items) // Returns nil when trip not found
}

func TestCreateTripItem_Success(t *testing.T) {
	svc := setupTestService(t)

	trip := createTestTrip(t, svc, "Trip", api.TripPurposeVacation)
	tripID := uuid.UUID(trip.Id)

	from := "JFK"
	to := "LAX"
	carrier := "Delta"
	req := &api.CreateItemRequest{
		Type:    api.Flight,
		From:    &from,
		To:      &to,
		Carrier: &carrier,
	}

	item, err := svc.CreateTripItem(tripID, req)
	require.NoError(t, err)
	require.NotNil(t, item)

	assert.Equal(t, api.Flight, item.Type)
	assert.Equal(t, tripID, uuid.UUID(item.TripId))
	assert.Equal(t, "JFK", *item.From)
	assert.Equal(t, "LAX", *item.To)
	assert.Equal(t, "Delta", *item.Carrier)
}

func TestCreateTripItem_TripNotFound(t *testing.T) {
	svc := setupTestService(t)

	req := &api.CreateItemRequest{
		Type: api.Flight,
	}

	item, err := svc.CreateTripItem(uuid.New(), req)
	require.NoError(t, err)
	assert.Nil(t, item) // Returns nil when trip not found
}

func TestDeleteItem_Success(t *testing.T) {
	svc := setupTestService(t)

	trip := createTestTrip(t, svc, "Trip", api.TripPurposeVacation)
	tripID := uuid.UUID(trip.Id)

	// Create item
	itemReq := &api.CreateItemRequest{
		Type: api.Event,
	}
	item, err := svc.CreateTripItem(tripID, itemReq)
	require.NoError(t, err)

	// Delete item
	err = svc.DeleteItem(uuid.UUID(item.Id))
	require.NoError(t, err)

	// Verify deleted
	items, err := svc.ListTripItems(tripID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

// =============================================================================
// Document Service Tests
// =============================================================================

func TestListDocuments_Empty(t *testing.T) {
	svc := setupTestService(t)

	docs, err := svc.ListDocuments(nil, nil)
	require.NoError(t, err)
	assert.Empty(t, docs)
}
