// Package entity defines the internal domain models with database tags.
package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// CalendarLink tracks the sync state between trip items and Google Calendar events.
type CalendarLink struct {
	ID         uuid.UUID  `db:"id" firestore:"id"`
	TripID     uuid.UUID  `db:"trip_id" firestore:"tripId"`
	ItemID     *uuid.UUID `db:"item_id" firestore:"itemId"` // nil if linked to whole trip
	CalendarID string     `db:"calendar_id" firestore:"calendarId"`
	EventID    string     `db:"event_id" firestore:"eventId"`
	SyncedAt   time.Time  `db:"synced_at" firestore:"syncedAt"`
}

// ToAPI converts a CalendarLink entity to an API CalendarLink response.
func (l *CalendarLink) ToAPI() api.CalendarLink {
	link := api.CalendarLink{
		TripId:     openapi_types.UUID(l.TripID),
		CalendarId: l.CalendarID,
		EventId:    l.EventID,
		SyncedAt:   l.SyncedAt,
	}

	if l.ItemID != nil {
		itemID := openapi_types.UUID(*l.ItemID)
		link.ItemId = &itemID
	}

	return link
}

// NewCalendarLink creates a new CalendarLink entity.
func NewCalendarLink(tripID uuid.UUID, itemID *uuid.UUID, calendarID, eventID string) CalendarLink {
	return CalendarLink{
		ID:         uuid.New(),
		TripID:     tripID,
		ItemID:     itemID,
		CalendarID: calendarID,
		EventID:    eventID,
		SyncedAt:   time.Now(),
	}
}

// CalendarLinksToAPI converts a slice of CalendarLink entities to API responses.
func CalendarLinksToAPI(links []CalendarLink) []api.CalendarLink {
	result := make([]api.CalendarLink, len(links))
	for i, l := range links {
		result[i] = l.ToAPI()
	}
	return result
}
