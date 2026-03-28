package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	icalparser "github.com/michaelwinser/travel-calendar/internal/ical"
)

// FilterConfig controls how staged events are categorized.
type FilterConfig struct {
	HidePatterns    []string `json:"hidePatterns,omitempty"`
	SelectPatterns  []string `json:"selectPatterns,omitempty"`
	// Legacy fields for backward compat
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
	Hidden   int `json:"hidden"`
	Selected int `json:"selected"`
	Errors   int `json:"errors"`
}

// SyncSource fetches events from a source, stages ALL of them, then applies
// global filters to set initial state (hidden vs new) and selection.
func SyncSource(source *ImportSource, stagedStore *StagedEventStore, activityStore *ActivityStore, configs *UserConfigStore) (*SyncResult, error) {
	result := &SyncResult{}

	// Fetch events
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

	// Time window: -30 to +360 days
	now := time.Now()
	windowStart := now.AddDate(0, 0, -30).Format("2006-01-02")
	windowEnd := now.AddDate(0, 0, 360).Format("2006-01-02")

	// Stage ALL events (no content filtering — filters set state, not entry)
	for _, event := range events {
		startDate := event.StartDate()
		endDate := event.EndDate()

		// Time window is the only gate
		if endDate < windowStart || startDate > windowEnd {
			continue
		}

		actType := inferActivityType(event)

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

				// Propagate to linked activity if imported
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
			// Create new staged event (state will be set by ApplyFilters)
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

	// Apply global filters to set initial state on new events
	filters := resolveGlobalFilters(configs, source.UserID)
	filterResult := applyFiltersToUser(source.UserID, filters, stagedStore)
	result.Hidden = filterResult.Hidden
	result.Selected = filterResult.Selected

	// Update source last sync time
	source.LastSyncAt = time.Now().Format(time.RFC3339)

	return result, nil
}

// ApplyFiltersResult summarizes what changed when filters were applied.
type ApplyFiltersResult struct {
	Hidden   int `json:"hidden"`
	Unhidden int `json:"unhidden"`
	Selected int `json:"selected"`
}

// applyFiltersToUser evaluates all staged events for a user against the given filters.
func applyFiltersToUser(userID string, filters []Filter, stagedStore *StagedEventStore) *ApplyFiltersResult {
	result := &ApplyFiltersResult{}

	events, err := stagedStore.ListByUser(userID, "")
	if err != nil {
		return result
	}

	activeHidePatterns := []string{}
	disabledHidePatterns := []string{}
	activeSelectPatterns := []string{}

	for _, f := range filters {
		if f.Type == "hide" || f.Type == "exclude" {
			if f.Enabled {
				activeHidePatterns = append(activeHidePatterns, strings.ToLower(f.Pattern))
			} else {
				disabledHidePatterns = append(disabledHidePatterns, strings.ToLower(f.Pattern))
			}
		}
		if (f.Type == "select" || f.Type == "include") && f.Enabled {
			activeSelectPatterns = append(activeSelectPatterns, strings.ToLower(f.Pattern))
		}
	}

	for _, e := range events {
		if e.State == "imported" {
			continue
		}

		title := strings.ToLower(e.Title)
		location := strings.ToLower(e.Location)
		matchesHide := matchesAny(title, location, activeHidePatterns)
		matchesDisabledHide := matchesAny(title, location, disabledHidePatterns)

		if e.Location != "" && isURLOnly(strings.ToLower(e.Location)) {
			matchesHide = true
		}

		if e.State == "new" && matchesHide {
			e.State = "hidden"
			stagedStore.Update(&e)
			result.Hidden++
		} else if e.State == "hidden" && matchesDisabledHide && !matchesHide {
			e.State = "new"
			stagedStore.Update(&e)
			result.Unhidden++
		}

		if e.State == "new" && matchesAny(title, location, activeSelectPatterns) {
			result.Selected++
		}
	}

	return result
}

// resolveGlobalFilters returns the user's global filters (built-in + user-defined).
func resolveGlobalFilters(configs *UserConfigStore, userID string) []Filter {
	filters := LoadDefaultFilters()

	if configs == nil {
		return filters
	}

	configJSON := configs.Get(userID, "import_filters")
	if configJSON == "" {
		return filters
	}

	var userFilters []Filter
	if err := json.Unmarshal([]byte(configJSON), &userFilters); err != nil {
		return filters
	}

	builtinByPattern := map[string]int{}
	for i, f := range filters {
		builtinByPattern[f.Pattern] = i
	}
	for _, uf := range userFilters {
		if uf.Builtin {
			if idx, ok := builtinByPattern[uf.Pattern]; ok {
				filters[idx].Enabled = uf.Enabled
			}
		} else {
			filters = append(filters, uf)
		}
	}

	return filters
}

func matchesAny(title, location string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(title, p) || strings.Contains(location, p) {
			return true
		}
	}
	return false
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

// isURLOnly returns true if the string is just a URL with no meaningful text.
// Map URLs are excluded — they indicate a real physical location.
func isURLOnly(s string) bool {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return false
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "google.com/maps") ||
		strings.Contains(lower, "maps.app.goo.gl") ||
		strings.Contains(lower, "maps.apple.com") ||
		strings.Contains(lower, "maps.bing.com") {
		return false
	}
	return true
}

// inferActivityType guesses the activity type from event content.
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
	for _, kw := range []string{"hotel", "airbnb", "vrbo", "stay", "accommodation"} {
		if strings.Contains(title, kw) {
			return TypeStay
		}
	}

	if event.StartDate() != event.EndDate() && event.Location != "" {
		return TypeVacation
	}
	if event.Location != "" {
		return TypeCommitment
	}
	return TypeStay
}
