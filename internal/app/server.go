package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
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
	store          *ActivityStore
	trips          *TripStore
	parseHistory   *ParseHistoryStore
	shareLinks     *ShareLinkStore
	shares         *ShareStore
	publicProfiles *PublicProfileStore
	places         *PlaceStore
}

// NewActivityServer creates a new server with all dependencies.
func NewActivityServer(store *ActivityStore, trips *TripStore, parseHistory *ParseHistoryStore, shareLinks *ShareLinkStore, shares *ShareStore, publicProfiles *PublicProfileStore, places *PlaceStore) *ActivityServer {
	return &ActivityServer{store: store, trips: trips, parseHistory: parseHistory, shareLinks: shareLinks, shares: shares, publicProfiles: publicProfiles, places: places}
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
	placeID := ""
	if req.PlaceId != nil {
		placeID = *req.PlaceId
	}
	originPlaceID := ""
	if req.OriginPlaceId != nil {
		originPlaceID = *req.OriginPlaceId
	}
	destPlaceID := ""
	if req.DestinationPlaceId != nil {
		destPlaceID = *req.DestinationPlaceId
	}

	a, err := s.store.Create(userID, req.Title, string(req.Type), startDate, endDate, location, notes, tripID, placeID, originPlaceID, destPlaceID)
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
	if req.PlaceId != nil {
		a.PlaceID = *req.PlaceId
	}
	if req.OriginPlaceId != nil {
		a.OriginPlaceID = *req.OriginPlaceId
	}
	if req.DestinationPlaceId != nil {
		a.DestinationPlaceID = *req.DestinationPlaceId
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
	for _, a := range items {
		if a.Location != "" && location == "Home" {
			location = a.Location
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
		HasConflict: detectConflict(items),
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

// --- Share Link handlers ---

func (s *ActivityServer) ListShareLinks(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	links, err := s.shareLinks.ListByUser(userID)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]api.ShareLink, len(links))
	for i, l := range links {
		result[i] = shareLinkToAPI(l)
	}
	server.RespondJSON(w, http.StatusOK, result)
}

func (s *ActivityServer) CreateShareLink(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req api.CreateShareLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	label := ""
	if req.Label != nil {
		label = *req.Label
	}
	expiresAt := ""
	if req.ExpiresAt != nil {
		expiresAt = req.ExpiresAt.Format(time.RFC3339)
	}
	fromDate := ""
	if req.FromDate != nil {
		fromDate = req.FromDate.Format("2006-01-02")
	}
	toDate := ""
	if req.ToDate != nil {
		toDate = req.ToDate.Format("2006-01-02")
	}
	tripIDs := ""
	if req.TripIds != nil {
		tripIDs = *req.TripIds
	}
	showTitle := false
	if req.ShowTitle != nil {
		showTitle = *req.ShowTitle
	}

	ownerEmail := appbase.Email(r)
	link, err := s.shareLinks.Create(userID, ownerEmail, label, expiresAt, fromDate, toDate, tripIDs, showTitle)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusCreated, shareLinkToAPI(*link))
}

func (s *ActivityServer) DeleteShareLink(w http.ResponseWriter, r *http.Request, id string) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	link, err := s.shareLinks.Get(id)
	if err != nil || link == nil || link.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	if err := s.shareLinks.Delete(id); err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusOK, api.OkResponse{Ok: ptr("true")})
}

// HandleSharedCalendar serves the public shared calendar endpoint (no auth required).
// Expects the route pattern /shared/{token}.json
func (s *ActivityServer) HandleSharedCalendar(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		// Fallback: extract from path, stripping .json suffix
		token = strings.TrimPrefix(r.URL.Path, "/shared/")
		token = strings.TrimSuffix(token, ".json")
	}
	if token == "" {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	link, err := s.shareLinks.GetByToken(token)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if link == nil {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	// Check expiry
	if link.ExpiresAt != "" {
		expiry, err := time.Parse(time.RFC3339, link.ExpiresAt)
		if err == nil && time.Now().After(expiry) {
			server.RespondError(w, http.StatusGone, "this share link has expired")
			return
		}
	}

	// Query activities with filters
	var activities []Activity
	if link.FromDate != "" && link.ToDate != "" {
		activities, err = s.store.ListRange(link.UserID, link.FromDate, link.ToDate)
	} else {
		activities, err = s.store.List(link.UserID)
	}
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Filter by trip IDs if specified
	if link.TripIDs != "" {
		tripIDSet := map[string]bool{}
		for _, id := range strings.Split(link.TripIDs, ",") {
			tripIDSet[strings.TrimSpace(id)] = true
		}
		filtered := activities[:0]
		for _, a := range activities {
			if tripIDSet[a.TripID] {
				filtered = append(filtered, a)
			}
		}
		activities = filtered
	}

	// Build trip lookup and convert to shared activities
	tripMap := s.buildTripMap(link.UserID)
	shared := make([]api.SharedActivity, len(activities))
	for i, a := range activities {
		shared[i] = entityToSharedActivity(a, link.ShowTitle, tripMap)
	}

	server.RespondJSON(w, http.StatusOK, api.SharedCalendarResponse{
		Label:      link.Label,
		OwnerEmail: &link.OwnerEmail,
		Activities: shared,
	})
}

// --- User-to-user share handlers ---

func (s *ActivityServer) ListShares(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	shares, err := s.shares.ListByOwner(userID)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]api.Share, len(shares))
	for i, sh := range shares {
		result[i] = shareToAPI(sh)
	}
	server.RespondJSON(w, http.StatusOK, result)
}

func (s *ActivityServer) CreateShare(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req api.CreateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" {
		server.RespondError(w, http.StatusBadRequest, "email is required")
		return
	}

	ownerEmail := appbase.Email(r)
	if req.Email == ownerEmail {
		server.RespondError(w, http.StatusBadRequest, "cannot share with yourself")
		return
	}

	// Check for duplicate
	existing, _ := s.shares.FindByOwnerAndRecipient(ownerEmail, req.Email)
	if existing != nil {
		server.RespondError(w, http.StatusBadRequest, "already shared with this user")
		return
	}

	showTitle := false
	if req.ShowTitle != nil {
		showTitle = *req.ShowTitle
	}

	sh, err := s.shares.Create(userID, ownerEmail, req.Email, showTitle)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusCreated, shareToAPI(*sh))
}

func (s *ActivityServer) DeleteShare(w http.ResponseWriter, r *http.Request, id string) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	sh, err := s.shares.Get(id)
	if err != nil || sh == nil || sh.OwnerUserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	if err := s.shares.Delete(id); err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusOK, api.OkResponse{Ok: ptr("true")})
}

func (s *ActivityServer) ListSharedWithMe(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	email := appbase.Email(r)
	shares, err := s.shares.ListByRecipient(email)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]api.SharedWithMeEntry, len(shares))
	for i, sh := range shares {
		result[i] = api.SharedWithMeEntry{
			ShareId:    sh.ID,
			OwnerEmail: sh.OwnerEmail,
		}
	}
	server.RespondJSON(w, http.StatusOK, result)
}

func (s *ActivityServer) GetSharedActivities(w http.ResponseWriter, r *http.Request, email string, params api.GetSharedActivitiesParams) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	myEmail := appbase.Email(r)
	share, err := s.shares.FindByOwnerAndRecipient(email, myEmail)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if share == nil {
		server.RespondError(w, http.StatusNotFound, "no share found from this user")
		return
	}

	// Query the owner's activities with optional filters
	var activities []Activity
	if params.Month != nil {
		t, perr := time.Parse("2006-01", *params.Month)
		if perr != nil {
			server.RespondError(w, http.StatusBadRequest, "invalid month (expected YYYY-MM)")
			return
		}
		from := t.Format("2006-01-02")
		to := t.AddDate(0, 1, -1).Format("2006-01-02")
		activities, err = s.store.ListRange(share.OwnerUserID, from, to)
	} else if params.From != nil && params.To != nil {
		activities, err = s.store.ListRange(share.OwnerUserID, params.From.Format("2006-01-02"), params.To.Format("2006-01-02"))
	} else {
		activities, err = s.store.List(share.OwnerUserID)
	}
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tripMap := s.buildTripMap(share.OwnerUserID)
	shared := make([]api.SharedActivity, len(activities))
	for i, a := range activities {
		shared[i] = entityToSharedActivity(a, share.ShowTitle, tripMap)
	}

	server.RespondJSON(w, http.StatusOK, api.SharedCalendarResponse{
		Label:      share.OwnerEmail + "'s calendar",
		OwnerEmail: &share.OwnerEmail,
		Activities: shared,
	})
}

// --- Public profile handlers ---

func (s *ActivityServer) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	p, err := s.publicProfiles.GetByUserID(userID)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if p == nil {
		// Return empty/default profile
		server.RespondJSON(w, http.StatusOK, api.PublicProfile{Handle: "", Enabled: false})
		return
	}

	server.RespondJSON(w, http.StatusOK, api.PublicProfile{Handle: p.Handle, Enabled: p.Enabled})
}

func (s *ActivityServer) UpdatePublicProfile(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req api.UpdatePublicProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate handle
	if err := validateHandle(req.Handle); err != nil {
		server.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check handle uniqueness
	existing, _ := s.publicProfiles.GetByHandle(req.Handle)
	if existing != nil && existing.UserID != userID {
		server.RespondError(w, http.StatusBadRequest, "handle is already taken")
		return
	}

	p, err := s.publicProfiles.GetByUserID(userID)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if p == nil {
		// Create new profile
		p = &PublicProfile{
			ID:        uuid.New().String(),
			UserID:    userID,
			Handle:    req.Handle,
			Enabled:   req.Enabled,
			CreatedAt: time.Now().Format(time.RFC3339),
		}
		if err := s.publicProfiles.Create(p); err != nil {
			server.RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		p.Handle = req.Handle
		p.Enabled = req.Enabled
		if err := s.publicProfiles.Update(p); err != nil {
			server.RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	server.RespondJSON(w, http.StatusOK, api.PublicProfile{Handle: p.Handle, Enabled: p.Enabled})
}

// HandlePublicDashboard serves the public "Where is X" dashboard (no auth).
func (s *ActivityServer) HandlePublicDashboard(w http.ResponseWriter, r *http.Request) {
	handle := r.PathValue("handle")
	if handle == "" {
		handle = strings.TrimPrefix(r.URL.Path, "/public/")
		handle = strings.TrimSuffix(handle, ".json")
	}
	isJSON := strings.HasSuffix(r.URL.Path, ".json")

	p, err := s.publicProfiles.GetByHandle(handle)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if p == nil || !p.Enabled {
		if isJSON {
			server.RespondError(w, http.StatusNotFound, "not found")
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("<html><body><p style='text-align:center;padding:3rem;color:#999'>Public profile not found.</p></body></html>"))
		}
		return
	}

	// Query -2 weeks to +4 weeks of activities
	now := time.Now()
	rangeStart := now.AddDate(0, 0, -14)
	rangeEnd := now.AddDate(0, 0, 28)
	from := rangeStart.Format("2006-01-02")
	to := rangeEnd.Format("2006-01-02")

	activities, err := s.store.ListRange(p.UserID, from, to)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	tripMap := s.buildTripMap(p.UserID)

	// Privacy filter: strip titles and notes, keep location + type + trip info
	shared := make([]api.SharedActivity, len(activities))
	for i, a := range activities {
		shared[i] = entityToSharedActivity(a, false, tripMap)
	}

	w.Header().Set("Cache-Control", "public, max-age=3600")

	if isJSON {
		server.RespondJSON(w, http.StatusOK, api.SharedCalendarResponse{
			Label:      "Where is " + p.Handle + "?",
			OwnerEmail: nil,
			Activities: shared,
		})
		return
	}

	// Serve the public frontend entry point
	// This is handled by the /public/{handle} route serving public.html
	// If we get here for a non-JSON request, it means the frontend isn't built yet
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("<html><body><p>Public dashboard frontend not available. Use /public/" + p.Handle + ".json for data.</p></body></html>"))
}


// --- conflict detection ---

// detectConflict determines whether a set of activities on the same day
// represent a scheduling conflict. It uses three layers:
//  1. Place identity: same placeId = same location (no conflict)
//  2. Travel bridging: travel-type activities connect locations
//  3. String fallback: different location strings = conflict
// locKey identifies a location by either placeID or lowercase string.
type locKey struct {
	placeID string
	locStr  string
}

func detectConflict(items []Activity) bool {
	allLocs := map[locKey]bool{}
	var routeTravelActivities []Activity

	for _, a := range items {
		if a.Location == "" && a.PlaceID == "" {
			continue
		}

		// Travel activities with route-style locations (A → B) are potential bridges.
		// Travel activities with plain locations (e.g., "Seattle") are treated like
		// any other activity for conflict purposes.
		if a.Type == TypeTravel {
			loc := a.Location
			if strings.Contains(loc, "→") || strings.Contains(loc, "->") ||
				(a.OriginPlaceID != "" && a.DestinationPlaceID != "") {
				routeTravelActivities = append(routeTravelActivities, a)
				continue
			}
		}

		key := locKey{placeID: a.PlaceID, locStr: strings.ToLower(a.Location)}
		allLocs[key] = true
	}

	if len(allLocs) <= 1 {
		return false
	}

	// Check if route-style travel activities bridge the conflicting locations.
	if len(routeTravelActivities) > 0 {
		// Try structured origin/destination graph
		if buildTravelGraph(routeTravelActivities, allLocs) {
			return false
		}

		// Fallback: route-style travel (e.g., "EWR → CDG") bridges exactly 2 locations
		if len(allLocs) <= 2 {
			return false
		}
	}

	return true
}

// buildTravelGraph checks whether travel activities connect all non-travel locations
// into a single connected component using origin/destination place IDs.
func buildTravelGraph(travels []Activity, locs map[locKey]bool) bool {
	keys := make([]locKey, 0, len(locs))
	for k := range locs {
		keys = append(keys, k)
	}

	// Union-Find
	parent := map[locKey]locKey{}
	for k := range locs {
		parent[k] = k
	}

	var find func(locKey) locKey
	find = func(k locKey) locKey {
		if parent[k] != k {
			parent[k] = find(parent[k])
		}
		return parent[k]
	}

	union := func(a, b locKey) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}

	// For each travel activity with origin/destination, union those locations
	for _, t := range travels {
		var originKey, destKey locKey
		hasOrigin, hasDest := false, false

		if t.OriginPlaceID != "" {
			originKey = locKey{placeID: t.OriginPlaceID}
			// Find matching non-travel location
			for k := range locs {
				if k.placeID == t.OriginPlaceID {
					originKey = k
					hasOrigin = true
					break
				}
			}
		}
		if t.DestinationPlaceID != "" {
			destKey = locKey{placeID: t.DestinationPlaceID}
			for k := range locs {
				if k.placeID == t.DestinationPlaceID {
					destKey = k
					hasDest = true
					break
				}
			}
		}

		if hasOrigin && hasDest {
			union(originKey, destKey)
		}
	}

	// Check if all locations are in the same component
	if len(keys) == 0 {
		return true
	}
	root := find(keys[0])
	for _, k := range keys[1:] {
		if find(k) != root {
			return false
		}
	}
	return true
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
	if a.PlaceID != "" {
		act.PlaceId = &a.PlaceID
	}
	if a.OriginPlaceID != "" {
		act.OriginPlaceId = &a.OriginPlaceID
	}
	if a.DestinationPlaceID != "" {
		act.DestinationPlaceId = &a.DestinationPlaceID
	}
	return act
}

func shareLinkToAPI(l ShareLink) api.ShareLink {
	createdAt, _ := time.Parse(time.RFC3339, l.CreatedAt)
	link := api.ShareLink{
		Id:        l.ID,
		Token:     l.Token,
		Label:     l.Label,
		ShowTitle: l.ShowTitle,
		CreatedAt: createdAt,
	}
	if l.ExpiresAt != "" {
		t, _ := time.Parse(time.RFC3339, l.ExpiresAt)
		link.ExpiresAt = &t
	}
	if l.FromDate != "" {
		d, _ := time.Parse("2006-01-02", l.FromDate)
		link.FromDate = &openapi_types.Date{Time: d}
	}
	if l.ToDate != "" {
		d, _ := time.Parse("2006-01-02", l.ToDate)
		link.ToDate = &openapi_types.Date{Time: d}
	}
	if l.TripIDs != "" {
		link.TripIds = &l.TripIDs
	}
	return link
}

func shareToAPI(sh Share) api.Share {
	createdAt, _ := time.Parse(time.RFC3339, sh.CreatedAt)
	return api.Share{
		Id:         sh.ID,
		OwnerEmail: sh.OwnerEmail,
		SharedWith: sh.SharedWith,
		ShowTitle:  sh.ShowTitle,
		CreatedAt:  createdAt,
	}
}

func entityToSharedActivity(a Activity, showTitle bool, tripMap map[string]Trip) api.SharedActivity {
	startDate, _ := time.Parse("2006-01-02", a.StartDate)
	endDate, _ := time.Parse("2006-01-02", a.EndDate)
	sa := api.SharedActivity{
		Type:      api.SharedActivityType(a.Type),
		StartDate: openapi_types.Date{Time: startDate},
		EndDate:   openapi_types.Date{Time: endDate},
	}
	if a.Location != "" {
		sa.Location = &a.Location
	}
	if showTitle {
		sa.Title = &a.Title
	}
	if t, ok := tripMap[a.TripID]; ok {
		sa.TripName = &t.Name
		sa.TripColor = &t.Color
	}
	return sa
}

// buildTripMap returns a map of trip ID to Trip for the given user.
func (s *ActivityServer) buildTripMap(userID string) map[string]Trip {
	tripMap := map[string]Trip{}
	if trips, err := s.trips.List(userID); err == nil {
		for _, t := range trips {
			tripMap[t.ID] = t
		}
	}
	return tripMap
}

var handleRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,28}[a-z0-9]$`)

func validateHandle(h string) error {
	if !handleRe.MatchString(h) {
		return fmt.Errorf("handle must be 3-30 chars, lowercase alphanumeric and hyphens")
	}
	return nil
}

func ptr(s string) *string { return &s }
