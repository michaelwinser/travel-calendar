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
	if e.StartDate() != "2026-01-31" {
		t.Errorf("start = %s, want 2026-01-31", e.StartDate())
	}
	// EndDate adjusts for exclusive DTEND: 2026-02-02 → 2026-02-01
	if e.EndDate() != "2026-02-01" {
		t.Errorf("end = %s, want 2026-02-01", e.EndDate())
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
	if e3.StartDate() != "2026-03-20" {
		t.Errorf("start = %s", e3.StartDate())
	}
	if e3.EndDate() != "2026-03-20" {
		t.Errorf("end should default to start, got %s", e3.EndDate())
	}
}

func TestTimedEventWithTimezone(t *testing.T) {
	// An evening event in America/New_York — should not cross midnight
	cal := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nSUMMARY:Hockey Game\r\nDTSTART;TZID=America/New_York:20260310T190000\r\nDTEND;TZID=America/New_York:20260310T220000\r\nLOCATION:Centre Bell\r\nEND:VEVENT\r\nEND:VCALENDAR"
	events, err := Parse(strings.NewReader(cal))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	e := events[0]
	if e.StartDate() != "2026-03-10" {
		t.Errorf("start = %s, want 2026-03-10", e.StartDate())
	}
	if e.EndDate() != "2026-03-10" {
		t.Errorf("end = %s, want 2026-03-10 (same day)", e.EndDate())
	}
	if e.Timezone != "America/New_York" {
		t.Errorf("timezone = %q, want America/New_York", e.Timezone)
	}
}

func TestTimedEventUTC(t *testing.T) {
	// An event stored in UTC that's 7pm-10pm in NYC (00:00-03:00 UTC next day)
	// Should still resolve to the correct local date
	cal := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nSUMMARY:Late Event\r\nDTSTART:20260311T000000Z\r\nDTEND:20260311T030000Z\r\nEND:VEVENT\r\nEND:VCALENDAR"
	events, err := Parse(strings.NewReader(cal))
	if err != nil {
		t.Fatal(err)
	}
	e := events[0]
	// Without timezone info, both dates should be March 11 (UTC)
	if e.StartDate() != "2026-03-11" {
		t.Errorf("start = %s, want 2026-03-11", e.StartDate())
	}
	if e.EndDate() != "2026-03-11" {
		t.Errorf("end = %s, want 2026-03-11", e.EndDate())
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
