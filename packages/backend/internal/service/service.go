// Package service provides business logic for the travel calendar.
package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/api"
	"github.com/user/travel-calendar/backend/internal/entity"
	"github.com/user/travel-calendar/backend/internal/store"
)

// Service provides business logic operations.
type Service struct {
	store *store.Store
}

// New creates a new Service with the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Trip operations

// ListTrips returns all trips, optionally filtered.
func (s *Service) ListTrips(upcoming, past *bool, purpose *api.TripPurpose) ([]api.Trip, error) {
	var purposeStr *string
	if purpose != nil {
		p := string(*purpose)
		purposeStr = &p
	}

	trips, err := s.store.ListTrips(upcoming, past, purposeStr)
	if err != nil {
		return nil, fmt.Errorf("listing trips: %w", err)
	}

	result := make([]api.Trip, len(trips))
	for i, trip := range trips {
		result[i] = trip.ToAPI()
	}
	return result, nil
}

// GetTrip returns a single trip by ID with its items.
func (s *Service) GetTrip(id uuid.UUID) (*api.Trip, error) {
	trip, err := s.store.GetTrip(id)
	if err != nil {
		return nil, fmt.Errorf("getting trip: %w", err)
	}
	if trip == nil {
		return nil, nil
	}

	items, err := s.store.ListItems(id)
	if err != nil {
		return nil, fmt.Errorf("getting trip items: %w", err)
	}

	result := trip.ToAPIWithItems(items)
	return &result, nil
}

// CreateTrip creates a new trip.
func (s *Service) CreateTrip(req *api.CreateTripRequest) (*api.Trip, error) {
	trip := entity.TripFromCreateRequest(req)
	if err := s.store.CreateTrip(&trip); err != nil {
		return nil, fmt.Errorf("creating trip: %w", err)
	}
	result := trip.ToAPI()
	return &result, nil
}

// UpdateTrip updates an existing trip.
func (s *Service) UpdateTrip(id uuid.UUID, req *api.UpdateTripRequest) (*api.Trip, error) {
	trip, err := s.store.GetTrip(id)
	if err != nil {
		return nil, fmt.Errorf("getting trip: %w", err)
	}
	if trip == nil {
		return nil, nil
	}

	trip.ApplyUpdate(req)
	if err := s.store.UpdateTrip(trip); err != nil {
		return nil, fmt.Errorf("updating trip: %w", err)
	}

	result := trip.ToAPI()
	return &result, nil
}

// DeleteTrip deletes a trip by ID.
func (s *Service) DeleteTrip(id uuid.UUID) error {
	return s.store.DeleteTrip(id)
}

// SearchTrips searches trips by query string.
func (s *Service) SearchTrips(q string) ([]api.Trip, error) {
	trips, err := s.store.SearchTrips(q)
	if err != nil {
		return nil, fmt.Errorf("searching trips: %w", err)
	}

	result := make([]api.Trip, len(trips))
	for i, trip := range trips {
		result[i] = trip.ToAPI()
	}
	return result, nil
}

// Item operations

// ListTripItems returns all items for a trip.
func (s *Service) ListTripItems(tripID uuid.UUID) ([]api.Item, error) {
	// Verify trip exists
	trip, err := s.store.GetTrip(tripID)
	if err != nil {
		return nil, fmt.Errorf("getting trip: %w", err)
	}
	if trip == nil {
		return nil, nil // Trip not found
	}

	items, err := s.store.ListItems(tripID)
	if err != nil {
		return nil, fmt.Errorf("listing items: %w", err)
	}

	result := make([]api.Item, len(items))
	for i, item := range items {
		result[i] = item.ToAPI()
	}
	return result, nil
}

// CreateTripItem creates a new item for a trip.
func (s *Service) CreateTripItem(tripID uuid.UUID, req *api.CreateItemRequest) (*api.Item, error) {
	// Verify trip exists
	trip, err := s.store.GetTrip(tripID)
	if err != nil {
		return nil, fmt.Errorf("getting trip: %w", err)
	}
	if trip == nil {
		return nil, nil // Trip not found
	}

	item := entity.ItemFromCreateRequest(tripID, req)
	if err := s.store.CreateItem(&item); err != nil {
		return nil, fmt.Errorf("creating item: %w", err)
	}

	result := item.ToAPI()
	return &result, nil
}

// DeleteItem deletes an item by ID.
func (s *Service) DeleteItem(id uuid.UUID) error {
	return s.store.DeleteItem(id)
}

// Document operations

// ListDocuments returns documents, optionally filtered.
func (s *Service) ListDocuments(tripID *uuid.UUID, unassociated *bool) ([]api.Document, error) {
	docs, err := s.store.ListDocuments(tripID, unassociated)
	if err != nil {
		return nil, fmt.Errorf("listing documents: %w", err)
	}

	result := make([]api.Document, len(docs))
	for i, doc := range docs {
		result[i] = doc.ToAPI()
	}
	return result, nil
}
