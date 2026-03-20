package entity

import (
	"time"

	"github.com/google/uuid"
)

// ProcessedCalendarEvent tracks calendar events that have been imported, dismissed, or merged.
// This prevents the same events from appearing as suggestions repeatedly.
type ProcessedCalendarEvent struct {
	ID              uuid.UUID  `firestore:"id"`              // Unique identifier
	UserID          string     `firestore:"userId"`          // Owner user
	CalendarEventID string     `firestore:"calendarEventId"` // Google Calendar event ID
	CalendarID      string     `firestore:"calendarId"`      // Which calendar the event belongs to
	Action          string     `firestore:"action"`          // "imported", "dismissed", or "merged"
	TripID          *uuid.UUID `firestore:"tripId"`          // Trip that received this event (if imported/merged)
	ItemID          *uuid.UUID `firestore:"itemId"`          // Item created from this event (if applicable)
	ProcessedAt     time.Time  `firestore:"processedAt"`     // When the event was processed
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
