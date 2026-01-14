// Package handler provides HTTP handlers implementing the OpenAPI ServerInterface.
package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/api"
	"github.com/user/travel-calendar/backend/internal/service"
)

// Handler implements api.ServerInterface.
type Handler struct {
	svc *service.Service
}

// New creates a new Handler with the given service.
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// Ensure Handler implements ServerInterface
var _ api.ServerInterface = (*Handler)(nil)

// GetHealth returns the health status of the service.
func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, api.HealthResponse{
		Status:  api.Ok,
		Service: "backend",
	})
}

// ListTrips returns a list of trips.
func (h *Handler) ListTrips(w http.ResponseWriter, r *http.Request, params api.ListTripsParams) {
	trips, err := h.svc.ListTrips(params.Upcoming, params.Past, params.Purpose)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, trips)
}

// CreateTrip creates a new trip.
func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request) {
	var req api.CreateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	trip, err := h.svc.CreateTrip(&req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, trip)
}

// SearchTrips searches trips by query.
func (h *Handler) SearchTrips(w http.ResponseWriter, r *http.Request, params api.SearchTripsParams) {
	trips, err := h.svc.SearchTrips(params.Q)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, trips)
}

// GetTrip returns a single trip by ID.
func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.svc.GetTrip(uuid.UUID(tripId))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if trip == nil {
		respondError(w, http.StatusNotFound, "trip not found")
		return
	}
	respondJSON(w, http.StatusOK, trip)
}

// UpdateTrip updates an existing trip.
func (h *Handler) UpdateTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	var req api.UpdateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	trip, err := h.svc.UpdateTrip(uuid.UUID(tripId), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if trip == nil {
		respondError(w, http.StatusNotFound, "trip not found")
		return
	}
	respondJSON(w, http.StatusOK, trip)
}

// DeleteTrip deletes a trip by ID.
func (h *Handler) DeleteTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	err := h.svc.DeleteTrip(uuid.UUID(tripId))
	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "trip not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListTripItems returns items for a trip.
func (h *Handler) ListTripItems(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	items, err := h.svc.ListTripItems(uuid.UUID(tripId))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		respondError(w, http.StatusNotFound, "trip not found")
		return
	}
	respondJSON(w, http.StatusOK, items)
}

// CreateTripItem adds an item to a trip.
func (h *Handler) CreateTripItem(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	var req api.CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.svc.CreateTripItem(uuid.UUID(tripId), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		respondError(w, http.StatusNotFound, "trip not found")
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

// DeleteItem deletes an item by ID.
func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request, itemId api.ItemId) {
	err := h.svc.DeleteItem(uuid.UUID(itemId))
	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListDocuments returns documents.
func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request, params api.ListDocumentsParams) {
	var tripID *uuid.UUID
	if params.TripId != nil {
		id := uuid.UUID(*params.TripId)
		tripID = &id
	}

	docs, err := h.svc.ListDocuments(tripID, params.Unassociated)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, docs)
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, api.Error{Error: message})
}
