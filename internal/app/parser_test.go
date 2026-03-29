package app

import (
	"testing"
	"time"
)

// Reference date for all parser tests: March 22, 2026
var testToday = time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)

func TestParse_UC2001_SimpleEventWithLocation(t *testing.T) {
	r := Parse("FOSDEM Jan 22 - Feb 3 in Brussels", testToday)
	assertTitle(t, r, "FOSDEM")
	assertStartDate(t, r, "2027-01-22") // Jan is past → next year
	assertEndDate(t, r, "2027-02-03")
	assertLocation(t, r, "Brussels")
}

func TestParse_UC2002_FlightWithRoute(t *testing.T) {
	r := Parse("Flight UA 19 on Mar 20 from MXP to EWR", testToday)
	assertTitle(t, r, "Flight UA 19")
	assertStartDate(t, r, "2027-03-20") // Mar 20 is before Mar 22 → next year
	assertType(t, r, TypeTravel)
	assertLocation(t, r, "MXP → EWR")
}

func TestParse_UC2003_SingleDayWithLocationKeyword(t *testing.T) {
	r := Parse("Dentist on Apr 5 at Home", testToday)
	assertTitle(t, r, "Dentist")
	assertStartDate(t, r, "2026-04-05") // Apr is ahead → this year
	assertEndDate(t, r, "2026-04-05")
	assertType(t, r, TypeCommitment)
	assertLocation(t, r, "Home")
}

func TestParse_UC2004_DateRangeWithFromTo(t *testing.T) {
	r := Parse("Team offsite from Mar 10 to Mar 14 in NYC", testToday)
	assertTitle(t, r, "Team offsite")
	assertStartDate(t, r, "2027-03-10") // Mar 10 is past → next year
	assertEndDate(t, r, "2027-03-14")
	assertLocation(t, r, "NYC")
}

func TestParse_UC2005_ExplicitYear(t *testing.T) {
	r := Parse("Christmas vacation Dec 20 2026 - Jan 3 2027", testToday)
	assertTitle(t, r, "Christmas vacation")
	assertStartDate(t, r, "2026-12-20")
	assertEndDate(t, r, "2027-01-03")
	assertType(t, r, TypeVacation)
}

func TestParse_UC2006_MinimalInput(t *testing.T) {
	r := Parse("meeting tomorrow", testToday)
	assertTitle(t, r, "meeting")
	assertType(t, r, TypeCommitment) // "meeting" keyword
	assertStartDate(t, r, "2026-03-23") // tomorrow resolved
}

func TestParse_UC2007_YearInferencePastMonth(t *testing.T) {
	r := Parse("Jan 15", testToday)
	assertStartDate(t, r, "2027-01-15")
}

func TestParse_UC2008_YearInferenceFutureMonth(t *testing.T) {
	r := Parse("Apr 10", testToday)
	assertStartDate(t, r, "2026-04-10")
}

func TestParse_UC2009_UnparsableInput(t *testing.T) {
	r := Parse("stuff and things", testToday)
	assertTitle(t, r, "stuff and things")
	if r.StartDate != nil {
		t.Errorf("expected no start date, got %s", r.StartDate.Format("2006-01-02"))
	}
	if r.Location != "" {
		t.Errorf("expected no location, got %q", r.Location)
	}
	if r.Confidence["title"] != ConfHigh {
		t.Errorf("expected title confidence high, got %q", r.Confidence["title"])
	}
	if r.Confidence["type"] != ConfLow {
		t.Errorf("expected type confidence low, got %q", r.Confidence["type"])
	}
}

func TestParse_UC2010_FromToAsLocations(t *testing.T) {
	r := Parse("Train from London to Paris on May 5", testToday)
	assertTitle(t, r, "Train")
	assertType(t, r, TypeTravel)
	assertStartDate(t, r, "2026-05-05")
	assertLocation(t, r, "London → Paris")
}

// --- Issue #71 test cases ---

func TestParse_71_BrusselsConference(t *testing.T) {
	r := Parse("Brussels conference Feb 3", testToday)
	assertLocation(t, r, "Brussels")
	assertStartDate(t, r, "2027-02-03")
	// Title should be "conference", not "Brussels conference"
	assertTitle(t, r, "conference")
}

func TestParse_71_FlightEWRtoCDG(t *testing.T) {
	r := Parse("Flight EWR to CDG March 5", testToday)
	assertType(t, r, TypeTravel)
	assertStartDate(t, r, "2027-03-05")
	assertLocation(t, r, "EWR → CDG")
}

func TestParse_71_TripToLondon(t *testing.T) {
	r := Parse("Trip to London March 12-16", testToday)
	assertLocation(t, r, "London")
	assertStartDate(t, r, "2027-03-12")
	assertEndDate(t, r, "2027-03-16")
}

func TestParse_71_OffsiteMilan(t *testing.T) {
	r := Parse("EF Security Team Offsite Milan Mar 17-19", testToday)
	assertLocation(t, r, "Milan")
	assertStartDate(t, r, "2027-03-17")
	assertEndDate(t, r, "2027-03-19")
	assertTitle(t, r, "EF Security Team Offsite")
}

func TestParse_71_DriveHomeToEWR(t *testing.T) {
	r := Parse("Drive Home to EWR March 12", testToday)
	assertType(t, r, TypeTravel)
	assertLocation(t, r, "Home → EWR")
	assertStartDate(t, r, "2027-03-12")
}

func TestParse_71_DentistAppointment(t *testing.T) {
	r := Parse("Dentist appointment March 3", testToday)
	assertType(t, r, TypeCommitment)
	assertStartDate(t, r, "2027-03-03")
	assertTitle(t, r, "Dentist appointment")
}

func TestParse_71_OSSNAMinneapolis(t *testing.T) {
	r := Parse("OSSNA in Minneapolis from May 18 to 21", testToday)
	assertTitle(t, r, "OSSNA")
	assertLocation(t, r, "Minneapolis")
	assertStartDate(t, r, "2026-05-18")
	assertEndDate(t, r, "2026-05-21")
}

func TestParse_71_PyCon(t *testing.T) {
	r := Parse("PyCon May 14 - May 15 in Long Beach, CA", testToday)
	assertTitle(t, r, "PyCon")
	assertLocation(t, r, "Long Beach CA")  // comma stripped by tokenizer
	assertStartDate(t, r, "2026-05-14")
	assertEndDate(t, r, "2026-05-15")
}

func TestParse_71_NumericDate(t *testing.T) {
	r := Parse("UA 16 from EWR to CDG on 4/12", testToday)
	assertTitle(t, r, "UA 16")
	assertType(t, r, TypeTravel)
	assertLocation(t, r, "EWR → CDG")
	assertStartDate(t, r, "2026-04-12")
}

func TestParse_71_VenueWithAt(t *testing.T) {
	r := Parse("Bob's birthday party at The Shed", testToday)
	assertTitle(t, r, "Bob's birthday party")
	assertLocation(t, r, "The Shed")
}

func TestParse_71_RelativeDayOfWeek(t *testing.T) {
	// testToday is March 22, 2026 (Sunday)
	r := Parse("Bob's birthday party at The Shed on Tuesday", testToday)
	assertTitle(t, r, "Bob's birthday party")
	assertLocation(t, r, "The Shed")
	assertStartDate(t, r, "2026-03-24") // next Tuesday
}

func TestParse_71_NextFriday(t *testing.T) {
	// testToday is March 22, 2026 (Sunday)
	r := Parse("Bob's birthday party at The Shed next Friday", testToday)
	assertTitle(t, r, "Bob's birthday party")
	assertLocation(t, r, "The Shed")
	assertStartDate(t, r, "2026-03-27") // next Friday
}

func TestParse_71_Tomorrow(t *testing.T) {
	r := Parse("Dentist tomorrow", testToday)
	assertTitle(t, r, "Dentist")
	assertStartDate(t, r, "2026-03-23")
	assertType(t, r, TypeCommitment)
}

func TestParse_71_SummitAtGoogleInBrussels(t *testing.T) {
	r := Parse("Package manager summit at Google in Brussels", testToday)
	assertTitle(t, r, "Package manager summit")
	assertLocation(t, r, "Google in Brussels")
	assertType(t, r, TypeConference) // "summit" keyword
}

func TestParse_71_HangOutInLondon(t *testing.T) {
	r := Parse("Hang out with Xander in London from Mar 13 2026 to Mar 15 2026", testToday)
	assertTitle(t, r, "Hang out with Xander")
	assertLocation(t, r, "London")
	assertStartDate(t, r, "2026-03-13")
	assertEndDate(t, r, "2026-03-15")
}

func TestParse_71_MeetingAtVenue(t *testing.T) {
	r := Parse("Alpha-Omega meeting with Bob at the Google NYC Office", testToday)
	assertTitle(t, r, "Alpha-Omega meeting with Bob")
	assertLocation(t, r, "the Google NYC Office")
}

// --- Helpers ---

func assertTitle(t *testing.T, r ParsedResult, expected string) {
	t.Helper()
	if r.Title != expected {
		t.Errorf("title: expected %q, got %q", expected, r.Title)
	}
}

func assertType(t *testing.T, r ParsedResult, expected string) {
	t.Helper()
	if r.Type != expected {
		t.Errorf("type: expected %q, got %q", expected, r.Type)
	}
}

func assertStartDate(t *testing.T, r ParsedResult, expected string) {
	t.Helper()
	if r.StartDate == nil {
		t.Errorf("startDate: expected %s, got nil", expected)
		return
	}
	got := r.StartDate.Format("2006-01-02")
	if got != expected {
		t.Errorf("startDate: expected %s, got %s", expected, got)
	}
}

func assertEndDate(t *testing.T, r ParsedResult, expected string) {
	t.Helper()
	if r.EndDate == nil {
		t.Errorf("endDate: expected %s, got nil", expected)
		return
	}
	got := r.EndDate.Format("2006-01-02")
	if got != expected {
		t.Errorf("endDate: expected %s, got %s", expected, got)
	}
}

func assertLocation(t *testing.T, r ParsedResult, expected string) {
	t.Helper()
	if r.Location != expected {
		t.Errorf("location: expected %q, got %q", expected, r.Location)
	}
}
