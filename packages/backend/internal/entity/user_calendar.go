// Package entity defines the internal domain models with database tags.
package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/api"
)

// UserCalendar represents a user's selected calendar for monitoring.
type UserCalendar struct {
	ID         uuid.UUID `db:"id" firestore:"id"`
	UserID     string    `db:"user_id" firestore:"userId"`
	CalendarID string    `db:"calendar_id" firestore:"calendarId"` // Google Calendar ID
	Name       string    `db:"name" firestore:"name"`
	Enabled    bool      `db:"enabled" firestore:"enabled"`
	CreatedAt  time.Time `db:"created_at" firestore:"createdAt"`
	UpdatedAt  time.Time `db:"updated_at" firestore:"updatedAt"`
}

// ToAPI converts a UserCalendar entity to an API UserCalendar response.
func (c *UserCalendar) ToAPI() api.UserCalendar {
	return api.UserCalendar{
		CalendarId: c.CalendarID,
		Name:       c.Name,
		Enabled:    c.Enabled,
	}
}

// NewUserCalendar creates a new UserCalendar entity.
func NewUserCalendar(userID, calendarID, name string, enabled bool) UserCalendar {
	now := time.Now()
	return UserCalendar{
		ID:         uuid.New(),
		UserID:     userID,
		CalendarID: calendarID,
		Name:       name,
		Enabled:    enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// UserCalendarsToAPI converts a slice of UserCalendar entities to API responses.
func UserCalendarsToAPI(calendars []UserCalendar) []api.UserCalendar {
	result := make([]api.UserCalendar, len(calendars))
	for i, c := range calendars {
		result[i] = c.ToAPI()
	}
	return result
}
