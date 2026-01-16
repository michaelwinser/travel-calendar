package service

import (
	"testing"

	"github.com/user/travel-calendar/backend/internal/api"
)

// boolPtr is a helper to create a pointer to a bool.
func boolPtr(b bool) *bool {
	return &b
}

func TestClassifyEvent(t *testing.T) {
	tests := []struct {
		name         string
		event        api.CalendarEvent
		wantClass    EventClassification
		wantItemType string
		wantSource   string
	}{
		// TripIt events
		{
			name: "TripIt summary -> trip level",
			event: api.CalendarEvent{
				Summary:     "user is in Paris from Jan 1 to Jan 5, 2026",
				AllDay:      boolPtr(true),
				Description: ptr("https://tripit.com/trip/123"),
			},
			wantClass:  ClassificationTripLevel,
			wantSource: "tripit",
		},
		{
			name: "TripIt flight -> item level flight",
			event: api.CalendarEvent{
				Summary:     "UA123 JFK to LAX",
				Description: ptr("View on tripit.com"),
			},
			wantClass:    ClassificationItemLevel,
			wantItemType: "flight",
			wantSource:   "tripit",
		},
		// All-day events
		{
			name: "all-day with location -> trip level",
			event: api.CalendarEvent{
				Summary:  "Brussels Conference",
				Location: ptr("Brussels, Belgium"),
				AllDay:   boolPtr(true),
			},
			wantClass:  ClassificationTripLevel,
			wantSource: "google",
		},
		{
			name: "all-day without location -> skip",
			event: api.CalendarEvent{
				Summary: "Company Holiday",
				AllDay:  boolPtr(true),
			},
			wantClass:  ClassificationSkip,
			wantSource: "google",
		},
		// Timed events with location
		{
			name: "timed event with hotel keyword -> hotel item",
			event: api.CalendarEvent{
				Summary:  "Hotel check-in",
				Location: ptr("Marriott Brussels"),
			},
			wantClass:    ClassificationItemLevel,
			wantItemType: "hotel",
			wantSource:   "google",
		},
		{
			name: "timed event with flight keyword -> flight item",
			event: api.CalendarEvent{
				Summary:  "Flight to Paris",
				Location: ptr("JFK Airport"),
			},
			wantClass:    ClassificationItemLevel,
			wantItemType: "flight",
			wantSource:   "google",
		},
		{
			name: "timed event with flight code -> flight item",
			event: api.CalendarEvent{
				Summary:  "AA100 departure",
				Location: ptr("Terminal 8"),
			},
			wantClass:    ClassificationItemLevel,
			wantItemType: "flight",
			wantSource:   "google",
		},
		{
			name: "timed event with train keyword -> train item",
			event: api.CalendarEvent{
				Summary:  "Eurostar to London",
				Location: ptr("Gare du Nord"),
			},
			wantClass:    ClassificationItemLevel,
			wantItemType: "train",
			wantSource:   "google",
		},
		{
			name: "timed event with drive keyword -> drive item",
			event: api.CalendarEvent{
				Summary:  "Car rental pickup",
				Location: ptr("Hertz SFO"),
			},
			wantClass:    ClassificationItemLevel,
			wantItemType: "drive",
			wantSource:   "google",
		},
		{
			name: "timed event with generic location -> event item",
			event: api.CalendarEvent{
				Summary:  "Dinner with team",
				Location: ptr("Restaurant ABC, Paris"),
			},
			wantClass:    ClassificationItemLevel,
			wantItemType: "event",
			wantSource:   "google",
		},
		// Skip cases
		{
			name: "virtual meeting URL -> skip",
			event: api.CalendarEvent{
				Summary:  "Team sync",
				Location: ptr("https://zoom.us/j/123"),
			},
			wantClass:  ClassificationSkip,
			wantSource: "google",
		},
		{
			name: "meeting room -> skip",
			event: api.CalendarEvent{
				Summary:  "Planning meeting",
				Location: ptr("US-NYC-42W-3-A-Maple (8) [GVC]"),
			},
			wantClass:  ClassificationSkip,
			wantSource: "google",
		},
		{
			name: "no location no keywords -> skip",
			event: api.CalendarEvent{
				Summary: "Team standup",
			},
			wantClass:  ClassificationSkip,
			wantSource: "google",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyEvent(tt.event)
			if got.Classification != tt.wantClass {
				t.Errorf("Classification = %v, want %v", got.Classification, tt.wantClass)
			}
			if tt.wantItemType != "" && got.ItemType != tt.wantItemType {
				t.Errorf("ItemType = %q, want %q", got.ItemType, tt.wantItemType)
			}
			if got.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", got.Source, tt.wantSource)
			}
		})
	}
}

func TestInferItemType(t *testing.T) {
	tests := []struct {
		summary string
		want    string
	}{
		// Flight indicators
		{"Flight to Paris", "flight"},
		{"AA100 departure", "flight"},
		{"UA57 check-in", "flight"},
		{"Catching my flight", "flight"},

		// Hotel indicators
		{"Hotel check-in", "hotel"},
		{"Marriott checkin", "hotel"},
		{"Check-in at Hilton", "hotel"},
		{"Hotel checkout", "hotel"},
		{"Accommodation booking", "hotel"},

		// Train indicators
		{"Train to London", "train"},
		{"Eurostar departure", "train"},
		{"Amtrak to NYC", "train"},
		{"TGV to Lyon", "train"},
		{"Thalys to Amsterdam", "train"},

		// Drive indicators
		{"Drive to airport", "drive"},
		{"Car rental pickup", "drive"},
		{"Rental car return", "drive"},
		{"Hertz pickup", "drive"}, // "pickup" triggers drive

		// Default to event
		{"Dinner reservation", "event"},
		{"Team meeting", "event"},
		{"Conference session", "event"},
	}

	for _, tt := range tests {
		t.Run(tt.summary, func(t *testing.T) {
			event := api.CalendarEvent{Summary: tt.summary}
			got := inferItemType(event)
			if got != tt.want {
				t.Errorf("inferItemType(%q) = %q, want %q", tt.summary, got, tt.want)
			}
		})
	}
}
