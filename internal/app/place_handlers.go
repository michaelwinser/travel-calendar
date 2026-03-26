package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/michaelwinser/appbase/server"
	"github.com/michaelwinser/travel-calendar/api"
	"github.com/michaelwinser/travel-calendar/gazetteer"
)

func (s *ActivityServer) ListPlaces(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	places, err := s.places.List(userID)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]api.Place, len(places))
	for i, p := range places {
		result[i] = placeToAPI(p)
	}
	server.RespondJSON(w, http.StatusOK, result)
}

func (s *ActivityServer) CreatePlace(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req api.CreatePlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		server.RespondError(w, http.StatusBadRequest, "name is required")
		return
	}

	kind := "city"
	if req.Kind != nil {
		kind = string(*req.Kind)
	}

	var aliases []string
	if req.Aliases != nil {
		aliases = *req.Aliases
	}

	p := &Place{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      req.Name,
		Aliases:   encodeAliases(aliases),
		City:      derefStr(req.City),
		Country:   derefStr(req.Country),
		Latitude:  derefFloat(req.Latitude),
		Longitude: derefFloat(req.Longitude),
		Timezone:  derefStr(req.Timezone),
		Kind:      kind,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	if err := s.places.Create(p); err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusCreated, placeToAPI(*p))
}

func (s *ActivityServer) GetPlace(w http.ResponseWriter, r *http.Request, id string) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	p, err := s.places.Get(id)
	if err != nil || p == nil || p.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	server.RespondJSON(w, http.StatusOK, placeToAPI(*p))
}

func (s *ActivityServer) UpdatePlace(w http.ResponseWriter, r *http.Request, id string) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	p, err := s.places.Get(id)
	if err != nil || p == nil || p.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	var req api.UpdatePlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Aliases != nil {
		p.Aliases = encodeAliases(*req.Aliases)
	}
	if req.City != nil {
		p.City = *req.City
	}
	if req.Country != nil {
		p.Country = *req.Country
	}
	if req.Latitude != nil {
		p.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		p.Longitude = *req.Longitude
	}
	if req.Timezone != nil {
		p.Timezone = *req.Timezone
	}
	if req.Kind != nil {
		p.Kind = string(*req.Kind)
	}

	if err := s.places.Update(p); err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusOK, placeToAPI(*p))
}

func (s *ActivityServer) DeletePlace(w http.ResponseWriter, r *http.Request, id string) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	p, err := s.places.Get(id)
	if err != nil || p == nil || p.UserID != userID {
		server.RespondError(w, http.StatusNotFound, "not found")
		return
	}

	if err := s.places.Delete(id); err != nil {
		server.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondJSON(w, http.StatusOK, api.OkResponse{Ok: ptr("true")})
}

func (s *ActivityServer) ResolvePlaces(w http.ResponseWriter, r *http.Request) {
	userID := requireUser(w, r)
	if userID == "" {
		return
	}

	var req api.PlaceResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		server.RespondError(w, http.StatusBadRequest, "text is required")
		return
	}

	resp := api.PlaceResolveResponse{
		Suggestions: []api.PlaceSuggestion{},
	}

	// 1. Check for exact match in user's places
	exact, _ := s.places.FindByName(userID, req.Text)
	if exact != nil {
		p := placeToAPI(*exact)
		resp.Exact = &p
	}

	// 2. Prefix search user's places
	userMatches, _ := s.places.SearchByPrefix(userID, req.Text, 5)
	seen := map[string]bool{}
	if exact != nil {
		seen[exact.ID] = true
	}
	for _, p := range userMatches {
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		ap := placeToAPI(p)
		resp.Suggestions = append(resp.Suggestions, api.PlaceSuggestion{
			Source: api.User,
			Place:  &ap,
			Name:   p.Name,
			Score:  0.9,
		})
	}

	// 3. Search gazetteer with right-to-left location parsing
	gaz, err := gazetteer.Get()
	if err == nil {
		results := gaz.ResolveLocation(req.Text, 8)

		for _, gr := range results {
			c := gr.City
			key := c.Name + "|" + c.Country
			if seen[key] {
				continue
			}
			seen[key] = true

			sug := api.PlaceSuggestion{
				Source: api.Gazetteer,
				Name:   c.Name,
				Score:  gr.Score,
			}
			if c.Country != "" {
				sug.Country = &c.Country
			}
			if c.Admin1 != "" {
				sug.Admin1 = &c.Admin1
			}
			if c.Latitude != 0 || c.Longitude != 0 {
				sug.Latitude = &c.Latitude
				sug.Longitude = &c.Longitude
			}
			if c.Timezone != "" {
				sug.Timezone = &c.Timezone
			}
			if c.Population > 0 {
				pop := c.Population
				sug.Population = &pop
			}
			resp.Suggestions = append(resp.Suggestions, sug)
		}
	}

	server.RespondJSON(w, http.StatusOK, resp)
}

// --- helpers ---

func placeToAPI(p Place) api.Place {
	createdAt, _ := time.Parse(time.RFC3339, p.CreatedAt)
	place := api.Place{
		Id:        p.ID,
		Name:      p.Name,
		Kind:      api.PlaceKind(p.Kind),
		CreatedAt: createdAt,
	}
	aliases := decodeAliases(p.Aliases)
	if len(aliases) > 0 {
		place.Aliases = &aliases
	}
	if p.City != "" {
		place.City = &p.City
	}
	if p.Country != "" {
		place.Country = &p.Country
	}
	if p.Latitude != 0 || p.Longitude != 0 {
		place.Latitude = &p.Latitude
		place.Longitude = &p.Longitude
	}
	if p.Timezone != "" {
		place.Timezone = &p.Timezone
	}
	return place
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Old parseLocationQuery, filterByQualifier, usStates removed —
// replaced by gazetteer.ResolveLocation with right-to-left parsing.

func derefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
