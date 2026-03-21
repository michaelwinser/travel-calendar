// Package handler provides HTTP handlers implementing the OpenAPI ServerInterface.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/user/travel-calendar/backend/internal/api"
	"github.com/user/travel-calendar/backend/internal/auth"
	"github.com/user/travel-calendar/backend/internal/service"
	"github.com/user/travel-calendar/backend/internal/store"
)

// Handler implements api.ServerInterface.
type Handler struct {
	svc      *service.Service
	calendar *service.CalendarService
	store    store.StoreInterface
}

// New creates a new Handler with the given service.
func New(svc *service.Service, s store.StoreInterface) *Handler {
	return &Handler{svc: svc, store: s}
}

// SetCalendarService sets the calendar service for OAuth operations.
func (h *Handler) SetCalendarService(cal *service.CalendarService) {
	h.calendar = cal
}

// Ensure Handler implements ServerInterface
var _ api.ServerInterface = (*Handler)(nil)

// userID extracts the authenticated user ID from the request context.
func userID(r *http.Request) string {
	return auth.UserIDFromContext(r.Context())
}

// allowedUsers returns the ALLOWED_USERS list from env, or nil if unset.
func allowedUsers() []string {
	val := os.Getenv("ALLOWED_USERS")
	if val == "" {
		return nil
	}
	var users []string
	for _, u := range strings.Split(val, ",") {
		u = strings.TrimSpace(u)
		if u != "" {
			users = append(users, u)
		}
	}
	return users
}

// GetHealth returns the health status of the service.
func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, api.HealthResponse{
		Status:  api.Ok,
		Service: "backend",
	})
}

// ListTrips returns a list of trips.
func (h *Handler) ListTrips(w http.ResponseWriter, r *http.Request, params api.ListTripsParams) {
	trips, err := h.svc.ListTrips(userID(r), params.Upcoming, params.Past, params.Purpose)
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

	trip, err := h.svc.CreateTrip(userID(r), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, trip)
}

// SearchTrips searches trips by query.
func (h *Handler) SearchTrips(w http.ResponseWriter, r *http.Request, params api.SearchTripsParams) {
	trips, err := h.svc.SearchTrips(userID(r), params.Q)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, trips)
}

// GetTrip returns a single trip by ID.
func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.svc.GetTrip(userID(r), uuid.UUID(tripId))
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

	trip, err := h.svc.UpdateTrip(userID(r), uuid.UUID(tripId), &req)
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
	err := h.svc.DeleteTrip(userID(r), uuid.UUID(tripId))
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusNotFound, "trip not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MergeTrips merges one trip into another.
func (h *Handler) MergeTrips(w http.ResponseWriter, r *http.Request, sourceId openapi_types.UUID, targetId openapi_types.UUID) {
	var req api.MergeTripsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate not same trip
	if sourceId == targetId {
		respondError(w, http.StatusBadRequest, "cannot merge trip into itself")
		return
	}

	trip, err := h.svc.MergeTrips(userID(r), uuid.UUID(sourceId), uuid.UUID(targetId), &req)
	if err != nil {
		if strings.Contains(err.Error(), "cannot merge trip into itself") {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if trip == nil {
		respondError(w, http.StatusNotFound, "source or target trip not found")
		return
	}
	respondJSON(w, http.StatusOK, trip)
}

// MoveItem moves an item to another trip.
func (h *Handler) MoveItem(w http.ResponseWriter, r *http.Request, itemId api.ItemId) {
	var req api.MoveItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request has either targetTripId or newTrip
	if req.TargetTripId == nil && req.NewTrip == nil {
		respondError(w, http.StatusBadRequest, "must provide targetTripId or newTrip")
		return
	}

	result, err := h.svc.MoveItem(userID(r), uuid.UUID(itemId), &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "already on this trip") {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if result == nil {
		respondError(w, http.StatusNotFound, "item not found")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// ListTripItems returns items for a trip.
func (h *Handler) ListTripItems(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	items, err := h.svc.ListTripItems(userID(r), uuid.UUID(tripId))
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

	item, err := h.svc.CreateTripItem(userID(r), uuid.UUID(tripId), &req)
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
	if errors.Is(err, store.ErrNotFound) {
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

// Config endpoints

// GetBaseLocations returns the user's base locations.
func (h *Handler) GetBaseLocations(w http.ResponseWriter, r *http.Request) {
	locations, err := h.svc.GetBaseLocations()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, locations)
}

// SetBaseLocations updates the user's base locations.
func (h *Handler) SetBaseLocations(w http.ResponseWriter, r *http.Request) {
	var req api.SetBaseLocationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	locations, err := h.svc.SetBaseLocations(&req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, locations)
}

// Trip Location endpoints

// GetTripLocations returns locations for a trip.
func (h *Handler) GetTripLocations(w http.ResponseWriter, r *http.Request, tripId openapi_types.UUID) {
	locations, err := h.svc.GetTripLocations(userID(r), uuid.UUID(tripId))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if locations == nil {
		respondError(w, http.StatusNotFound, "trip not found")
		return
	}
	respondJSON(w, http.StatusOK, locations)
}

// SetTripLocations sets locations for a trip.
func (h *Handler) SetTripLocations(w http.ResponseWriter, r *http.Request, tripId openapi_types.UUID) {
	var req api.SetTripLocationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	locations, err := h.svc.SetTripLocations(userID(r), uuid.UUID(tripId), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if locations == nil {
		respondError(w, http.StatusNotFound, "trip not found")
		return
	}
	respondJSON(w, http.StatusOK, locations)
}

// Location Query endpoints

// GetLocationOnDate returns the user's location on a specific date.
func (h *Handler) GetLocationOnDate(w http.ResponseWriter, r *http.Request, date openapi_types.Date) {
	location, err := h.svc.GetLocationOnDate(userID(r), date.Time)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, location)
}

// GetLocationRange returns the user's locations for a date range.
func (h *Handler) GetLocationRange(w http.ResponseWriter, r *http.Request, params api.GetLocationRangeParams) {
	from := params.From.Time
	to := params.To.Time

	if to.Before(from) {
		respondError(w, http.StatusBadRequest, "'to' date must be after 'from' date")
		return
	}

	segments, err := h.svc.GetLocationRange(userID(r), from, to)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, segments)
}

// Google Calendar OAuth / Auth endpoints

// GetGoogleAuthUrl returns the OAuth URL to initiate login.
func (h *Handler) GetGoogleAuthUrl(w http.ResponseWriter, r *http.Request, params api.GetGoogleAuthUrlParams) {
	if h.calendar == nil || !h.calendar.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar integration not configured")
		return
	}

	scopes := ""
	if params.Scopes != nil {
		scopes = *params.Scopes
	}

	result, err := h.calendar.GetAuthURL(scopes)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// HandleGoogleCallback handles the OAuth callback — creates a session and sets a cookie.
func (h *Handler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request, params api.HandleGoogleCallbackParams) {
	if h.calendar == nil || !h.calendar.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar integration not configured")
		return
	}

	// Check for error from Google
	if params.Error != nil && *params.Error != "" {
		respondError(w, http.StatusBadRequest, "Google authorization failed: "+*params.Error)
		return
	}

	result, err := h.calendar.HandleCallback(r.Context(), params.Code, allowedUsers())
	if err != nil {
		if strings.Contains(err.Error(), "not authorized") {
			respondError(w, http.StatusForbidden, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "travel_session",
		Value:    result.SessionID,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})

	respondJSON(w, http.StatusOK, result.Status)
}

// DisconnectGoogle revokes Google Calendar access.
func (h *Handler) DisconnectGoogle(w http.ResponseWriter, r *http.Request) {
	if h.calendar == nil {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar integration not configured")
		return
	}

	uid := userID(r)
	if err := h.calendar.Disconnect(r.Context(), uid); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetGoogleAuthStatus returns the current auth status.
// When called without a session (exempt path), returns login status.
// When called with a session, returns calendar connection status.
func (h *Handler) GetGoogleAuthStatus(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	if uid == "" {
		// No session — return not logged in
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"connected": false,
			"loggedIn":  false,
		})
		return
	}

	if h.calendar == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"connected": false,
			"loggedIn":  true,
			"email":     auth.EmailFromContext(r.Context()),
		})
		return
	}

	status, err := h.calendar.GetAuthStatus(r.Context(), uid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"connected": status.Connected,
		"loggedIn":  true,
		"email":     auth.EmailFromContext(r.Context()),
		"scopes":    status.Scopes,
		"expiresAt": status.ExpiresAt,
	})
}

// Calendar endpoints

// ListCalendars returns available Google Calendars.
func (h *Handler) ListCalendars(w http.ResponseWriter, r *http.Request) {
	if h.calendar == nil || !h.calendar.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar integration not configured")
		return
	}

	calendars, err := h.calendar.ListCalendars(r.Context(), userID(r))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, calendars)
}

// GetSelectedCalendars returns the user's selected calendars.
func (h *Handler) GetSelectedCalendars(w http.ResponseWriter, r *http.Request) {
	if h.calendar == nil {
		respondJSON(w, http.StatusOK, []api.UserCalendar{})
		return
	}

	calendars, err := h.calendar.GetSelectedCalendars(r.Context(), userID(r))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, calendars)
}

// SetSelectedCalendars updates the user's selected calendars.
func (h *Handler) SetSelectedCalendars(w http.ResponseWriter, r *http.Request) {
	if h.calendar == nil || !h.calendar.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar integration not configured")
		return
	}

	var req api.SetSelectedCalendarsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	calendars, err := h.calendar.SetSelectedCalendars(r.Context(), userID(r), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, calendars)
}

// ListCalendarEvents returns calendar events for a date range.
func (h *Handler) ListCalendarEvents(w http.ResponseWriter, r *http.Request, params api.ListCalendarEventsParams) {
	if h.calendar == nil || !h.calendar.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not configured")
		return
	}

	from := time.Time(params.From.Time)
	to := time.Time(params.To.Time)

	events, err := h.calendar.ListCalendarEvents(r.Context(), userID(r), from, to, params.CalendarId)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, events)
}

// GetCalendarConflicts detects conflicts between calendar events and trips.
func (h *Handler) GetCalendarConflicts(w http.ResponseWriter, r *http.Request, params api.GetCalendarConflictsParams) {
	// TODO: Implement when CalendarService is ready
	respondJSON(w, http.StatusOK, []api.CalendarConflict{})
}

// SuggestTripsFromCalendar suggests trips based on calendar events.
func (h *Handler) SuggestTripsFromCalendar(w http.ResponseWriter, r *http.Request, params api.SuggestTripsFromCalendarParams) {
	if h.calendar == nil || !h.calendar.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not configured")
		return
	}

	from := time.Now()
	to := from.AddDate(0, 0, 90)
	if params.From != nil {
		from = time.Time(params.From.Time)
	}
	if params.To != nil {
		to = time.Time(params.To.Time)
	}

	suggestions, err := h.calendar.SuggestTripsFromCalendar(r.Context(), userID(r), from, to)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, suggestions)
}

// ImportTripSuggestion imports a trip suggestion as a new trip.
func (h *Handler) ImportTripSuggestion(w http.ResponseWriter, r *http.Request, suggestionId string) {
	if h.calendar == nil || !h.calendar.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not configured")
		return
	}

	from := time.Now()
	to := from.AddDate(0, 0, 90)

	trip, err := h.calendar.ImportTripSuggestion(r.Context(), h.svc, userID(r), suggestionId, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, trip)
}

// DismissTripSuggestion marks a suggestion as dismissed so it won't appear again.
func (h *Handler) DismissTripSuggestion(w http.ResponseWriter, r *http.Request, suggestionId string) {
	if h.calendar == nil || !h.calendar.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not configured")
		return
	}

	from := time.Now()
	to := from.AddDate(0, 0, 90)

	err := h.calendar.DismissTripSuggestion(r.Context(), userID(r), suggestionId, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MergeTripSuggestion merges a suggestion into an existing trip.
func (h *Handler) MergeTripSuggestion(w http.ResponseWriter, r *http.Request, suggestionId string, tripId api.TripId) {
	if h.calendar == nil || !h.calendar.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not configured")
		return
	}

	from := time.Now()
	to := from.AddDate(0, 0, 90)

	trip, err := h.calendar.MergeTripSuggestion(r.Context(), h.svc, userID(r), suggestionId, uuid.UUID(tripId), from, to)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, trip)
}

// ResetProcessedEvents clears all processed event records.
func (h *Handler) ResetProcessedEvents(w http.ResponseWriter, r *http.Request) {
	if h.calendar == nil || !h.calendar.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Google Calendar not configured")
		return
	}

	err := h.calendar.ResetProcessedEvents(userID(r))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Day Entry endpoints

// ListDayEntries returns day entries for a date range.
func (h *Handler) ListDayEntries(w http.ResponseWriter, r *http.Request, params api.ListDayEntriesParams) {
	from := params.From.Time
	to := params.To.Time

	entries, err := h.svc.ListDayEntries(userID(r), from, to)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, entries)
}

// CreateDayEntry creates a new day entry.
func (h *Handler) CreateDayEntry(w http.ResponseWriter, r *http.Request) {
	var req api.CreateDayEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	entry, err := h.svc.CreateDayEntry(userID(r), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, entry)
}

// UpdateDayEntry updates an existing day entry.
func (h *Handler) UpdateDayEntry(w http.ResponseWriter, r *http.Request, dayEntryId api.DayEntryId) {
	var req api.UpdateDayEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	entry, err := h.svc.UpdateDayEntry(userID(r), uuid.UUID(dayEntryId), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entry == nil {
		respondError(w, http.StatusNotFound, "day entry not found")
		return
	}
	respondJSON(w, http.StatusOK, entry)
}

// DeleteDayEntry deletes a day entry.
func (h *Handler) DeleteDayEntry(w http.ResponseWriter, r *http.Request, dayEntryId api.DayEntryId) {
	err := h.svc.DeleteDayEntry(userID(r), uuid.UUID(dayEntryId))
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusNotFound, "day entry not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Calendar sync endpoints

// GetTripCalendarLinks returns calendar links for a trip.
func (h *Handler) GetTripCalendarLinks(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	// TODO: Implement when CalendarService is ready
	respondJSON(w, http.StatusOK, []api.CalendarLink{})
}

// SyncTripToCalendar syncs a trip to Google Calendar.
func (h *Handler) SyncTripToCalendar(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	// TODO: Implement when CalendarService is ready
	respondError(w, http.StatusNotImplemented, "Google Calendar integration not yet implemented")
}
