package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/michaelwinser/appbase"
	"github.com/michaelwinser/appbase/server"
	"github.com/michaelwinser/travel-calendar/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Ensure ActivityServer implements the generated interface.
var _ api.ServerInterface = (*ActivityServer)(nil)

// ActivityServer implements the generated ServerInterface.
type ActivityServer struct {
	store *ActivityStore
}

func (s *ActivityServer) ListActivities(w http.ResponseWriter, r *http.Request, params api.ListActivitiesParams) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var items []Activity
	var err error

	if params.Month != nil {
		t, perr := time.Parse("2006-01", *params.Month)
		if perr != nil {
			server.RespondError(w, http.StatusBadRequest, "invalid month (expected YYYY-MM)")
			return
		}
		from := t.Format("2006-01-02")
		to := t.AddDate(0, 1, -1).Format("2006-01-02")
		items, err = s.store.ListRange(userID, from, to)
	} else if params.From != nil && params.To != nil {
		items, err = s.store.ListRange(userID, params.From.Format("2006-01-02"), params.To.Format("2006-01-02"))
	} else {
		items, err = s.store.List(userID)
	}

	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]api.Activity, len(items))
	for i, a := range items {
		result[i] = entityToAPI(a)
	}
	server.RespondJSON(w, http.StatusOK, result)
}

func (s *ActivityServer) CreateActivity(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req api.CreateActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		server.RespondError(w, http.StatusBadRequest, "title is required")
		return
	}

	startDate := req.StartDate.Format("2006-01-02")
	endDate := startDate
	if req.EndDate != nil {
		endDate = req.EndDate.Format("2006-01-02")
	}

	location := ""
	if req.Location != nil {
		location = *req.Location
	}
	notes := ""
	if req.Notes != nil {
		notes = *req.Notes
	}

	a, err := s.store.Create(userID, req.Title, string(req.Type), startDate, endDate, location, notes)
	if err != nil {
		server.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusCreated, entityToAPI(*a))
}

func (s *ActivityServer) GetActivity(w http.ResponseWriter, r *http.Request, id string) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	a, err := s.store.Get(id)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if a == nil || a.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	server.RespondJSON(w, http.StatusOK, entityToAPI(*a))
}

func (s *ActivityServer) UpdateActivity(w http.ResponseWriter, r *http.Request, id string) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	a, err := s.store.Get(id)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if a == nil || a.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	var req api.UpdateActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Apply only provided fields
	if req.Title != nil {
		a.Title = *req.Title
	}
	if req.Type != nil {
		a.Type = string(*req.Type)
	}
	if req.StartDate != nil {
		a.StartDate = req.StartDate.Format("2006-01-02")
	}
	if req.EndDate != nil {
		a.EndDate = req.EndDate.Format("2006-01-02")
	}
	if req.Location != nil {
		a.Location = *req.Location
	}
	if req.Notes != nil {
		a.Notes = *req.Notes
	}

	if err := s.store.Update(a); err != nil {
		server.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusOK, entityToAPI(*a))
}

func (s *ActivityServer) DeleteActivity(w http.ResponseWriter, r *http.Request, id string) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	a, err := s.store.Get(id)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if a == nil || a.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	if err := s.store.Delete(id); err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	server.RespondJSON(w, http.StatusOK, api.OkResponse{Ok: ptr("true")})
}

func (s *ActivityServer) CheckDate(w http.ResponseWriter, r *http.Request, date openapi_types.Date) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	dateStr := date.Format("2006-01-02")
	items, err := s.store.ForDate(userID, dateStr)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	location := "Home"
	locations := map[string]bool{}
	for _, a := range items {
		if a.Location != "" {
			locations[a.Location] = true
			if location == "Home" {
				location = a.Location
			}
		}
	}

	activities := make([]api.Activity, len(items))
	for i, a := range items {
		activities[i] = entityToAPI(a)
	}

	server.RespondJSON(w, http.StatusOK, api.DateCheck{
		Date:        date,
		Location:    location,
		Activities:  activities,
		HasConflict: len(locations) > 1,
	})
}

// --- helpers ---

func requireUser(w http.ResponseWriter, r *http.Request) string {
	userID := appbase.UserID(r)
	if userID == "" {
		server.RespondError(w, http.StatusUnauthorized, "not authenticated")
	}
	return userID
}

func entityToAPI(a Activity) api.Activity {
	startDate, _ := time.Parse("2006-01-02", a.StartDate)
	endDate, _ := time.Parse("2006-01-02", a.EndDate)
	createdAt, _ := time.Parse(time.RFC3339, a.CreatedAt)

	act := api.Activity{
		Id:        a.ID,
		UserId:    a.UserID,
		Title:     a.Title,
		Type:      api.ActivityType(a.Type),
		StartDate: openapi_types.Date{Time: startDate},
		EndDate:   openapi_types.Date{Time: endDate},
		Source:    api.ActivitySource(a.Source),
		CreatedAt: createdAt,
	}
	if a.Location != "" {
		act.Location = &a.Location
	}
	if a.Notes != "" {
		act.Notes = &a.Notes
	}
	return act
}

func ptr(s string) *string { return &s }
