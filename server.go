package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/michaelwinser/appbase"
	"github.com/michaelwinser/appbase/server"
	"github.com/michaelwinser/travel-calendar/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Ensure ActivityServer implements the generated interface.
var _ api.ServerInterface = (*ActivityServer)(nil)

// Trip color palette for auto-assignment.
var tripColors = []string{
	"#4f86c6", "#e07b53", "#6bb86a", "#c75ca2",
	"#d4a843", "#5cbcb6", "#8b6cc1", "#c95454",
	"#5a8f5a", "#c4853d",
}

// ActivityServer implements the generated ServerInterface.
type ActivityServer struct {
	store        *ActivityStore
	trips        *TripStore
	parseHistory *ParseHistoryStore
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
	tripID := ""
	if req.TripId != nil {
		tripID = *req.TripId
	}

	a, err := s.store.Create(userID, req.Title, string(req.Type), startDate, endDate, location, notes, tripID)
	if err != nil {
		server.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Link to parse history if provided
	if req.ParseHistoryId != nil && *req.ParseHistoryId != "" && s.parseHistory != nil {
		s.parseHistory.MarkAccepted(*req.ParseHistoryId, a.ID)
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
	if req.TripId != nil {
		a.TripID = *req.TripId
	}

	if err := s.store.Update(a); err != nil {
		server.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusOK, entityToAPI(*a))
}

func (s *ActivityServer) ParseActivity(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req api.ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		server.RespondError(w, http.StatusBadRequest, "text is required")
		return
	}

	result := Parse(req.Text, time.Now())

	// Build API response
	parsed := api.ParsedActivity{}
	if result.Title != "" {
		parsed.Title = &result.Title
	}
	if result.Type != "" {
		t := api.ParsedActivityType(result.Type)
		parsed.Type = &t
	}
	if result.StartDate != nil {
		parsed.StartDate = &openapi_types.Date{Time: *result.StartDate}
	}
	if result.EndDate != nil {
		parsed.EndDate = &openapi_types.Date{Time: *result.EndDate}
	}
	if result.Location != "" {
		parsed.Location = &result.Location
	}

	confidence := api.ParseConfidence{}
	if v, ok := result.Confidence["title"]; ok {
		c := api.ParseConfidenceTitle(v)
		confidence.Title = &c
	}
	if v, ok := result.Confidence["type"]; ok {
		c := api.ParseConfidenceType(v)
		confidence.Type = &c
	}
	if v, ok := result.Confidence["startDate"]; ok {
		c := api.ParseConfidenceStartDate(v)
		confidence.StartDate = &c
	}
	if v, ok := result.Confidence["endDate"]; ok {
		c := api.ParseConfidenceEndDate(v)
		confidence.EndDate = &c
	}
	if v, ok := result.Confidence["location"]; ok {
		c := api.ParseConfidenceLocation(v)
		confidence.Location = &c
	}

	unparsed := strings.Join(result.Unparsed, " ")

	// Save to parse history
	apiResult := api.ParseResult{
		Activity:   parsed,
		Confidence: confidence,
		Unparsed:   unparsed,
	}
	resultJSON, _ := json.Marshal(apiResult)
	todayStr := time.Now().Format("2006-01-02")
	historyID := ""
	if s.parseHistory != nil {
		h, err := s.parseHistory.Create(userID, req.Text, todayStr, string(resultJSON))
		if err == nil {
			historyID = h.ID
		}
	}
	apiResult.Id = historyID

	server.RespondJSON(w, http.StatusOK, api.ParseResult{
		Id:         historyID,
		Activity:   parsed,
		Confidence: confidence,
		Unparsed:   unparsed,
	})
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

// --- Trip handlers ---

func (s *ActivityServer) ListTrips(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	trips, err := s.trips.List(userID)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	activities, _ := s.store.List(userID)

	result := make([]api.TripSummary, 0, len(trips))
	for _, t := range trips {
		// Compute derived fields from activities
		var startDate, endDate string
		locSet := map[string]bool{}
		count := 0
		for _, a := range activities {
			if a.TripID != t.ID {
				continue
			}
			count++
			if startDate == "" || a.StartDate < startDate {
				startDate = a.StartDate
			}
			if endDate == "" || a.EndDate > endDate {
				endDate = a.EndDate
			}
			if a.Location != "" {
				locSet[a.Location] = true
			}
		}

		sd, _ := time.Parse("2006-01-02", startDate)
		ed, _ := time.Parse("2006-01-02", endDate)

		locs := make([]string, 0, len(locSet))
		for l := range locSet {
			locs = append(locs, l)
		}

		summary := api.TripSummary{
			Id:            t.ID,
			Name:          t.Name,
			Color:         t.Color,
			StartDate:     openapi_types.Date{Time: sd},
			EndDate:       openapi_types.Date{Time: ed},
			ActivityCount: count,
		}
		if len(locs) > 0 {
			summary.Locations = &locs
		}
		result = append(result, summary)
	}

	server.RespondJSON(w, http.StatusOK, result)
}

func (s *ActivityServer) CreateTrip(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req api.CreateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		server.RespondError(w, http.StatusBadRequest, "name is required")
		return
	}

	color := ""
	if req.Color != nil {
		color = *req.Color
	}
	if color == "" {
		// Auto-assign from palette based on name hash
		hash := 0
		for _, ch := range req.Name {
			hash = ((hash << 5) - hash + int(ch))
		}
		if hash < 0 {
			hash = -hash
		}
		color = tripColors[hash%len(tripColors)]
	}

	t, err := s.trips.Create(userID, req.Name, color)
	if err != nil {
		server.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	createdAt, _ := time.Parse(time.RFC3339, t.CreatedAt)
	server.RespondJSON(w, http.StatusCreated, api.Trip{
		Id:        t.ID,
		UserId:    t.UserID,
		Name:      t.Name,
		Color:     t.Color,
		CreatedAt: createdAt,
	})
}

func (s *ActivityServer) UpdateTrip(w http.ResponseWriter, r *http.Request, id string) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	t, err := s.trips.Get(id)
	if err != nil || t == nil || t.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	var req api.UpdateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Color != nil {
		t.Color = *req.Color
	}

	if err := s.trips.Update(t); err != nil {
		server.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	createdAt, _ := time.Parse(time.RFC3339, t.CreatedAt)
	server.RespondJSON(w, http.StatusOK, api.Trip{
		Id:        t.ID,
		UserId:    t.UserID,
		Name:      t.Name,
		Color:     t.Color,
		CreatedAt: createdAt,
	})
}

func (s *ActivityServer) DeleteTrip(w http.ResponseWriter, r *http.Request, id string) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	t, err := s.trips.Get(id)
	if err != nil || t == nil || t.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	// Unlink activities from this trip
	activities, _ := s.store.List(userID)
	for _, a := range activities {
		if a.TripID == id {
			a.TripID = ""
			s.store.Update(&a)
		}
	}

	if err := s.trips.Delete(id); err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusOK, api.OkResponse{Ok: ptr("true")})
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
	if a.TripID != "" {
		act.TripId = &a.TripID
	}
	return act
}

func ptr(s string) *string { return &s }
