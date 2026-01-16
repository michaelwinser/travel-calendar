package service

import (
	"regexp"
	"strings"

	"github.com/user/travel-calendar/backend/internal/api"
)

// EventClassification indicates how an event should be treated.
type EventClassification int

const (
	// ClassificationSkip means the event is not travel-related.
	ClassificationSkip EventClassification = iota
	// ClassificationTripLevel means the event represents a trip (container).
	ClassificationTripLevel
	// ClassificationItemLevel means the event is an item within a trip.
	ClassificationItemLevel
)

// ClassifiedEvent wraps an event with its classification.
type ClassifiedEvent struct {
	Event          api.CalendarEvent
	Classification EventClassification
	ItemType       string // "flight", "hotel", "train", "drive", "event"
	Source         string // "google", "tripit"
}

// flightCodePattern matches airline codes like "UA123", "BA456".
var flightCodePattern = regexp.MustCompile(`\b[A-Z]{2}\d{1,4}\b`)

// ClassifyEvent determines how a calendar event should be handled.
func ClassifyEvent(event api.CalendarEvent) ClassifiedEvent {
	result := ClassifiedEvent{
		Event:  event,
		Source: "google",
	}

	// Check if it's a TripIt event
	if IsTripItEvent(event) {
		result.Source = "tripit"

		// Try TripIt summary pattern (all-day trip summary)
		if _, ok := ParseTripItSummary(event); ok {
			result.Classification = ClassificationTripLevel
			return result
		}

		// Try TripIt flight pattern
		if _, ok := ParseTripItFlight(event); ok {
			result.Classification = ClassificationItemLevel
			result.ItemType = "flight"
			return result
		}
	}

	// Not TripIt - use general classification rules

	// Check if event passes travel filter
	if !isTravelRelatedEvent(event) {
		result.Classification = ClassificationSkip
		return result
	}

	// All-day events with location -> trip level
	if event.AllDay != nil && *event.AllDay {
		if event.Location != nil && *event.Location != "" {
			result.Classification = ClassificationTripLevel
			return result
		}
		// All-day without location -> skip
		result.Classification = ClassificationSkip
		return result
	}

	// Timed events -> item level
	result.Classification = ClassificationItemLevel
	result.ItemType = inferItemType(event)

	return result
}

// inferItemType guesses the item type from event content.
func inferItemType(event api.CalendarEvent) string {
	summary := strings.ToLower(event.Summary)

	// Flight indicators
	if strings.Contains(summary, "flight") ||
		flightCodePattern.MatchString(event.Summary) {
		return "flight"
	}

	// Hotel indicators
	if strings.Contains(summary, "hotel") ||
		strings.Contains(summary, "checkin") ||
		strings.Contains(summary, "check-in") ||
		strings.Contains(summary, "checkout") ||
		strings.Contains(summary, "check-out") ||
		strings.Contains(summary, "accommodation") {
		return "hotel"
	}

	// Train indicators
	if strings.Contains(summary, "train") ||
		strings.Contains(summary, "eurostar") ||
		strings.Contains(summary, "amtrak") ||
		strings.Contains(summary, "tgv") ||
		strings.Contains(summary, "thalys") {
		return "train"
	}

	// Drive indicators
	if strings.Contains(summary, "drive") ||
		strings.Contains(summary, "car rental") ||
		strings.Contains(summary, "pickup") ||
		strings.Contains(summary, "rental car") {
		return "drive"
	}

	// Default to generic event
	return "event"
}
