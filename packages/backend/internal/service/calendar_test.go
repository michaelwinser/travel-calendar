package service

import (
	"testing"

	"github.com/user/travel-calendar/backend/internal/api"
)

func TestIsTravelRelatedEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    api.CalendarEvent
		expected bool
	}{
		{
			name: "physical location is travel-related",
			event: api.CalendarEvent{
				Summary:  "Team offsite",
				Location: ptr("123 Main St, San Francisco, CA"),
			},
			expected: true,
		},
		{
			name: "zoom URL in location is NOT travel-related",
			event: api.CalendarEvent{
				Summary:  "Weekly sync",
				Location: ptr("https://zoom.us/j/123456789"),
			},
			expected: false,
		},
		{
			name: "teams URL in location is NOT travel-related",
			event: api.CalendarEvent{
				Summary:  "Project review",
				Location: ptr("https://teams.microsoft.com/l/meetup-join/123"),
			},
			expected: false,
		},
		{
			name: "google meet URL in location is NOT travel-related",
			event: api.CalendarEvent{
				Summary:  "1:1 meeting",
				Location: ptr("https://meet.google.com/abc-defg-hij"),
			},
			expected: false,
		},
		{
			name: "webex URL in location is NOT travel-related",
			event: api.CalendarEvent{
				Summary:  "Client call",
				Location: ptr("https://company.webex.com/meet/john"),
			},
			expected: false,
		},
		{
			name: "http URL in location is NOT travel-related",
			event: api.CalendarEvent{
				Summary:  "Virtual event",
				Location: ptr("http://example.com/meeting"),
			},
			expected: false,
		},
		{
			name: "travel keyword in summary is travel-related",
			event: api.CalendarEvent{
				Summary: "Flight to NYC",
			},
			expected: true,
		},
		{
			name: "hotel keyword in summary is travel-related",
			event: api.CalendarEvent{
				Summary: "Hotel check-in",
			},
			expected: true,
		},
		{
			name: "empty location and no keywords is NOT travel-related",
			event: api.CalendarEvent{
				Summary:  "Team standup",
				Location: ptr(""),
			},
			expected: false,
		},
		{
			name: "nil location and no keywords is NOT travel-related",
			event: api.CalendarEvent{
				Summary: "Sprint planning",
			},
			expected: false,
		},
		{
			name: "URL location but with travel keyword is travel-related",
			event: api.CalendarEvent{
				Summary:  "Flight booking confirmation",
				Location: ptr("https://zoom.us/j/123456789"),
			},
			expected: true,
		},
		// Meeting room tests (Issue #28)
		{
			name: "Google Workspace meeting room AU-SYD is NOT travel-related",
			event: api.CalendarEvent{
				Summary:  "Capslock working group",
				Location: ptr("AU-SYD-ODI-1-3-Oblique (7) [GVC]"),
			},
			expected: false,
		},
		{
			name: "Google Workspace meeting room US-BLD is NOT travel-related",
			event: api.CalendarEvent{
				Summary:  "Team sync",
				Location: ptr("US-BLD-PEARL2930-1-B-Boxelder (2) [GVC, Preview]"),
			},
			expected: false,
		},
		{
			name: "meeting room with capacity indicator is NOT travel-related",
			event: api.CalendarEvent{
				Summary:  "Planning meeting",
				Location: ptr("Conference Room Alpha (12)"),
			},
			expected: false,
		},
		{
			name: "meeting room with GVC tag is NOT travel-related",
			event: api.CalendarEvent{
				Summary:  "Video call",
				Location: ptr("Room 101 [GVC]"),
			},
			expected: false,
		},
		{
			name: "multiple meeting rooms is NOT travel-related",
			event: api.CalendarEvent{
				Summary:  "Cross-office sync",
				Location: ptr("AU-SYD-ODI-1-3-Oblique (7) [GVC], US-BLD-PEARL2930-1-B-Boxelder (2) [GVC, Preview]"),
			},
			expected: false,
		},
		{
			name: "meeting room but with travel keyword IS travel-related",
			event: api.CalendarEvent{
				Summary:  "Flight planning meeting",
				Location: ptr("AU-SYD-ODI-1-3-Oblique (7) [GVC]"),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTravelRelatedEvent(tt.event)
			if got != tt.expected {
				t.Errorf("isTravelRelatedEvent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ptr is a helper to create a pointer to a string.
func ptr(s string) *string {
	return &s
}

func TestLocationsMatch(t *testing.T) {
	tests := []struct {
		loc1, loc2 string
		want       bool
	}{
		// Exact matches
		{"Paris", "Paris", true},
		{"Paris", "paris", true},
		{"paris", "Paris", true},

		// With comma normalization
		{"Paris", "Paris, France", true},
		{"Brussels", "Brussels, Belgium", true},
		{"New York", "New York, NY, USA", true},

		// Aliases
		{"NYC", "New York", true},
		{"New York", "NYC", true},
		{"SF", "San Francisco", true},
		{"LA", "Los Angeles", true},
		{"DC", "Washington", true},

		// Non-matches
		{"Paris", "London", false},
		{"New York", "Los Angeles", false},
		{"Brussels", "Amsterdam", false},

		// Empty strings
		{"", "Paris", false},
		{"Paris", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		name := tt.loc1 + " vs " + tt.loc2
		t.Run(name, func(t *testing.T) {
			got := locationsMatch(tt.loc1, tt.loc2)
			if got != tt.want {
				t.Errorf("locationsMatch(%q, %q) = %v, want %v", tt.loc1, tt.loc2, got, tt.want)
			}
		})
	}
}

func TestIsMeetingRoomLocation(t *testing.T) {
	tests := []struct {
		name     string
		location string
		expected bool
	}{
		// Meeting rooms (should return true)
		{"Google Workspace AU-SYD format", "AU-SYD-ODI-1-3-Oblique (7) [GVC]", true},
		{"Google Workspace US-BLD format", "US-BLD-PEARL2930-1-B-Boxelder (2) [GVC, Preview]", true},
		{"Google Workspace US-NYC format", "US-NYC-123-4-5-Room (10) [GVC]", true},
		{"room with capacity only", "Conference Room (12)", true},
		{"room with GVC tag only", "Meeting Room [GVC]", true},
		{"multiple rooms", "AU-SYD-Room (5), US-BLD-Room (3)", true},

		// Physical addresses (should return false)
		{"street address", "123 Main St, San Francisco, CA", false},
		{"city only", "New York, NY", false},
		{"venue name", "Marriott Hotel Downtown", false},
		{"airport code in address", "SFO Airport Terminal 2", false},
		{"simple room name", "Conference Room A", false},
		{"building name", "Empire State Building", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMeetingRoomLocation(tt.location)
			if got != tt.expected {
				t.Errorf("isMeetingRoomLocation(%q) = %v, want %v", tt.location, got, tt.expected)
			}
		})
	}
}
