package app

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// GCalClient wraps the Google Calendar API.
type GCalClient struct {
	svc *calendar.Service
}

// NewGCalClient creates a Calendar API client from an OAuth token.
func NewGCalClient(ctx context.Context, accessToken string) (*GCalClient, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	svc, err := calendar.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("creating calendar service: %w", err)
	}
	return &GCalClient{svc: svc}, nil
}

// CreateCalendar creates a new calendar and returns its ID.
func (c *GCalClient) CreateCalendar(ctx context.Context, name string) (string, error) {
	cal, err := c.svc.Calendars.Insert(&calendar.Calendar{
		Summary:     name,
		Description: "Managed by Travel Calendar",
		TimeZone:    "UTC",
	}).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("creating calendar: %w", err)
	}
	return cal.Id, nil
}

// ListCalendars returns the user's calendar list.
func (c *GCalClient) ListCalendars(ctx context.Context) ([]*calendar.CalendarListEntry, error) {
	list, err := c.svc.CalendarList.List().Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// UpsertEvent creates or updates an all-day event on the calendar.
// Uses the event ID for idempotent upsert.
func (c *GCalClient) UpsertEvent(ctx context.Context, calendarID string, event *calendar.Event) (*calendar.Event, error) {
	// Try update first (if event has an ID)
	if event.Id != "" {
		existing, err := c.svc.Events.Get(calendarID, event.Id).Context(ctx).Do()
		if err == nil && existing != nil {
			updated, err := c.svc.Events.Update(calendarID, event.Id, event).Context(ctx).Do()
			if err != nil {
				return nil, fmt.Errorf("updating event: %w", err)
			}
			return updated, nil
		}
	}

	// Create new
	created, err := c.svc.Events.Insert(calendarID, event).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating event: %w", err)
	}
	return created, nil
}

// DeleteEvent removes an event from the calendar.
func (c *GCalClient) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	return c.svc.Events.Delete(calendarID, eventID).Context(ctx).Do()
}

// ListEvents returns events from the calendar.
func (c *GCalClient) ListEvents(ctx context.Context, calendarID string) ([]*calendar.Event, error) {
	var allEvents []*calendar.Event
	pageToken := ""
	for {
		call := c.svc.Events.List(calendarID).Context(ctx).SingleEvents(true).MaxResults(250)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		result, err := call.Do()
		if err != nil {
			return nil, err
		}
		allEvents = append(allEvents, result.Items...)
		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}
	return allEvents, nil
}

// DeleteCalendar deletes a calendar entirely.
func (c *GCalClient) DeleteCalendar(ctx context.Context, calendarID string) error {
	return c.svc.Calendars.Delete(calendarID).Context(ctx).Do()
}

// --- Helpers ---

// TripToEvent converts a trip with its activities to a Google Calendar all-day event.
func TripToEvent(trip Trip, activities []Activity) *calendar.Event {
	// Compute date range and dominant location
	startDate := ""
	endDate := ""
	locCount := map[string]int{}

	for _, a := range activities {
		if startDate == "" || a.StartDate < startDate {
			startDate = a.StartDate
		}
		if endDate == "" || a.EndDate > endDate {
			endDate = a.EndDate
		}
		if a.Location != "" {
			locCount[a.Location]++
		}
	}

	location := ""
	maxCount := 0
	for loc, count := range locCount {
		if count > maxCount {
			location = loc
			maxCount = count
		}
	}

	// DTEND is exclusive for all-day events
	endExclusive := endDate
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			endExclusive = t.AddDate(0, 0, 1).Format("2006-01-02")
		}
	}

	return &calendar.Event{
		Summary:      trip.Name,
		Location:     location,
		Description:  fmt.Sprintf("Trip: %s\nManaged by Travel Calendar", trip.Name),
		Transparency: "transparent", // free, not blocking
		Start: &calendar.EventDateTime{
			Date: startDate,
		},
		End: &calendar.EventDateTime{
			Date: endExclusive,
		},
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				"travel-calendar-type": "trip",
				"travel-calendar-id":   trip.ID,
				"travel-calendar-key":  trip.Key,
			},
		},
	}
}

// ActivityToEvent converts an activity to a Google Calendar all-day event.
func ActivityToEvent(a Activity) *calendar.Event {
	endExclusive := a.EndDate
	if t, err := time.Parse("2006-01-02", a.EndDate); err == nil {
		endExclusive = t.AddDate(0, 0, 1).Format("2006-01-02")
	}

	return &calendar.Event{
		Summary:      a.Title,
		Location:     a.Location,
		Transparency: "transparent",
		Start: &calendar.EventDateTime{
			Date: a.StartDate,
		},
		End: &calendar.EventDateTime{
			Date: endExclusive,
		},
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				"travel-calendar-type": "activity",
				"travel-calendar-id":   a.ID,
				"travel-calendar-key":  a.Key,
			},
		},
	}
}

// SyncHash computes a hash of the fields that matter for sync.
func SyncHash(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return fmt.Sprintf("%x", h[:8])
}

