package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	icalparser "github.com/michaelwinser/travel-calendar/internal/ical"
)

// FilterConfig controls which events pass from source to staging.
type FilterConfig struct {
	ExcludeKeywords        []string `json:"excludeKeywords,omitempty"`
	IncludeKeywords        []string `json:"includeKeywords,omitempty"`
	DisableBuiltinExcludes bool     `json:"disableBuiltinExcludes,omitempty"`
	DisableBuiltinIncludes bool     `json:"disableBuiltinIncludes,omitempty"`
}

// SyncResult summarizes what happened during a sync.
type SyncResult struct {
	Fetched  int `json:"fetched"`
	Staged   int `json:"staged"`
	Updated  int `json:"updated"`
	Filtered int `json:"filtered"`
	Errors   int `json:"errors"`
}

// SyncSource fetches events from a source, applies filters, and creates/updates staged events.
// If activityStore is non-nil, changes to imported events are propagated to their linked activities.
func SyncSource(source *ImportSource, stagedStore *StagedEventStore, activityStore *ActivityStore) (*SyncResult, error) {
	result := &SyncResult{}

	// Parse filter config
	var fc FilterConfig
	if source.FilterConfig != "" {
		json.Unmarshal([]byte(source.FilterConfig), &fc)
	}

	// Fetch events based on source type
	var events []icalparser.Event
	switch source.SourceType {
	case "ical", "":
		var err error
		events, err = fetchIcalEvents(source.URL)
		if err != nil {
			return nil, err
		}
	default:
		return nil, nil
	}

	result.Fetched = len(events)

	// Time window: only stage events within -30 to +360 days
	now := time.Now()
	windowStart := now.AddDate(0, 0, -30).Format("2006-01-02")
	windowEnd := now.AddDate(0, 0, 360).Format("2006-01-02")

	// Process each event
	for _, event := range events {
		startDate := event.StartDate()
		endDate := event.EndDate()

		// Filter by time window
		if endDate < windowStart || startDate > windowEnd {
			result.Filtered++
			continue
		}

		// Apply content filters
		if !shouldStage(event, &fc) {
			result.Filtered++
			continue
		}

		actType := inferActivityType(event)

		// Check if this event already exists in staging
		existing, _ := stagedStore.FindBySourceEventID(source.ID, event.UID)

		if existing != nil {
			// Update existing staged event (preserve state and activityId)
			changed := false
			if existing.Title != event.Summary {
				existing.Title = event.Summary
				changed = true
			}
			if existing.StartDate != startDate {
				existing.StartDate = startDate
				changed = true
			}
			if existing.EndDate != endDate {
				existing.EndDate = endDate
				changed = true
			}
			if existing.Location != event.Location {
				existing.Location = event.Location
				changed = true
			}
			if changed {
				stagedStore.Update(existing)
				result.Updated++

				// Propagate changes to linked activity if imported
				if existing.State == "imported" && existing.ActivityID != "" && activityStore != nil {
					if act, err := activityStore.Get(existing.ActivityID); err == nil && act != nil {
						act.Title = existing.Title
						act.StartDate = existing.StartDate
						act.EndDate = existing.EndDate
						act.Location = existing.Location
						activityStore.Update(act)
					}
				}
			}
		} else {
			// Create new staged event
			staged := &StagedEvent{
				UserID:        source.UserID,
				SourceID:      source.ID,
				SourceEventID: event.UID,
				Title:         event.Summary,
				Type:          actType,
				StartDate:     startDate,
				EndDate:       endDate,
				Location:      event.Location,
				Notes:         event.Description,
				State:         "new",
			}
			if err := stagedStore.Create(staged); err != nil {
				result.Errors++
				continue
			}
			result.Staged++
		}
	}

	// Update source last sync time
	source.LastSyncAt = time.Now().Format(time.RFC3339)

	return result, nil
}

func fetchIcalEvents(url string) ([]icalparser.Event, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, err
	}

	return icalparser.Parse(resp.Body)
}

// --- Filter logic ---

// Built-in include keywords for type inference (separate from filter patterns)
var filterStayKeywords = []string{"hotel", "airbnb", "vrbo", "stay", "accommodation", "check-in", "checkout"}

func shouldStage(event icalparser.Event, fc *FilterConfig) bool {
	title := strings.ToLower(event.Summary)
	location := strings.ToLower(event.Location)

	// Load filter patterns (built-in + user-defined from FilterConfig)
	filters := LoadDefaultFilters()
	if fc.DisableBuiltinExcludes {
		var kept []Filter
		for _, f := range filters {
			if !(f.Builtin && f.Type == "exclude") {
				kept = append(kept, f)
			}
		}
		filters = kept
	}
	if fc.DisableBuiltinIncludes {
		var kept []Filter
		for _, f := range filters {
			if !(f.Builtin && f.Type == "include") {
				kept = append(kept, f)
			}
		}
		filters = kept
	}
	// Add user keywords as filters
	for _, kw := range fc.ExcludeKeywords {
		filters = append(filters, Filter{Pattern: kw, Type: "exclude", Enabled: true})
	}
	for _, kw := range fc.IncludeKeywords {
		filters = append(filters, Filter{Pattern: kw, Type: "include", Enabled: true})
	}

	// Step 1: Check exclude filters — match against LOCATION and TITLE only (not description)
	excluded := false
	for _, f := range filters {
		if f.Type != "exclude" || !f.Enabled {
			continue
		}
		lower := strings.ToLower(f.Pattern)
		if strings.Contains(location, lower) || strings.Contains(title, lower) {
			excluded = true
			break
		}
	}

	// Step 1b: URL-only location — if location is just a URL with no other text, exclude
	if !excluded && location != "" && isURLOnly(location) {
		excluded = true
	}

	// Step 2: Check include filters — match against TITLE and LOCATION (not description)
	// Include overrides exclude
	included := false
	for _, f := range filters {
		if f.Type != "include" || !f.Enabled {
			continue
		}
		lower := strings.ToLower(f.Pattern)
		if strings.Contains(title, lower) || strings.Contains(location, lower) {
			included = true
			break
		}
	}

	// Step 3: Structural includes (always apply)
	// All-day events and multi-day events are almost always travel-relevant
	if event.AllDay {
		included = true
	}
	if event.StartDate() != event.EndDate() {
		included = true
	}

	// Step 4: Physical location — if location has non-URL text, likely a real place
	if location != "" && !excluded && !isURLOnly(location) {
		included = true
	}

	// Step 5: Decision
	if included {
		return true
	}
	if excluded {
		return false
	}
	// No location, no keyword match — skip
	return false
}

// isURLOnly returns true if the string is just a URL with no meaningful text.
func isURLOnly(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func inferActivityType(event icalparser.Event) string {
	title := strings.ToLower(event.Summary)
	words := strings.Fields(title)

	for _, w := range words {
		if travelKeywords[w] {
			return TypeTravel
		}
		if conferenceKeywords[w] {
			return TypeConference
		}
		if commitmentKeywords[w] {
			return TypeCommitment
		}
	}
	for _, kw := range filterStayKeywords {
		if strings.Contains(title, kw) {
			return TypeStay
		}
	}

	// Multi-day with location → vacation
	if event.StartDate() != event.EndDate() && event.Location != "" {
		return TypeVacation
	}

	// Single day with location → commitment
	if event.Location != "" {
		return TypeCommitment
	}

	return TypeStay
}
