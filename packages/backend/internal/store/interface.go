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

	// Trip methods
	ListTrips(upcoming, past *bool, purpose *string) ([]entity.Trip, error)
	GetTrip(id uuid.UUID) (*entity.Trip, error)
	CreateTrip(trip *entity.Trip) error
	UpdateTrip(trip *entity.Trip) error
	DeleteTrip(id uuid.UUID) error
	SearchTrips(q string) ([]entity.Trip, error)

	// Item methods
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

	// Trip Location methods
	GetTripLocations(tripID uuid.UUID) ([]entity.TripLocation, error)
	SetTripLocations(tripID uuid.UUID, locations []entity.TripLocation) error
	GetTripsForDateRange(from, to time.Time) ([]entity.Trip, error)
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

	// Processed Calendar Events methods
	CreateProcessedEvent(event *entity.ProcessedCalendarEvent) error
	GetProcessedEventByCalendarEvent(calendarID, eventID string) (*entity.ProcessedCalendarEvent, error)
	IsEventProcessed(calendarID, eventID string) (bool, error)
	ListProcessedEvents(calendarID string) ([]entity.ProcessedCalendarEvent, error)
	DeleteAllProcessedEvents() error
}
