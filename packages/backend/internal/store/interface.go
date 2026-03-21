// Package store provides database access layer abstractions.
package store

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/entity"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// StoreInterface defines the contract for all store implementations.
type StoreInterface interface {
	// Lifecycle
	Close() error

	// Trip methods (user-scoped)
	ListTrips(userID string, upcoming, past *bool, purpose *string) ([]entity.Trip, error)
	GetTrip(userID string, id uuid.UUID) (*entity.Trip, error)
	CreateTrip(trip *entity.Trip) error
	UpdateTrip(userID string, trip *entity.Trip) error
	DeleteTrip(userID string, id uuid.UUID) error
	SearchTrips(userID string, q string) ([]entity.Trip, error)

	// Item methods (scoped through trip ownership)
	ListItems(tripID uuid.UUID) ([]entity.Item, error)
	GetItem(id uuid.UUID) (*entity.Item, error)
	CreateItem(item *entity.Item) error
	DeleteItem(id uuid.UUID) error
	UpdateItemTrip(itemID, newTripID uuid.UUID) error

	// Trip Organization methods
	MergeTripsTransaction(sourceID, targetID uuid.UUID) error

	// Document methods
	ListDocuments(tripID *uuid.UUID, unassociated *bool) ([]entity.Document, error)

	// Config methods
	GetConfig(key string) (*string, error)
	SetConfig(key, value string) error
	DeleteConfig(key string) error

	// Day Entry methods
	ListDayEntries(userID string, from, to time.Time) ([]entity.DayEntry, error)
	GetDayEntry(userID string, id uuid.UUID) (*entity.DayEntry, error)
	CreateDayEntry(entry *entity.DayEntry) error
	UpdateDayEntry(userID string, entry *entity.DayEntry) error
	DeleteDayEntry(userID string, id uuid.UUID) error
	GetDayEntriesForTrip(userID string, tripID uuid.UUID) ([]entity.DayEntry, error)
	DeleteDayEntriesByTrip(tripID uuid.UUID) error

	// Trip Location methods (legacy — being replaced by day entries)
	GetTripLocations(tripID uuid.UUID) ([]entity.TripLocation, error)
	SetTripLocations(tripID uuid.UUID, locations []entity.TripLocation) error
	GetTripsForDateRange(userID string, from, to time.Time) ([]entity.Trip, error)
	GetTripLocationsForDateRange(tripID uuid.UUID, from, to time.Time) ([]entity.TripLocation, error)

	// Google Credentials methods
	GetGoogleCredentials(userID string) (*entity.GoogleCredentials, error)
	SaveGoogleCredentials(creds *entity.GoogleCredentials) error
	DeleteGoogleCredentials(userID string) error

	// User Calendar methods
	ListUserCalendars(userID string) ([]entity.UserCalendar, error)
	GetUserCalendar(id uuid.UUID) (*entity.UserCalendar, error)
	GetUserCalendarByCalendarID(userID, calendarID string) (*entity.UserCalendar, error)
	SaveUserCalendar(cal *entity.UserCalendar) error
	DeleteUserCalendar(id uuid.UUID) error
	DeleteUserCalendarsByUser(userID string) error
	SetUserCalendars(userID string, calendars []entity.UserCalendar) error

	// Calendar Link methods
	ListCalendarLinks(tripID uuid.UUID) ([]entity.CalendarLink, error)
	GetCalendarLink(id uuid.UUID) (*entity.CalendarLink, error)
	GetCalendarLinkByEvent(tripID uuid.UUID, calendarID, eventID string) (*entity.CalendarLink, error)
	CreateCalendarLink(link *entity.CalendarLink) error
	DeleteCalendarLink(id uuid.UUID) error
	DeleteCalendarLinksByTrip(tripID uuid.UUID) error

	// Session methods
	CreateSession(session *entity.Session) error
	GetSession(id string) (*entity.Session, error)
	DeleteSession(id string) error
	DeleteExpiredSessions() error

	// Processed Calendar Events methods (user-scoped)
	CreateProcessedEvent(event *entity.ProcessedCalendarEvent) error
	GetProcessedEventByCalendarEvent(userID string, calendarID, eventID string) (*entity.ProcessedCalendarEvent, error)
	IsEventProcessed(userID string, calendarID, eventID string) (bool, error)
	ListProcessedEvents(userID string, calendarID string) ([]entity.ProcessedCalendarEvent, error)
	DeleteAllProcessedEvents(userID string) error
}
