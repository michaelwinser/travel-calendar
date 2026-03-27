package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/michaelwinser/appbase"
	"github.com/michaelwinser/appbase/server"
)

// SyncConnect creates a dedicated "Travel Calendar" on Google Calendar
// and stores it as a sync target.
func (s *ActivityServer) SyncConnect(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	token := appbase.AccessToken(r)
	if token == "" {
		server.RespondError(w, http.StatusBadRequest, "no OAuth token available — login with Google first")
		return
	}

	// Check if already connected
	existing, _ := s.syncTargets.ListByUser(userID)
	for _, t := range existing {
		if t.Type == "google_calendar" && t.Status == "active" {
			server.RespondJSON(w, http.StatusOK, map[string]string{
				"status":     "already_connected",
				"calendarId": t.CalendarID,
				"message":    "Google Calendar sync is already connected",
			})
			return
		}
	}

	// Create Calendar API client
	ctx := r.Context()
	gcal, err := NewGCalClient(ctx, token)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, "failed to create calendar client: "+err.Error())
		return
	}

	// Create a dedicated calendar
	calendarID, err := gcal.CreateCalendar(ctx, "Travel Calendar")
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, "failed to create calendar: "+err.Error())
		return
	}

	// Save sync target
	target := &SyncTarget{
		ID:           uuid.New().String(),
		UserID:       userID,
		Type:         "google_calendar",
		CalendarID:   calendarID,
		CalendarName: "Travel Calendar",
		Status:       "active",
		CreatedAt:    nowRFC3339(),
	}
	if err := s.syncTargets.Create(target); err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusCreated, map[string]string{
		"status":     "connected",
		"calendarId": calendarID,
		"targetId":   target.ID,
		"message":    "Created 'Travel Calendar' on Google Calendar",
	})
}

// SyncPush pushes trips and eligible activities to the connected Google Calendar.
func (s *ActivityServer) SyncPush(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	token := appbase.AccessToken(r)
	if token == "" {
		server.RespondError(w, http.StatusBadRequest, "no OAuth token available")
		return
	}

	// Find active sync target
	targets, _ := s.syncTargets.ListByUser(userID)
	var target *SyncTarget
	for _, t := range targets {
		if t.Type == "google_calendar" && t.Status == "active" {
			target = &t
			break
		}
	}
	if target == nil {
		server.RespondError(w, http.StatusBadRequest, "no connected Google Calendar — run sync connect first")
		return
	}

	ctx := r.Context()
	gcal, err := NewGCalClient(ctx, token)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get existing sync records
	existingRecords, _ := s.syncRecords.ListByTarget(target.ID)
	recordByEntity := map[string]*SyncRecord{} // "type/id" → record
	for i := range existingRecords {
		r := &existingRecords[i]
		recordByEntity[r.EntityType+"/"+r.EntityID] = r
	}

	// Time window: -1 month to +6 months
	now := time.Now()
	from := now.AddDate(0, -1, 0).Format("2006-01-02")
	to := now.AddDate(0, 6, 0).Format("2006-01-02")

	// Fetch trips and their activities
	trips, _ := s.trips.List(userID)
	activities, _ := s.store.ListRange(userID, from, to)

	created, updated, skipped := 0, 0, 0

	// Sync trips
	tripActivities := map[string][]Activity{}
	for _, a := range activities {
		if a.TripID != "" {
			tripActivities[a.TripID] = append(tripActivities[a.TripID], a)
		}
	}

	for _, trip := range trips {
		acts := tripActivities[trip.ID]
		if len(acts) == 0 {
			continue
		}

		// Check if any activities fall in the time window
		inWindow := false
		for _, a := range acts {
			if a.EndDate >= from && a.StartDate <= to {
				inWindow = true
				break
			}
		}
		if !inWindow {
			continue
		}

		event := TripToEvent(trip, acts)
		hash := SyncHash(trip.Name, event.Start.Date, event.End.Date, event.Location)

		key := "trip/" + trip.ID
		if rec, exists := recordByEntity[key]; exists {
			if rec.SyncHash == hash {
				skipped++
				continue
			}
			// Update existing event
			event.Id = rec.CalendarEventID
			_, err := gcal.UpsertEvent(ctx, target.CalendarID, event)
			if err != nil {
				fmt.Printf("  Failed to update trip %q: %v\n", trip.Name, err)
				continue
			}
			rec.SyncHash = hash
			rec.LastSyncAt = nowRFC3339()
			s.syncRecords.Update(rec)
			updated++
		} else {
			// Create new event
			created_event, err := gcal.UpsertEvent(ctx, target.CalendarID, event)
			if err != nil {
				fmt.Printf("  Failed to create trip %q: %v\n", trip.Name, err)
				continue
			}
			s.syncRecords.Create(&SyncRecord{
				UserID:          userID,
				TargetID:        target.ID,
				EntityType:      "trip",
				EntityID:        trip.ID,
				EntityKey:       trip.Key,
				CalendarEventID: created_event.Id,
				SyncHash:        hash,
				LastSyncAt:      nowRFC3339(),
			})
			created++
		}
	}

	// Sync standalone activities (source=manual only, not already in a trip)
	for _, a := range activities {
		if a.TripID != "" {
			continue // handled as part of trip
		}
		if a.Source != "manual" {
			continue // don't export imported activities
		}

		event := ActivityToEvent(a)
		hash := SyncHash(a.Title, a.StartDate, a.EndDate, a.Location)

		key := "activity/" + a.ID
		if rec, exists := recordByEntity[key]; exists {
			if rec.SyncHash == hash {
				skipped++
				continue
			}
			event.Id = rec.CalendarEventID
			_, err := gcal.UpsertEvent(ctx, target.CalendarID, event)
			if err != nil {
				continue
			}
			rec.SyncHash = hash
			rec.LastSyncAt = nowRFC3339()
			s.syncRecords.Update(rec)
			updated++
		} else {
			created_event, err := gcal.UpsertEvent(ctx, target.CalendarID, event)
			if err != nil {
				continue
			}
			s.syncRecords.Create(&SyncRecord{
				UserID:          userID,
				TargetID:        target.ID,
				EntityType:      "activity",
				EntityID:        a.ID,
				EntityKey:       a.Key,
				CalendarEventID: created_event.Id,
				SyncHash:        hash,
				LastSyncAt:      nowRFC3339(),
			})
			created++
		}
	}

	// Update last sync time
	target.LastSyncAt = nowRFC3339()
	s.syncTargets.Update(target)

	server.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"created": created,
		"updated": updated,
		"skipped": skipped,
	})
}

// SyncCleanup deletes all app-created events from the connected Google Calendar.
func (s *ActivityServer) SyncCleanup(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	token := appbase.AccessToken(r)
	if token == "" {
		server.RespondError(w, http.StatusBadRequest, "no OAuth token available")
		return
	}

	targets, _ := s.syncTargets.ListByUser(userID)
	var target *SyncTarget
	for _, t := range targets {
		if t.Type == "google_calendar" && t.Status == "active" {
			target = &t
			break
		}
	}
	if target == nil {
		server.RespondError(w, http.StatusBadRequest, "no connected Google Calendar")
		return
	}

	ctx := r.Context()
	gcal, err := NewGCalClient(ctx, token)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Delete all synced events
	records, _ := s.syncRecords.ListByTarget(target.ID)
	deleted := 0
	for _, rec := range records {
		if err := gcal.DeleteEvent(ctx, target.CalendarID, rec.CalendarEventID); err == nil {
			deleted++
		}
		s.syncRecords.Delete(rec.ID)
	}

	server.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"deleted": deleted,
		"message": fmt.Sprintf("Deleted %d events from Google Calendar", deleted),
	})
}

// SyncDisconnect removes the sync target and optionally cleans up events.
func (s *ActivityServer) SyncDisconnect(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req struct {
		Cleanup bool `json:"cleanup"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	targets, _ := s.syncTargets.ListByUser(userID)
	var target *SyncTarget
	for _, t := range targets {
		if t.Type == "google_calendar" {
			target = &t
			break
		}
	}
	if target == nil {
		server.RespondError(w, http.StatusNotFound, "no Google Calendar sync configured")
		return
	}

	// Optionally delete the calendar entirely
	if req.Cleanup {
		token := appbase.AccessToken(r)
		if token != "" {
			ctx := r.Context()
			if gcal, err := NewGCalClient(ctx, token); err == nil {
				gcal.DeleteCalendar(ctx, target.CalendarID)
			}
		}
	}

	// Delete sync records and target
	s.syncRecords.DeleteByTarget(target.ID)
	s.syncTargets.Delete(target.ID)

	server.RespondJSON(w, http.StatusOK, map[string]string{
		"status":  "disconnected",
		"message": "Google Calendar sync removed",
	})
}

// SyncStatus returns the current sync status.
func (s *ActivityServer) SyncStatus(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	targets, _ := s.syncTargets.ListByUser(userID)

	type targetStatus struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		CalendarName string `json:"calendarName"`
		CalendarID   string `json:"calendarId"`
		Status       string `json:"status"`
		LastSyncAt   string `json:"lastSyncAt"`
		SyncedCount  int    `json:"syncedCount"`
	}

	var result []targetStatus
	for _, t := range targets {
		records, _ := s.syncRecords.ListByTarget(t.ID)
		result = append(result, targetStatus{
			ID:           t.ID,
			Type:         t.Type,
			CalendarName: t.CalendarName,
			CalendarID:   t.CalendarID,
			Status:       t.Status,
			LastSyncAt:   t.LastSyncAt,
			SyncedCount:  len(records),
		})
	}

	server.RespondJSON(w, http.StatusOK, result)
}

// RegisterSyncRoutes adds the sync endpoints to the router.
// These are registered manually (not via codegen) since they're operational endpoints.
func (s *ActivityServer) RegisterSyncRoutes(mux interface {
	Post(pattern string, handler http.HandlerFunc)
	Get(pattern string, handler http.HandlerFunc)
	Delete(pattern string, handler http.HandlerFunc)
}) {
	mux.Post("/api/sync/connect", s.SyncConnect)
	mux.Post("/api/sync/push", s.SyncPush)
	mux.Post("/api/sync/cleanup", s.SyncCleanup)
	mux.Post("/api/sync/disconnect", s.SyncDisconnect)
	mux.Get("/api/sync/status", s.SyncStatus)
}
