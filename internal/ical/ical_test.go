package ical

import (
	"strings"
	"testing"
)

const testCal = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
UID:event-1@example.com
SUMMARY:FOSDEM 2026
DTSTART;VALUE=DATE:20260131
DTEND;VALUE=DATE:20260202
LOCATION:Brussels\, Belgium
DESCRIPTION:Free and Open Source Software conference
END:VEVENT
BEGIN:VEVENT
UID:event-2@example.com
SUMMARY:Team Offsite
DTSTART:20260315T090000Z
DTEND:20260317T170000Z
LOCATION:Milan
END:VEVENT
BEGIN:VEVENT
UID:event-3@example.com
SUMMARY:Dentist
DTSTART;VALUE=DATE:20260320
END:VEVENT
END:VCALENDAR`

func TestParse(t *testing.T) {
	events, err := Parse(strings.NewReader(testCal))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	// Event 1: all-day, multi-day
	e := events[0]
	if e.Summary != "FOSDEM 2026" {
		t.Errorf("summary = %q, want FOSDEM 2026", e.Summary)
	}
	if !e.AllDay {
		t.Error("FOSDEM should be all-day")
	}
	if e.Start.Format("2006-01-02") != "2026-01-31" {
		t.Errorf("start = %s, want 2026-01-31", e.Start.Format("2006-01-02"))
	}
	if e.End.Format("2006-01-02") != "2026-02-02" {
		t.Errorf("end = %s, want 2026-02-02", e.End.Format("2006-01-02"))
	}
	if e.Location != "Brussels, Belgium" {
		t.Errorf("location = %q, want 'Brussels, Belgium'", e.Location)
	}
	if e.UID != "event-1@example.com" {
		t.Errorf("uid = %q", e.UID)
	}

	// Event 2: timed event
	e2 := events[1]
	if e2.Summary != "Team Offsite" {
		t.Errorf("summary = %q", e2.Summary)
	}
	if e2.AllDay {
		t.Error("Team Offsite should not be all-day")
	}

	// Event 3: single day, no DTEND
	e3 := events[2]
	if e3.Start.Format("2006-01-02") != "2026-03-20" {
		t.Errorf("start = %s", e3.Start.Format("2006-01-02"))
	}
	if e3.End.Format("2006-01-02") != "2026-03-20" {
		t.Errorf("end should default to start, got %s", e3.End.Format("2006-01-02"))
	}
}

func TestParseLongDescription(t *testing.T) {
	cal := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nSUMMARY:Test\r\nDESCRIPTION:This is a long\r\n  description that spans\r\n  multiple lines\r\nDTSTART;VALUE=DATE:20260101\r\nEND:VEVENT\r\nEND:VCALENDAR"
	events, err := Parse(strings.NewReader(cal))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	want := "This is a long description that spans multiple lines"
	if events[0].Description != want {
		t.Errorf("description = %q, want %q", events[0].Description, want)
	}
}
