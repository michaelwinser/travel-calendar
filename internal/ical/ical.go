// Package ical provides a minimal iCalendar (RFC 5545) parser for VEVENT extraction.
package ical

import (
	"bufio"
	"io"
	"strings"
	"time"
)

// Event represents a parsed VEVENT.
type Event struct {
	UID         string
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	AllDay      bool
	Timezone    string // IANA timezone from DTSTART, if specified
}

// StartDate returns the start date as YYYY-MM-DD in the event's local timezone.
func (e Event) StartDate() string {
	return e.localDate(e.Start)
}

// EndDate returns the end date as YYYY-MM-DD in the event's local timezone.
// For all-day events, adjusts for the exclusive DTEND convention.
// For short timed events (< 12 hours), uses the start date to avoid
// midnight-crossing artifacts from UTC conversion.
func (e Event) EndDate() string {
	end := e.End
	if end.IsZero() {
		end = e.Start
	}
	if e.AllDay && end.After(e.Start) {
		// All-day DTEND is exclusive — subtract a day for inclusive display
		end = end.AddDate(0, 0, -1)
		return e.localDate(end)
	}
	// For short timed events, the end date should match the start date.
	// A 7pm-10pm event stored as 23:00Z-01:00Z+1 shouldn't span two days.
	if !e.AllDay && end.Sub(e.Start) < 12*time.Hour {
		return e.StartDate()
	}
	return e.localDate(end)
}

func (e Event) localDate(t time.Time) string {
	if e.Timezone != "" {
		if loc, err := time.LoadLocation(e.Timezone); err == nil {
			return t.In(loc).Format("2006-01-02")
		}
	}
	// Fallback: if no timezone, use the raw time's date
	// (which may be UTC — better than crossing midnight)
	return t.Format("2006-01-02")
}

// Parse reads iCalendar data and extracts all VEVENTs.
func Parse(r io.Reader) ([]Event, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	var events []Event
	var current *Event
	var propName, propValue string

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")

		// Handle line continuations (lines starting with space or tab)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			propValue += line[1:]
			continue
		}

		// Process the previous property if we were in an event
		if current != nil && propName != "" {
			applyProperty(current, propName, propValue)
		}

		// Parse new property
		if idx := strings.Index(line, ":"); idx >= 0 {
			propName = line[:idx]
			propValue = line[idx+1:]
		} else {
			propName = line
			propValue = ""
		}

		// Handle structural elements
		switch {
		case strings.EqualFold(propName, "BEGIN") && strings.EqualFold(propValue, "VEVENT"):
			current = &Event{}
			propName = ""
		case strings.EqualFold(propName, "END") && strings.EqualFold(propValue, "VEVENT"):
			if current != nil && !current.Start.IsZero() {
				if current.End.IsZero() {
					current.End = current.Start
				}
				events = append(events, *current)
			}
			current = nil
			propName = ""
		}
	}

	return events, scanner.Err()
}

func applyProperty(e *Event, name, value string) {
	// Strip parameters from property name: "DTSTART;VALUE=DATE" → "DTSTART"
	baseName := name
	params := ""
	if idx := strings.Index(name, ";"); idx >= 0 {
		baseName = name[:idx]
		params = name[idx+1:]
	}

	switch strings.ToUpper(baseName) {
	case "UID":
		e.UID = value
	case "SUMMARY":
		e.Summary = unescapeIcal(value)
	case "DESCRIPTION":
		e.Description = unescapeIcal(value)
	case "LOCATION":
		e.Location = unescapeIcal(value)
	case "DTSTART":
		t, allDay := parseIcalTime(value, params)
		e.Start = t
		e.AllDay = allDay
		// Extract timezone from params: TZID=America/New_York
		if tz := extractTZID(params); tz != "" {
			e.Timezone = tz
		}
	case "DTEND":
		t, _ := parseIcalTime(value, params)
		e.End = t
	}
}

func parseIcalTime(value, params string) (time.Time, bool) {
	value = strings.TrimSpace(value)

	// Check for VALUE=DATE parameter (all-day event)
	isDate := strings.Contains(strings.ToUpper(params), "VALUE=DATE")

	// Date only: 20260401
	if len(value) == 8 && !strings.Contains(value, "T") {
		t, err := time.Parse("20060102", value)
		if err == nil {
			return t, true
		}
	}

	// Date-time with UTC: 20260401T120000Z
	if strings.HasSuffix(value, "Z") {
		t, err := time.Parse("20060102T150405Z", value)
		if err == nil {
			return t, isDate
		}
	}

	// Date-time without timezone: 20260401T120000
	t, err := time.Parse("20060102T150405", value)
	if err == nil {
		return t, isDate
	}

	return time.Time{}, isDate
}

func extractTZID(params string) string {
	for _, part := range strings.Split(params, ";") {
		if strings.HasPrefix(strings.ToUpper(part), "TZID=") {
			return part[5:]
		}
	}
	return ""
}

func unescapeIcal(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\N", "\n")
	s = strings.ReplaceAll(s, "\\,", ",")
	s = strings.ReplaceAll(s, "\\;", ";")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}
