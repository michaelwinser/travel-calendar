package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/travel-calendar/backend/internal/api"
	"github.com/user/travel-calendar/backend/internal/service"
	"github.com/user/travel-calendar/backend/internal/store"
)

// setupTestHandler creates a handler backed by in-memory SQLite for testing.
func setupTestHandler(t *testing.T) *Handler {
	s, err := store.NewSQLite(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	svc := service.New(s)
	return New(svc, s)
}

// Helper to create a test trip via the handler
func createTripViaHandler(t *testing.T, h *Handler, name string, purpose api.TripPurpose) api.Trip {
	body := api.CreateTripRequest{
		Name:    name,
		Purpose: purpose,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/trips", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateTrip(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var trip api.Trip
	err := json.Unmarshal(rec.Body.Bytes(), &trip)
	require.NoError(t, err)
	return trip
}

// =============================================================================
// Health Endpoint Tests
// =============================================================================

func TestGetHealth_ReturnsOK(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	h.GetHealth(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp api.HealthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, api.Ok, resp.Status)
	assert.Equal(t, "backend", resp.Service)
}

// =============================================================================
// Trip Endpoint Tests
// =============================================================================

func TestListTrips_Empty(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/trips", nil)
	rec := httptest.NewRecorder()

	h.ListTrips(rec, req, api.ListTripsParams{})

	assert.Equal(t, http.StatusOK, rec.Code)

	var trips []api.Trip
	err := json.Unmarshal(rec.Body.Bytes(), &trips)
	require.NoError(t, err)
	assert.Empty(t, trips)
}

func TestListTrips_ReturnsTrips(t *testing.T) {
	h := setupTestHandler(t)

	createTripViaHandler(t, h, "Trip 1", api.TripPurposeVacation)
	createTripViaHandler(t, h, "Trip 2", api.TripPurposeBusiness)

	req := httptest.NewRequest(http.MethodGet, "/api/trips", nil)
	rec := httptest.NewRecorder()

	h.ListTrips(rec, req, api.ListTripsParams{})

	assert.Equal(t, http.StatusOK, rec.Code)

	var trips []api.Trip
	err := json.Unmarshal(rec.Body.Bytes(), &trips)
	require.NoError(t, err)
	assert.Len(t, trips, 2)
}

func TestCreateTrip_Success(t *testing.T) {
	h := setupTestHandler(t)

	body := api.CreateTripRequest{
		Name:    "Test Trip",
		Purpose: api.TripPurposeVacation,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/trips", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateTrip(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var trip api.Trip
	err := json.Unmarshal(rec.Body.Bytes(), &trip)
	require.NoError(t, err)
	assert.Equal(t, "Test Trip", trip.Name)
	assert.Equal(t, api.TripPurposeVacation, trip.Purpose)
	assert.Equal(t, api.Planning, trip.Status)
}

func TestCreateTrip_InvalidJSON(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/trips", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateTrip(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp api.Error
	err := json.Unmarshal(rec.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", errResp.Error)
}

func TestGetTrip_Found(t *testing.T) {
	h := setupTestHandler(t)

	created := createTripViaHandler(t, h, "Test Trip", api.TripPurposeVacation)

	req := httptest.NewRequest(http.MethodGet, "/api/trips/"+created.Id.String(), nil)
	rec := httptest.NewRecorder()

	h.GetTrip(rec, req, types.UUID(created.Id))

	assert.Equal(t, http.StatusOK, rec.Code)

	var trip api.Trip
	err := json.Unmarshal(rec.Body.Bytes(), &trip)
	require.NoError(t, err)
	assert.Equal(t, "Test Trip", trip.Name)
}

func TestGetTrip_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/trips/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	h.GetTrip(rec, req, types.UUID(uuid.New()))

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var errResp api.Error
	err := json.Unmarshal(rec.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "trip not found", errResp.Error)
}

func TestUpdateTrip_Success(t *testing.T) {
	h := setupTestHandler(t)

	created := createTripViaHandler(t, h, "Original Name", api.TripPurposeVacation)

	newName := "Updated Name"
	body := api.UpdateTripRequest{
		Name: &newName,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/trips/"+created.Id.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.UpdateTrip(rec, req, types.UUID(created.Id))

	assert.Equal(t, http.StatusOK, rec.Code)

	var trip api.Trip
	err := json.Unmarshal(rec.Body.Bytes(), &trip)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", trip.Name)
}

func TestUpdateTrip_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	newName := "Updated"
	body := api.UpdateTripRequest{
		Name: &newName,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/trips/"+uuid.New().String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.UpdateTrip(rec, req, types.UUID(uuid.New()))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateTrip_InvalidJSON(t *testing.T) {
	h := setupTestHandler(t)

	created := createTripViaHandler(t, h, "Test", api.TripPurposeVacation)

	req := httptest.NewRequest(http.MethodPatch, "/api/trips/"+created.Id.String(), bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.UpdateTrip(rec, req, types.UUID(created.Id))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteTrip_Success(t *testing.T) {
	h := setupTestHandler(t)

	created := createTripViaHandler(t, h, "To Delete", api.TripPurposeVacation)

	req := httptest.NewRequest(http.MethodDelete, "/api/trips/"+created.Id.String(), nil)
	rec := httptest.NewRecorder()

	h.DeleteTrip(rec, req, types.UUID(created.Id))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDeleteTrip_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/trips/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	h.DeleteTrip(rec, req, types.UUID(uuid.New()))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSearchTrips_ReturnsMatches(t *testing.T) {
	h := setupTestHandler(t)

	createTripViaHandler(t, h, "FOSDEM Conference", api.TripPurposeConference)
	createTripViaHandler(t, h, "Beach Vacation", api.TripPurposeVacation)

	req := httptest.NewRequest(http.MethodGet, "/api/trips/search?q=FOSDEM", nil)
	rec := httptest.NewRecorder()

	h.SearchTrips(rec, req, api.SearchTripsParams{Q: "FOSDEM"})

	assert.Equal(t, http.StatusOK, rec.Code)

	var trips []api.Trip
	err := json.Unmarshal(rec.Body.Bytes(), &trips)
	require.NoError(t, err)
	assert.Len(t, trips, 1)
	assert.Equal(t, "FOSDEM Conference", trips[0].Name)
}

// =============================================================================
// Item Endpoint Tests
// =============================================================================

func TestListTripItems_Empty(t *testing.T) {
	h := setupTestHandler(t)

	trip := createTripViaHandler(t, h, "Trip", api.TripPurposeVacation)

	req := httptest.NewRequest(http.MethodGet, "/api/trips/"+trip.Id.String()+"/items", nil)
	rec := httptest.NewRecorder()

	h.ListTripItems(rec, req, types.UUID(trip.Id))

	assert.Equal(t, http.StatusOK, rec.Code)

	var items []api.Item
	err := json.Unmarshal(rec.Body.Bytes(), &items)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestListTripItems_TripNotFound(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/trips/"+uuid.New().String()+"/items", nil)
	rec := httptest.NewRecorder()

	h.ListTripItems(rec, req, types.UUID(uuid.New()))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCreateTripItem_Success(t *testing.T) {
	h := setupTestHandler(t)

	trip := createTripViaHandler(t, h, "Trip", api.TripPurposeVacation)

	from := "JFK"
	to := "LAX"
	body := api.CreateItemRequest{
		Type: api.Flight,
		From: &from,
		To:   &to,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/trips/"+trip.Id.String()+"/items", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateTripItem(rec, req, types.UUID(trip.Id))

	assert.Equal(t, http.StatusCreated, rec.Code)

	var item api.Item
	err := json.Unmarshal(rec.Body.Bytes(), &item)
	require.NoError(t, err)
	assert.Equal(t, api.Flight, item.Type)
	assert.Equal(t, "JFK", *item.From)
	assert.Equal(t, "LAX", *item.To)
}

func TestCreateTripItem_TripNotFound(t *testing.T) {
	h := setupTestHandler(t)

	body := api.CreateItemRequest{
		Type: api.Flight,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/trips/"+uuid.New().String()+"/items", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateTripItem(rec, req, types.UUID(uuid.New()))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCreateTripItem_InvalidJSON(t *testing.T) {
	h := setupTestHandler(t)

	trip := createTripViaHandler(t, h, "Trip", api.TripPurposeVacation)

	req := httptest.NewRequest(http.MethodPost, "/api/trips/"+trip.Id.String()+"/items", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateTripItem(rec, req, types.UUID(trip.Id))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteItem_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/items/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	h.DeleteItem(rec, req, types.UUID(uuid.New()))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// =============================================================================
// Document Endpoint Tests
// =============================================================================

func TestListDocuments_Empty(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/documents", nil)
	rec := httptest.NewRecorder()

	h.ListDocuments(rec, req, api.ListDocumentsParams{})

	assert.Equal(t, http.StatusOK, rec.Code)

	var docs []api.Document
	err := json.Unmarshal(rec.Body.Bytes(), &docs)
	require.NoError(t, err)
	assert.Empty(t, docs)
}
