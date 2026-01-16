package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/user/travel-calendar/backend/internal/api"
)

// TripIt detection patterns.
var (
	// Pattern: "michaelwinser is in Brussels, Belgium from Jan 23 to Feb 3, 2026"
	tripItSummaryPattern = regexp.MustCompile(
		`(?i)^(\w+)\s+is\s+in\s+(.+?)\s+from\s+(\w+\s+\d+)\s+to\s+(\w+\s+\d+),?\s*(\d{4})$`)

	// Pattern: "UA57 EWR to CDG"
	tripItFlightPattern = regexp.MustCompile(
		`^([A-Z]{2,3})(\d{1,4})\s+([A-Z]{3})\s+to\s+([A-Z]{3})$`)

	// Confirmation pattern in description
	confirmationPattern = regexp.MustCompile(`(?i)confirmation[:\s]+([A-Z0-9]+)`)

	// Terminal pattern in description
	terminalPattern = regexp.MustCompile(`(?i)terminal\s+(\w+)`)
)

// TripIt marker in description.
const tripItMarker = "tripit.com"

// TripItSummary represents parsed TripIt all-day summary event.
type TripItSummary struct {
	Username  string
	Location  string
	StartDate time.Time
	EndDate   time.Time
}

// TripItFlight represents parsed TripIt flight segment event.
type TripItFlight struct {
	Carrier      string
	FlightNumber string
	Origin       string
	Destination  string
	Date         time.Time
	Time         string // HH:MM format
	Confirmation string
	Notes        string // Terminal info, etc.
}

// IsTripItEvent returns true if the event appears to be from TripIt.
func IsTripItEvent(event api.CalendarEvent) bool {
	if event.Description != nil {
		return strings.Contains(strings.ToLower(*event.Description), tripItMarker)
	}
	return false
}

// ParseTripItSummary attempts to parse a TripIt all-day summary event.
func ParseTripItSummary(event api.CalendarEvent) (*TripItSummary, bool) {
	matches := tripItSummaryPattern.FindStringSubmatch(event.Summary)
	if matches == nil {
		return nil, false
	}

	// Parse dates - matches[3]="Jan 23", matches[4]="Feb 3", matches[5]="2026"
	year := matches[5]
	startStr := matches[3] + " " + year
	endStr := matches[4] + " " + year

	start, err := time.Parse("Jan 2 2006", startStr)
	if err != nil {
		// Try full month name
		start, err = time.Parse("January 2 2006", startStr)
		if err != nil {
			return nil, false
		}
	}

	end, err := time.Parse("Jan 2 2006", endStr)
	if err != nil {
		// Try full month name
		end, err = time.Parse("January 2 2006", endStr)
		if err != nil {
			return nil, false
		}
	}

	// Handle year boundary (e.g., Dec 28 to Jan 5)
	if end.Before(start) {
		end = end.AddDate(1, 0, 0)
	}

	return &TripItSummary{
		Username:  matches[1],
		Location:  matches[2],
		StartDate: start,
		EndDate:   end,
	}, true
}

// ParseTripItFlight attempts to parse a TripIt flight segment event.
func ParseTripItFlight(event api.CalendarEvent) (*TripItFlight, bool) {
	matches := tripItFlightPattern.FindStringSubmatch(event.Summary)
	if matches == nil {
		return nil, false
	}

	flight := &TripItFlight{
		Carrier:      matches[1],
		FlightNumber: matches[2],
		Origin:       matches[3],
		Destination:  matches[4],
		Date:         event.Start,
		Time:         event.Start.Format("15:04"),
	}

	// Parse description for additional details
	if event.Description != nil {
		flight.Notes, flight.Confirmation = parseTripItFlightDescription(*event.Description)
	}

	return flight, true
}

// parseTripItFlightDescription extracts confirmation and terminal info from description.
func parseTripItFlightDescription(desc string) (notes, confirmation string) {
	// Look for confirmation pattern
	if m := confirmationPattern.FindStringSubmatch(desc); m != nil {
		confirmation = m[1]
	}

	// Look for terminal info
	terminals := terminalPattern.FindAllStringSubmatch(desc, -1)
	if len(terminals) >= 2 {
		notes = fmt.Sprintf("Terminal %s -> Terminal %s", terminals[0][1], terminals[1][1])
	} else if len(terminals) == 1 {
		notes = fmt.Sprintf("Terminal %s", terminals[0][1])
	}

	return notes, confirmation
}
