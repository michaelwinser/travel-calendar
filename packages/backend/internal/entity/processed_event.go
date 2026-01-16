package entity

import (
	"time"

	"github.com/google/uuid"
)

// ProcessedCalendarEvent tracks calendar events that have been imported, dismissed, or merged.
// This prevents the same events from appearing as suggestions repeatedly.
type ProcessedCalendarEvent struct {
	ID              uuid.UUID  // Unique identifier
	CalendarEventID string     // Google Calendar event ID
	CalendarID      string     // Which calendar the event belongs to
	Action          string     // "imported", "dismissed", or "merged"
	TripID          *uuid.UUID // Trip that received this event (if imported/merged)
	ItemID          *uuid.UUID // Item created from this event (if applicable)
	ProcessedAt     time.Time  // When the event was processed
}

// ProcessedEventAction constants.
const (
	ProcessedActionImported  = "imported"
	ProcessedActionDismissed = "dismissed"
	ProcessedActionMerged    = "merged"
)

// NewProcessedCalendarEvent creates a new ProcessedCalendarEvent with a generated ID.
func NewProcessedCalendarEvent(calendarEventID, calendarID, action string) *ProcessedCalendarEvent {
	return &ProcessedCalendarEvent{
		ID:              uuid.New(),
		CalendarEventID: calendarEventID,
		CalendarID:      calendarID,
		Action:          action,
		ProcessedAt:     time.Now(),
	}
}
