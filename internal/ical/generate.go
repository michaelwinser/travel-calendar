package ical

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// CalendarEvent represents an event to be written to an iCal feed.
type CalendarEvent struct {
	UID       string
	Summary   string
	Location  string
	StartDate string // YYYY-MM-DD
	EndDate   string // YYYY-MM-DD (inclusive)
	AllDay    bool
	Status    string // TENTATIVE, CONFIRMED, CANCELLED
}

// WriteCalendar writes a VCALENDAR with the given events to w.
func WriteCalendar(w io.Writer, calName string, events []CalendarEvent) error {
	fmt.Fprintf(w, "BEGIN:VCALENDAR\r\n")
	fmt.Fprintf(w, "VERSION:2.0\r\n")
	fmt.Fprintf(w, "PRODID:-//Travel Calendar//EN\r\n")
	fmt.Fprintf(w, "CALSCALE:GREGORIAN\r\n")
	fmt.Fprintf(w, "METHOD:PUBLISH\r\n")
	if calName != "" {
		fmt.Fprintf(w, "X-WR-CALNAME:%s\r\n", escapeIcalValue(calName))
	}

	now := time.Now().UTC().Format("20060102T150405Z")

	for _, e := range events {
		fmt.Fprintf(w, "BEGIN:VEVENT\r\n")
		fmt.Fprintf(w, "UID:%s\r\n", e.UID)
		fmt.Fprintf(w, "DTSTAMP:%s\r\n", now)

		if e.AllDay {
			// All-day events use VALUE=DATE. DTEND is exclusive in iCal.
			fmt.Fprintf(w, "DTSTART;VALUE=DATE:%s\r\n", dateToIcal(e.StartDate))
			// Add one day to EndDate for exclusive DTEND
			endExclusive := addOneDay(e.EndDate)
			fmt.Fprintf(w, "DTEND;VALUE=DATE:%s\r\n", dateToIcal(endExclusive))
			// Mark as free (transparent) — informational, not blocking
			fmt.Fprintf(w, "TRANSP:TRANSPARENT\r\n")
		} else {
			fmt.Fprintf(w, "DTSTART;VALUE=DATE:%s\r\n", dateToIcal(e.StartDate))
			endExclusive := addOneDay(e.EndDate)
			fmt.Fprintf(w, "DTEND;VALUE=DATE:%s\r\n", dateToIcal(endExclusive))
			fmt.Fprintf(w, "TRANSP:TRANSPARENT\r\n")
		}

		fmt.Fprintf(w, "SUMMARY:%s\r\n", escapeIcalValue(e.Summary))

		if e.Location != "" {
			fmt.Fprintf(w, "LOCATION:%s\r\n", escapeIcalValue(e.Location))
		}

		if e.Status != "" {
			fmt.Fprintf(w, "STATUS:%s\r\n", e.Status)
		}

		fmt.Fprintf(w, "END:VEVENT\r\n")
	}

	fmt.Fprintf(w, "END:VCALENDAR\r\n")
	return nil
}

func dateToIcal(date string) string {
	// Convert YYYY-MM-DD to YYYYMMDD
	return strings.ReplaceAll(date, "-", "")
}

func addOneDay(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}

func escapeIcalValue(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
