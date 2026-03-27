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
func SyncSource(source *ImportSource, stagedStore *StagedEventStore) (*SyncResult, error) {
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

	// Process each event
	for _, event := range events {
		// Apply filters
		if !shouldStage(event, &fc) {
			result.Filtered++
			continue
		}

		startDate := event.StartDate()
		endDate := event.EndDate()
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

// Built-in exclude patterns
var videoConferencePatterns = []string{
	"zoom.us", "meet.google.com", "teams.microsoft.com", "webex.com",
	"whereby.com", "gotomeeting.com",
}

var meetingRoomPatterns = []string{
	"meeting room", "conf room", "conference room", "phone booth",
	"huddle room",
}

// Built-in include keywords for filtering (separate from parser keywords)
var filterStayKeywords = []string{"hotel", "airbnb", "vrbo", "stay", "accommodation", "check-in", "checkout"}
var filterIncludeKeywords = []string{
	"flight", "fly", "plane", "airport",
	"hotel", "airbnb", "vrbo", "stay", "accommodation",
	"conference", "summit", "expo", "convention", "offsite",
	"dentist", "doctor", "appointment", "haircut", "vet",
}

func shouldStage(event icalparser.Event, fc *FilterConfig) bool {
	title := strings.ToLower(event.Summary)
	location := strings.ToLower(event.Location)

	// Step 1: Built-in excludes
	excluded := false
	if !fc.DisableBuiltinExcludes {
		// Video conference URLs in location
		for _, pattern := range videoConferencePatterns {
			if strings.Contains(location, pattern) {
				excluded = true
				break
			}
		}
		// Meeting room names
		if !excluded {
			for _, pattern := range meetingRoomPatterns {
				if strings.Contains(location, pattern) {
					excluded = true
					break
				}
			}
		}
		// Room + digit pattern
		if !excluded && strings.Contains(location, "room") {
			for _, r := range location {
				if r >= '0' && r <= '9' {
					excluded = true
					break
				}
			}
		}
	}

	// Step 2: User exclude keywords
	for _, kw := range fc.ExcludeKeywords {
		lower := strings.ToLower(kw)
		if strings.Contains(title, lower) || strings.Contains(location, lower) {
			excluded = true
			break
		}
	}

	// Step 3: Built-in includes (override excludes)
	included := false
	if !fc.DisableBuiltinIncludes {
		// All-day events always pass
		if event.AllDay {
			included = true
		}
		// Multi-day events always pass
		startDate := event.StartDate()
		endDate := event.EndDate()
		if endDate > startDate {
			included = true
		}
		// Events with physical locations (non-video, non-room)
		if location != "" && !excluded {
			included = true
		}
	}

	// Step 4: User include keywords (override excludes)
	for _, kw := range fc.IncludeKeywords {
		lower := strings.ToLower(kw)
		if strings.Contains(title, lower) || strings.Contains(location, lower) {
			included = true
			break
		}
	}

	// Also check built-in travel/commitment keywords
	for _, kw := range filterIncludeKeywords {
		if strings.Contains(title, kw) {
			included = true
			break
		}
	}

	// Step 5: Default rule
	if included {
		return true
	}
	if excluded {
		return false
	}
	// If has a location that's not a video URL, include
	if location != "" {
		return true
	}
	return false
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
