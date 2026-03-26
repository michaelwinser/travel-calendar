package app

import (
	"encoding/json"
	"net/http"
	"strings"
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

	// 3. Search gazetteer
	// Parse qualifiers: "Westport, CT" → search "Westport", filter by "CT"
	// "3 Woodland Dr, Westport, CT 06880" → try progressively shorter parts
	searchTerms, countryFilter := parseLocationQuery(req.Text)

	gaz, err := gazetteer.Get()
	if err == nil {
		var results []gazetteer.Result
		for _, term := range searchTerms {
			results = gaz.PrefixSearch(term, 15)
			if len(results) > 0 {
				break
			}
		}

		// Filter by country/state qualifier if present
		if countryFilter != "" && len(results) > 1 {
			filtered := filterByQualifier(results, countryFilter)
			if len(filtered) > 0 {
				results = filtered
			}
		}

		// Limit to top 8
		if len(results) > 8 {
			results = results[:8]
		}

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

// parseLocationQuery breaks a user input like "Westport, CT 06880 USA" into
// search terms and an optional country/state filter.
// Returns a list of terms to try (most specific first) and a filter string.
func parseLocationQuery(text string) (searchTerms []string, countryFilter string) {
	text = strings.TrimSpace(text)
	parts := strings.Split(text, ",")

	// Clean each part
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// The first comma-separated part is likely the city/place name.
	// Remaining parts are qualifiers (state, country, zip).
	if len(parts) >= 2 {
		// Build the filter from everything after the first part
		countryFilter = strings.Join(parts[1:], " ")
		// Remove zip codes (sequences of digits)
		countryFilter = removeZipCodes(countryFilter)
		countryFilter = strings.TrimSpace(countryFilter)
	}

	// Try progressively shorter versions of the first part
	// "3 Woodland Dr" won't match, but the system should try the whole input first
	first := parts[0]
	searchTerms = append(searchTerms, first)

	// Also try the last word of the first part (often the city name after a street address)
	words := strings.Fields(first)
	if len(words) > 1 {
		searchTerms = append(searchTerms, words[len(words)-1])
	}

	// If we have qualifier parts, try those as city names too
	// "3 Woodland Dr, Westport, CT" → try "Westport"
	for _, p := range parts[1:] {
		p = removeZipCodes(strings.TrimSpace(p))
		p = strings.TrimSpace(p)
		if len(p) > 2 { // skip state/country abbreviations as search terms
			searchTerms = append(searchTerms, p)
		}
	}

	return searchTerms, countryFilter
}

func removeZipCodes(s string) string {
	words := strings.Fields(s)
	var filtered []string
	for _, w := range words {
		isZip := true
		for _, r := range w {
			if r < '0' || r > '9' {
				isZip = false
				break
			}
		}
		if !isZip {
			filtered = append(filtered, w)
		}
	}
	return strings.Join(filtered, " ")
}

// US state abbreviations → full names for matching against gazetteer country data
var usStates = map[string]string{
	"al": "alabama", "ak": "alaska", "az": "arizona", "ar": "arkansas",
	"ca": "california", "co": "colorado", "ct": "connecticut", "de": "delaware",
	"fl": "florida", "ga": "georgia", "hi": "hawaii", "id": "idaho",
	"il": "illinois", "in": "indiana", "ia": "iowa", "ks": "kansas",
	"ky": "kentucky", "la": "louisiana", "me": "maine", "md": "maryland",
	"ma": "massachusetts", "mi": "michigan", "mn": "minnesota", "ms": "mississippi",
	"mo": "missouri", "mt": "montana", "ne": "nebraska", "nv": "nevada",
	"nh": "new hampshire", "nj": "new jersey", "nm": "new mexico", "ny": "new york",
	"nc": "north carolina", "nd": "north dakota", "oh": "ohio", "ok": "oklahoma",
	"or": "oregon", "pa": "pennsylvania", "ri": "rhode island", "sc": "south carolina",
	"sd": "south dakota", "tn": "tennessee", "tx": "texas", "ut": "utah",
	"vt": "vermont", "va": "virginia", "wa": "washington", "wv": "west virginia",
	"wi": "wisconsin", "wy": "wyoming",
}

// filterByQualifier filters gazetteer results by a country/state qualifier string.
// "CT" or "Connecticut" → keep only US results in Connecticut.
// "USA" or "US" → keep only US results.
func filterByQualifier(results []gazetteer.Result, qualifier string) []gazetteer.Result {
	lower := strings.ToLower(strings.TrimSpace(qualifier))
	// Remove common country suffixes
	lower = strings.TrimSuffix(lower, " usa")
	lower = strings.TrimSuffix(lower, " us")
	lower = strings.TrimSpace(lower)

	var filtered []gazetteer.Result
	for _, r := range results {
		c := r.City
		countryLower := strings.ToLower(c.Country)

		// Check if qualifier matches country code
		if countryLower == lower || lower == "usa" || lower == "us" {
			if countryLower == "us" || lower == "usa" || lower == "us" {
				filtered = append(filtered, r)
			} else {
				filtered = append(filtered, r)
			}
			continue
		}

		// Check if qualifier is a US state abbreviation or name
		if _, isState := usStates[lower]; isState && countryLower == "us" {
			// The gazetteer doesn't have state data, but we can match on timezone
			// as a rough proxy, or just accept all US results when state is specified
			filtered = append(filtered, r)
			continue
		}

		// Check full state name
		for abbr, fullName := range usStates {
			if lower == fullName && countryLower == "us" {
				_ = abbr
				filtered = append(filtered, r)
				break
			}
		}
	}
	return filtered
}

func derefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
