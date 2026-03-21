package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// DayEntry represents a location on a specific date.
// Day entries can exist standalone or be associated with a trip.
type DayEntry struct {
	ID          uuid.UUID  `db:"id" firestore:"-"`
	UserID      string     `db:"user_id" firestore:"userId"`
	Date        time.Time  `db:"date" firestore:"date"`
	Location    string     `db:"location" firestore:"location"`
	Description *string    `db:"description" firestore:"description"`
	TripID      *uuid.UUID `db:"trip_id" firestore:"tripId"`
	CreatedAt   time.Time  `db:"created_at" firestore:"createdAt"`
}

// ToAPI converts a DayEntry entity to an API DayEntry response.
func (e *DayEntry) ToAPI() api.DayEntry {
	entry := api.DayEntry{
		Id:        openapi_types.UUID(e.ID),
		UserId:    e.UserID,
		Date:      openapi_types.Date{Time: e.Date},
		Location:  e.Location,
		CreatedAt: e.CreatedAt,
	}
	if e.Description != nil {
		entry.Description = e.Description
	}
	if e.TripID != nil {
		tid := openapi_types.UUID(*e.TripID)
		entry.TripId = &tid
	}
	return entry
}

// NewDayEntry creates a new DayEntry with a generated ID.
func NewDayEntry(userID string, date time.Time, location string, description *string, tripID *uuid.UUID) DayEntry {
	return DayEntry{
		ID:          uuid.New(),
		UserID:      userID,
		Date:        date,
		Location:    location,
		Description: description,
		TripID:      tripID,
		CreatedAt:   time.Now(),
	}
}
