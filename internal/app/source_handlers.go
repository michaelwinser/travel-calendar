package app

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/michaelwinser/appbase"
	"github.com/michaelwinser/appbase/server"
)

// SetSourceStores adds source and staged event stores.
func (s *ActivityServer) SetSourceStores(sources *ImportSourceStore, staged *StagedEventStore) {
	s.importSources = sources
	s.stagedEvents = staged
}

// RegisterSourceRoutes adds source and staging endpoints.
func (s *ActivityServer) RegisterSourceRoutes(mux interface {
	Get(pattern string, handler http.HandlerFunc)
	Post(pattern string, handler http.HandlerFunc)
	Put(pattern string, handler http.HandlerFunc)
	Delete(pattern string, handler http.HandlerFunc)
}) {
	mux.Get("/api/sources", s.ListSources)
	mux.Post("/api/sources", s.CreateSource)
	mux.Get("/api/sources/{id}", s.GetSource)
	mux.Put("/api/sources/{id}", s.UpdateSource)
	mux.Delete("/api/sources/{id}", s.DeleteSource)
	mux.Post("/api/sources/{id}/sync", s.SyncSourceHandler)

	mux.Get("/api/staged", s.ListStagedEvents)
	mux.Post("/api/staged/import", s.ImportStagedEvents)
	mux.Post("/api/staged/hide", s.HideStagedEvents)
	mux.Post("/api/staged/unhide", s.UnhideStagedEvents)
}

// --- Source handlers ---

func (s *ActivityServer) ListSources(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	sources, err := s.importSources.List(userID)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Enrich with staged event counts
	type sourceWithCounts struct {
		ImportSource
		NewCount      int `json:"newCount"`
		ImportedCount int `json:"importedCount"`
		HiddenCount   int `json:"hiddenCount"`
	}

	result := make([]sourceWithCounts, len(sources))
	for i, src := range sources {
		result[i] = sourceWithCounts{ImportSource: src}
		events, _ := s.stagedEvents.ListBySource(src.ID)
		for _, e := range events {
			switch e.State {
			case "new":
				result[i].NewCount++
			case "imported":
				result[i].ImportedCount++
			case "hidden":
				result[i].HiddenCount++
			}
		}
	}

	server.RespondJSON(w, http.StatusOK, result)
}

func (s *ActivityServer) CreateSource(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req struct {
		Name         string `json:"name"`
		URL          string `json:"url"`
		SourceType   string `json:"sourceType"`
		FilterConfig string `json:"filterConfig"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.URL == "" {
		server.RespondError(w, http.StatusBadRequest, "name and url are required")
		return
	}
	if req.SourceType == "" {
		req.SourceType = "ical"
	}

	// Check for duplicate URL
	existing, _ := s.importSources.FindByURL(userID, req.URL)
	if existing != nil {
		server.RespondError(w, http.StatusBadRequest, "a source with this URL already exists")
		return
	}

	src, err := s.importSources.Create(userID, req.Name, req.URL, req.SourceType, req.FilterConfig)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Trigger initial sync
	result, syncErr := SyncSource(src, s.stagedEvents, s.store)
	s.importSources.Update(src) // save lastSyncAt

	resp := map[string]interface{}{
		"source": src,
	}
	if syncErr != nil {
		resp["syncError"] = syncErr.Error()
	} else {
		resp["syncResult"] = result
	}

	server.RespondJSON(w, http.StatusCreated, resp)
}

func (s *ActivityServer) GetSource(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	id := r.PathValue("id")
	src, err := s.importSources.Get(id)
	if err != nil || src == nil || src.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	server.RespondJSON(w, http.StatusOK, src)
}

func (s *ActivityServer) UpdateSource(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	id := r.PathValue("id")
	src, err := s.importSources.Get(id)
	if err != nil || src == nil || src.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	var req struct {
		Name         *string `json:"name"`
		FilterConfig *string `json:"filterConfig"`
		Status       *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		src.Name = *req.Name
	}
	if req.FilterConfig != nil {
		src.FilterConfig = *req.FilterConfig
	}
	if req.Status != nil {
		src.Status = *req.Status
	}

	if err := s.importSources.Update(src); err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusOK, src)
}

func (s *ActivityServer) DeleteSource(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	id := r.PathValue("id")
	src, err := s.importSources.Get(id)
	if err != nil || src == nil || src.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	// Check query param for whether to delete imported activities
	deleteActivities := r.URL.Query().Get("deleteActivities") == "true"

	// Delete imported activities if requested
	activitiesDeleted := 0
	staged, _ := s.stagedEvents.ListBySource(id)
	if deleteActivities {
		for _, e := range staged {
			if e.State == "imported" && e.ActivityID != "" {
				if err := s.store.Delete(e.ActivityID); err == nil {
					activitiesDeleted++
				}
			}
		}
	}

	// Delete staged events and source
	s.stagedEvents.DeleteBySource(id)
	s.importSources.Delete(id)

	server.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":                "true",
		"activitiesDeleted": activitiesDeleted,
	})
}

func (s *ActivityServer) SyncSourceHandler(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	id := r.PathValue("id")
	src, err := s.importSources.Get(id)
	if err != nil || src == nil || src.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	result, syncErr := SyncSource(src, s.stagedEvents, s.store)
	s.importSources.Update(src)

	if syncErr != nil {
		server.RespondError(w, http.StatusInternalServerError, "sync failed: "+syncErr.Error())
		return
	}

	server.RespondJSON(w, http.StatusOK, result)
}

// --- Staged event handlers ---

func (s *ActivityServer) ListStagedEvents(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	sourceID := r.URL.Query().Get("sourceId")
	state := r.URL.Query().Get("state")

	var events []StagedEvent
	var err error

	if sourceID != "" {
		events, err = s.stagedEvents.ListBySource(sourceID)
		// Filter by state if specified
		if state != "" {
			var filtered []StagedEvent
			for _, e := range events {
				if e.State == state {
					filtered = append(filtered, e)
				}
			}
			events = filtered
		}
	} else {
		events, err = s.stagedEvents.ListByUser(userID, state)
	}

	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusOK, events)
}

func (s *ActivityServer) ImportStagedEvents(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	imported := 0
	for _, id := range req.IDs {
		staged, err := s.stagedEvents.Get(id)
		if err != nil || staged == nil || staged.UserID != userID {
			continue
		}
		if staged.State == "imported" {
			continue // already imported
		}

		// Create activity from staged event
		endDate := staged.EndDate
		if endDate == "" {
			endDate = staged.StartDate
		}

		activity, err := s.store.Create(userID, staged.Title, staged.Type,
			staged.StartDate, endDate, staged.Location, staged.Notes, "", "", "", "")
		if err != nil {
			continue
		}

		// Resolve location to place
		if staged.Location != "" && s.places != nil {
			if place, _ := s.places.FindByName(userID, staged.Location); place != nil {
				activity.PlaceID = place.ID
				s.store.Update(activity)
			}
		}

		// Update staged event state
		staged.State = "imported"
		staged.ActivityID = activity.ID
		s.stagedEvents.Update(staged)
		imported++
	}

	server.RespondJSON(w, http.StatusOK, map[string]int{"imported": imported})
}

func (s *ActivityServer) HideStagedEvents(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	hidden := 0
	for _, id := range req.IDs {
		staged, err := s.stagedEvents.Get(id)
		if err != nil || staged == nil || staged.UserID != userID {
			continue
		}
		if staged.State != "new" {
			continue
		}
		staged.State = "hidden"
		s.stagedEvents.Update(staged)
		hidden++
	}

	server.RespondJSON(w, http.StatusOK, map[string]int{"hidden": hidden})
}

func (s *ActivityServer) UnhideStagedEvents(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	unhidden := 0
	for _, id := range req.IDs {
		staged, err := s.stagedEvents.Get(id)
		if err != nil || staged == nil || staged.UserID != userID {
			continue
		}
		if staged.State != "hidden" {
			continue
		}
		staged.State = "new"
		s.stagedEvents.Update(staged)
		unhidden++
	}

	server.RespondJSON(w, http.StatusOK, map[string]int{"unhidden": unhidden})
}

// ensure appbase import is used
var _ = appbase.UserID
var _ = strings.Contains
