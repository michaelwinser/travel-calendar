package service

import (
	"testing"
	"time"

	"github.com/user/travel-calendar/backend/internal/api"
)

func TestIsTripItEvent(t *testing.T) {
	tests := []struct {
		name        string
		description *string
		want        bool
	}{
		{
			name:        "has tripit.com in description",
			description: ptr("View and/or edit details in TripIt : https://www.tripit.com/trip/show?id=123"),
			want:        true,
		},
		{
			name:        "has tripit.com lowercase",
			description: ptr("Book via tripit.com for best rates"),
			want:        true,
		},
		{
			name:        "no tripit reference",
			description: ptr("Team meeting in conference room"),
			want:        false,
		},
		{
			name:        "nil description",
			description: nil,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := api.CalendarEvent{
				Summary:     "Test Event",
				Description: tt.description,
			}
			got := IsTripItEvent(event)
			if got != tt.want {
				t.Errorf("IsTripItEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTripItSummary(t *testing.T) {
	tests := []struct {
		name    string
		summary string
		want    *TripItSummary
		wantOk  bool
	}{
		{
			name:    "standard format",
			summary: "michaelwinser is in Brussels, Belgium from Jan 23 to Feb 3, 2026",
			want: &TripItSummary{
				Username:  "michaelwinser",
				Location:  "Brussels, Belgium",
				StartDate: time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC),
			},
			wantOk: true,
		},
		{
			name:    "multi-part location",
			summary: "user is in New York, NY, USA from Mar 1 to Mar 5, 2026",
			want: &TripItSummary{
				Username:  "user",
				Location:  "New York, NY, USA",
				StartDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
			},
			wantOk: true,
		},
		{
			name:    "year boundary - Dec to Jan",
			summary: "user is in Paris from Dec 28 to Jan 5, 2026",
			want: &TripItSummary{
				Username:  "user",
				Location:  "Paris",
				StartDate: time.Date(2026, 12, 28, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2027, 1, 5, 0, 0, 0, 0, time.UTC),
			},
			wantOk: true,
		},
		{
			name:    "without trailing comma before year",
			summary: "john is in London from Apr 10 to Apr 15 2026",
			want: &TripItSummary{
				Username:  "john",
				Location:  "London",
				StartDate: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
			},
			wantOk: true,
		},
		{
			name:    "non-matching format - regular event",
			summary: "Team meeting in Brussels",
			wantOk:  false,
		},
		{
			name:    "non-matching format - flight",
			summary: "UA57 EWR to CDG",
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := api.CalendarEvent{
				Summary: tt.summary,
			}
			got, ok := ParseTripItSummary(event)
			if ok != tt.wantOk {
				t.Errorf("ParseTripItSummary() ok = %v, wantOk %v", ok, tt.wantOk)
				return
			}
			if !tt.wantOk {
				return
			}
			if got.Username != tt.want.Username {
				t.Errorf("Username = %q, want %q", got.Username, tt.want.Username)
			}
			if got.Location != tt.want.Location {
				t.Errorf("Location = %q, want %q", got.Location, tt.want.Location)
			}
			if !got.StartDate.Equal(tt.want.StartDate) {
				t.Errorf("StartDate = %v, want %v", got.StartDate, tt.want.StartDate)
			}
			if !got.EndDate.Equal(tt.want.EndDate) {
				t.Errorf("EndDate = %v, want %v", got.EndDate, tt.want.EndDate)
			}
		})
	}
}

func TestParseTripItFlight(t *testing.T) {
	baseTime := time.Date(2026, 1, 23, 18, 5, 0, 0, time.UTC)

	tests := []struct {
		name        string
		summary     string
		description *string
		want        *TripItFlight
		wantOk      bool
	}{
		{
			name:    "standard two-letter carrier",
			summary: "UA57 EWR to CDG",
			want: &TripItFlight{
				Carrier:      "UA",
				FlightNumber: "57",
				Origin:       "EWR",
				Destination:  "CDG",
				Time:         "18:05",
			},
			wantOk: true,
		},
		{
			name:    "three-letter carrier",
			summary: "BAW123 LHR to JFK",
			want: &TripItFlight{
				Carrier:      "BAW",
				FlightNumber: "123",
				Origin:       "LHR",
				Destination:  "JFK",
				Time:         "18:05",
			},
			wantOk: true,
		},
		{
			name:    "four-digit flight number",
			summary: "DL1234 ATL to LAX",
			want: &TripItFlight{
				Carrier:      "DL",
				FlightNumber: "1234",
				Origin:       "ATL",
				Destination:  "LAX",
				Time:         "18:05",
			},
			wantOk: true,
		},
		{
			name:        "with confirmation in description",
			summary:     "AA100 JFK to LHR",
			description: ptr("Confirmation: ABC123\nTerminal 8"),
			want: &TripItFlight{
				Carrier:      "AA",
				FlightNumber: "100",
				Origin:       "JFK",
				Destination:  "LHR",
				Time:         "18:05",
				Confirmation: "ABC123",
				Notes:        "Terminal 8",
			},
			wantOk: true,
		},
		{
			name:        "with two terminals in description",
			summary:     "LH456 FRA to SFO",
			description: ptr("Terminal 1\nArrives Terminal A"),
			want: &TripItFlight{
				Carrier:      "LH",
				FlightNumber: "456",
				Origin:       "FRA",
				Destination:  "SFO",
				Time:         "18:05",
				Notes:        "Terminal 1 -> Terminal A",
			},
			wantOk: true,
		},
		{
			name:    "non-matching - lowercase",
			summary: "ua57 ewr to cdg",
			wantOk:  false,
		},
		{
			name:    "non-matching - natural language",
			summary: "Flight to Paris",
			wantOk:  false,
		},
		{
			name:    "non-matching - trip summary",
			summary: "user is in Paris from Jan 1 to Jan 5, 2026",
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := api.CalendarEvent{
				Summary:     tt.summary,
				Start:       baseTime,
				Description: tt.description,
			}
			got, ok := ParseTripItFlight(event)
			if ok != tt.wantOk {
				t.Errorf("ParseTripItFlight() ok = %v, wantOk %v", ok, tt.wantOk)
				return
			}
			if !tt.wantOk {
				return
			}
			if got.Carrier != tt.want.Carrier {
				t.Errorf("Carrier = %q, want %q", got.Carrier, tt.want.Carrier)
			}
			if got.FlightNumber != tt.want.FlightNumber {
				t.Errorf("FlightNumber = %q, want %q", got.FlightNumber, tt.want.FlightNumber)
			}
			if got.Origin != tt.want.Origin {
				t.Errorf("Origin = %q, want %q", got.Origin, tt.want.Origin)
			}
			if got.Destination != tt.want.Destination {
				t.Errorf("Destination = %q, want %q", got.Destination, tt.want.Destination)
			}
			if got.Time != tt.want.Time {
				t.Errorf("Time = %q, want %q", got.Time, tt.want.Time)
			}
			if got.Confirmation != tt.want.Confirmation {
				t.Errorf("Confirmation = %q, want %q", got.Confirmation, tt.want.Confirmation)
			}
			if got.Notes != tt.want.Notes {
				t.Errorf("Notes = %q, want %q", got.Notes, tt.want.Notes)
			}
		})
	}
}
