package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/travel-calendar/backend/internal/api"
	"github.com/user/travel-calendar/backend/internal/store"
)

// setupTestService creates a service backed by the Firestore emulator for testing.
func setupTestService(t *testing.T) *Service {
	host := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if host == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "travel-calendar-test"
	}

	// Clear emulator data
	clearURL := fmt.Sprintf("http://%s/emulator/v1/projects/%s/databases/(default)/documents", host, projectID)
	req, err := http.NewRequest("DELETE", clearURL, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	s, err := store.NewFirestore(context.Background(), projectID)
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

// =============================================================================
// Trip Organization Tests
// =============================================================================

func TestMergeTrips_Success(t *testing.T) {
	svc := setupTestService(t)

	// Create source trip with items
	source := createTestTrip(t, svc, "Source Trip", api.TripPurposeVacation)
	sourceID := uuid.UUID(source.Id)
	_, err := svc.CreateTripItem(sourceID, &api.CreateItemRequest{Type: api.Flight})
	require.NoError(t, err)
	_, err = svc.CreateTripItem(sourceID, &api.CreateItemRequest{Type: api.Hotel})
	require.NoError(t, err)

	// Create target trip with an item
	target := createTestTrip(t, svc, "Target Trip", api.TripPurposeBusiness)
	targetID := uuid.UUID(target.Id)
	_, err = svc.CreateTripItem(targetID, &api.CreateItemRequest{Type: api.Event})
	require.NoError(t, err)

	// Merge source into target
	req := &api.MergeTripsRequest{}
	merged, err := svc.MergeTrips(sourceID, targetID, req)
	require.NoError(t, err)
	require.NotNil(t, merged)

	// Target trip should now have all 3 items
	assert.Len(t, *merged.Items, 3)

	// Source trip should be deleted
	deleted, err := svc.GetTrip(sourceID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestMergeTrips_ExtendsDates(t *testing.T) {
	svc := setupTestService(t)

	// Create source with wider date range
	sourceReq := &api.CreateTripRequest{
		Name:      "Source",
		Purpose:   api.TripPurposeVacation,
		StartDate: apiDate(2025, 1, 1),
		EndDate:   apiDate(2025, 1, 20),
	}
	source, err := svc.CreateTrip(sourceReq)
	require.NoError(t, err)
	sourceID := uuid.UUID(source.Id)

	// Create target with narrower date range
	targetReq := &api.CreateTripRequest{
		Name:      "Target",
		Purpose:   api.TripPurposeBusiness,
		StartDate: apiDate(2025, 1, 5),
		EndDate:   apiDate(2025, 1, 15),
	}
	target, err := svc.CreateTrip(targetReq)
	require.NoError(t, err)
	targetID := uuid.UUID(target.Id)

	// Merge - target dates should be extended
	merged, err := svc.MergeTrips(sourceID, targetID, &api.MergeTripsRequest{})
	require.NoError(t, err)
	require.NotNil(t, merged)

	// Target should have source's wider date range
	assert.Equal(t, 2025, merged.StartDate.Time.Year())
	assert.Equal(t, 1, int(merged.StartDate.Time.Month()))
	assert.Equal(t, 1, merged.StartDate.Time.Day())
	assert.Equal(t, 20, merged.EndDate.Time.Day())
}

func TestMergeTrips_ConcatenatesNotes(t *testing.T) {
	svc := setupTestService(t)

	// Create trips with notes
	sourceNotes := "Source notes"
	sourceReq := &api.CreateTripRequest{
		Name:    "Source",
		Purpose: api.TripPurposeVacation,
		Notes:   &sourceNotes,
	}
	source, err := svc.CreateTrip(sourceReq)
	require.NoError(t, err)

	targetNotes := "Target notes"
	targetReq := &api.CreateTripRequest{
		Name:    "Target",
		Purpose: api.TripPurposeBusiness,
		Notes:   &targetNotes,
	}
	target, err := svc.CreateTrip(targetReq)
	require.NoError(t, err)

	// Merge with mergeNotes=true (default)
	merged, err := svc.MergeTrips(uuid.UUID(source.Id), uuid.UUID(target.Id), &api.MergeTripsRequest{})
	require.NoError(t, err)
	require.NotNil(t, merged)
	require.NotNil(t, merged.Notes)

	// Notes should be concatenated
	assert.Contains(t, *merged.Notes, "Target notes")
	assert.Contains(t, *merged.Notes, "Source notes")
}

func TestMergeTrips_SameTripError(t *testing.T) {
	svc := setupTestService(t)

	trip := createTestTrip(t, svc, "Trip", api.TripPurposeVacation)
	tripID := uuid.UUID(trip.Id)

	// Try to merge trip into itself
	_, err := svc.MergeTrips(tripID, tripID, &api.MergeTripsRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot merge trip into itself")
}

func TestMergeTrips_SourceNotFound(t *testing.T) {
	svc := setupTestService(t)

	target := createTestTrip(t, svc, "Target", api.TripPurposeBusiness)

	// Try to merge non-existent source
	merged, err := svc.MergeTrips(uuid.New(), uuid.UUID(target.Id), &api.MergeTripsRequest{})
	require.NoError(t, err)
	assert.Nil(t, merged)
}

func TestMergeTrips_TargetNotFound(t *testing.T) {
	svc := setupTestService(t)

	source := createTestTrip(t, svc, "Source", api.TripPurposeVacation)

	// Try to merge into non-existent target
	merged, err := svc.MergeTrips(uuid.UUID(source.Id), uuid.New(), &api.MergeTripsRequest{})
	require.NoError(t, err)
	assert.Nil(t, merged)
}

func TestMoveItem_ToExistingTrip(t *testing.T) {
	svc := setupTestService(t)

	// Create two trips
	trip1 := createTestTrip(t, svc, "Trip 1", api.TripPurposeVacation)
	trip1ID := uuid.UUID(trip1.Id)
	trip2 := createTestTrip(t, svc, "Trip 2", api.TripPurposeBusiness)
	trip2ID := uuid.UUID(trip2.Id)

	// Create item on trip 1
	item, err := svc.CreateTripItem(trip1ID, &api.CreateItemRequest{Type: api.Flight})
	require.NoError(t, err)
	itemID := uuid.UUID(item.Id)

	// Move item to trip 2
	targetID := types.UUID(trip2ID)
	result, err := svc.MoveItem(itemID, &api.MoveItemRequest{
		TargetTripId: &targetID,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Item should now be on trip 2
	assert.Equal(t, trip2ID, uuid.UUID(result.Item.TripId))

	// Trip 1 should have no items
	items1, err := svc.ListTripItems(trip1ID)
	require.NoError(t, err)
	assert.Empty(t, items1)

	// Trip 2 should have the item
	items2, err := svc.ListTripItems(trip2ID)
	require.NoError(t, err)
	assert.Len(t, items2, 1)
}

func TestMoveItem_CreateNewTrip(t *testing.T) {
	svc := setupTestService(t)

	// Create trip with item
	trip := createTestTrip(t, svc, "Original Trip", api.TripPurposeVacation)
	tripID := uuid.UUID(trip.Id)
	item, err := svc.CreateTripItem(tripID, &api.CreateItemRequest{Type: api.Hotel})
	require.NoError(t, err)
	itemID := uuid.UUID(item.Id)

	// Move item to new trip
	result, err := svc.MoveItem(itemID, &api.MoveItemRequest{
		NewTrip: &api.CreateTripRequest{
			Name:    "New Trip",
			Purpose: api.TripPurposeBusiness,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Trip)

	// New trip should be created
	assert.Equal(t, "New Trip", result.Trip.Name)
	assert.Equal(t, api.TripPurposeBusiness, result.Trip.Purpose)

	// Item should be on new trip
	newTripID := uuid.UUID(result.Trip.Id)
	assert.Equal(t, newTripID, uuid.UUID(result.Item.TripId))

	// Original trip should have no items
	items, err := svc.ListTripItems(tripID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestMoveItem_SameTripError(t *testing.T) {
	svc := setupTestService(t)

	trip := createTestTrip(t, svc, "Trip", api.TripPurposeVacation)
	tripID := uuid.UUID(trip.Id)
	item, err := svc.CreateTripItem(tripID, &api.CreateItemRequest{Type: api.Flight})
	require.NoError(t, err)

	// Try to move item to same trip
	targetID := types.UUID(tripID)
	_, err = svc.MoveItem(uuid.UUID(item.Id), &api.MoveItemRequest{
		TargetTripId: &targetID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already on this trip")
}

func TestMoveItem_ItemNotFound(t *testing.T) {
	svc := setupTestService(t)

	trip := createTestTrip(t, svc, "Trip", api.TripPurposeVacation)
	targetID := types.UUID(uuid.UUID(trip.Id))

	// Try to move non-existent item
	result, err := svc.MoveItem(uuid.New(), &api.MoveItemRequest{
		TargetTripId: &targetID,
	})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMoveItem_TargetTripNotFound(t *testing.T) {
	svc := setupTestService(t)

	// Create trip with item
	trip := createTestTrip(t, svc, "Trip", api.TripPurposeVacation)
	item, err := svc.CreateTripItem(uuid.UUID(trip.Id), &api.CreateItemRequest{Type: api.Flight})
	require.NoError(t, err)

	// Try to move to non-existent trip
	nonExistentID := types.UUID(uuid.New())
	_, err = svc.MoveItem(uuid.UUID(item.Id), &api.MoveItemRequest{
		TargetTripId: &nonExistentID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMoveItem_MissingTarget(t *testing.T) {
	svc := setupTestService(t)

	trip := createTestTrip(t, svc, "Trip", api.TripPurposeVacation)
	item, err := svc.CreateTripItem(uuid.UUID(trip.Id), &api.CreateItemRequest{Type: api.Flight})
	require.NoError(t, err)

	// Try to move without providing target
	_, err = svc.MoveItem(uuid.UUID(item.Id), &api.MoveItemRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must provide targetTripId or newTrip")
}
