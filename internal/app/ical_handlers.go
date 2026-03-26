package app

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/michaelwinser/travel-calendar/internal/ical"
)

// HandleSharedFeed serves an iCal feed for a share link token.
func (s *ActivityServer) HandleSharedFeed(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		token = strings.TrimPrefix(r.URL.Path, "/shared/")
		token = strings.TrimSuffix(token, "/feed.ics")
	}
	if token == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	link, err := s.shareLinks.GetByToken(token)
	if err != nil || link == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Check expiry
	if link.ExpiresAt != "" {
		expiry, err := time.Parse(time.RFC3339, link.ExpiresAt)
		if err == nil && time.Now().After(expiry) {
			http.Error(w, "expired", http.StatusGone)
			return
		}
	}

	// Fetch activities with filters
	var activities []Activity
	if link.FromDate != "" && link.ToDate != "" {
		activities, err = s.store.ListRange(link.UserID, link.FromDate, link.ToDate)
	} else {
		activities, err = s.store.List(link.UserID)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Filter by trip IDs if specified
	if link.TripIDs != "" {
		tripIDSet := map[string]bool{}
		for _, id := range strings.Split(link.TripIDs, ",") {
			tripIDSet[strings.TrimSpace(id)] = true
		}
		var filtered []Activity
		for _, a := range activities {
			if tripIDSet[a.TripID] {
				filtered = append(filtered, a)
			}
		}
		activities = filtered
	}

	// Build trip map for trip-level events
	tripMap := s.buildTripMap(link.UserID)

	// Generate events
	events := activitiesToIcalEvents(activities, tripMap, link.ShowTitle)

	calName := link.Label
	if calName == "" {
		calName = "Shared Travel Calendar"
	}

	serveIcalFeed(w, calName, events)
}

// HandlePublicFeed serves an iCal feed for a public profile handle.
func (s *ActivityServer) HandlePublicFeed(w http.ResponseWriter, r *http.Request) {
	handle := r.PathValue("handle")
	if handle == "" {
		handle = strings.TrimPrefix(r.URL.Path, "/public/")
		handle = strings.TrimSuffix(handle, "/feed.ics")
	}

	p, err := s.publicProfiles.GetByHandle(handle)
	if err != nil || p == nil || !p.Enabled {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Query -1 month to +6 months
	now := time.Now()
	from := now.AddDate(0, -1, 0).Format("2006-01-02")
	to := now.AddDate(0, 6, 0).Format("2006-01-02")

	activities, err := s.store.ListRange(p.UserID, from, to)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tripMap := s.buildTripMap(p.UserID)

	// Public feed: no titles, location + type only
	events := activitiesToIcalEvents(activities, tripMap, false)

	serveIcalFeed(w, "Where is "+p.Handle+"?", events)
}

func activitiesToIcalEvents(activities []Activity, tripMap map[string]Trip, showTitle bool) []ical.CalendarEvent {
	// Group activities by trip to create trip-level events
	tripActivities := map[string][]Activity{}
	var standalone []Activity

	for _, a := range activities {
		if a.TripID != "" {
			tripActivities[a.TripID] = append(tripActivities[a.TripID], a)
		} else {
			standalone = append(standalone, a)
		}
	}

	var events []ical.CalendarEvent

	// Trip-level events: one all-day event per trip
	for tripID, acts := range tripActivities {
		trip, ok := tripMap[tripID]
		if !ok {
			// Trip not found — fall through to individual activities
			standalone = append(standalone, acts...)
			continue
		}

		// Compute trip date range and dominant location
		startDate := acts[0].StartDate
		endDate := acts[0].EndDate
		locCount := map[string]int{}
		for _, a := range acts {
			if a.StartDate < startDate {
				startDate = a.StartDate
			}
			if a.EndDate > endDate {
				endDate = a.EndDate
			}
			if a.Location != "" {
				locCount[a.Location]++
			}
		}

		// Dominant location = most frequent
		location := ""
		maxCount := 0
		for loc, count := range locCount {
			if count > maxCount {
				location = loc
				maxCount = count
			}
		}

		summary := trip.Name
		if !showTitle {
			summary = location
			if summary == "" {
				summary = "Away"
			}
		}

		events = append(events, ical.CalendarEvent{
			UID:       fmt.Sprintf("trip-%s@travel-calendar", tripID),
			Summary:   summary,
			Location:  location,
			StartDate: startDate,
			EndDate:   endDate,
			AllDay:    true,
			Status:    "CONFIRMED",
		})
	}

	// Standalone activity events
	for _, a := range standalone {
		summary := a.Location
		if showTitle {
			summary = a.Title
		}
		if summary == "" {
			summary = a.Type
		}

		events = append(events, ical.CalendarEvent{
			UID:       fmt.Sprintf("activity-%s@travel-calendar", a.Key),
			Summary:   summary,
			Location:  a.Location,
			StartDate: a.StartDate,
			EndDate:   a.EndDate,
			AllDay:    true,
			Status:    "CONFIRMED",
		})
	}

	return events
}

func serveIcalFeed(w http.ResponseWriter, calName string, events []ical.CalendarEvent) {
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Content-Disposition", "inline; filename=calendar.ics")
	ical.WriteCalendar(w, calName, events)
}
