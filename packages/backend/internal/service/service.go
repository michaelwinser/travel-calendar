// Package service provides business logic for the travel calendar.
package service

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
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

	// If a location was provided and trip has dates, set location for all days
	if req.Location != nil && *req.Location != "" && trip.StartDate != nil && trip.EndDate != nil {
		locReq := &api.SetTripLocationsRequest{
			DefaultLocation: req.Location,
		}
		locations := entity.TripLocationsFromRequest(trip.ID, trip.StartDate, trip.EndDate, locReq)
		if err := s.store.SetTripLocations(trip.ID, locations); err != nil {
			return nil, fmt.Errorf("setting trip locations: %w", err)
		}
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

// Config operations

// GetBaseLocations returns the user's configured base locations.
func (s *Service) GetBaseLocations() (*api.BaseLocations, error) {
	home, err := s.store.GetConfig("home-location")
	if err != nil {
		return nil, fmt.Errorf("getting home location: %w", err)
	}
	work, err := s.store.GetConfig("work-location")
	if err != nil {
		return nil, fmt.Errorf("getting work location: %w", err)
	}

	result := &api.BaseLocations{}
	if home != nil {
		result.Home = home
	}
	if work != nil {
		result.Work = work
	}
	return result, nil
}

// SetBaseLocations updates the user's base locations.
func (s *Service) SetBaseLocations(req *api.SetBaseLocationsRequest) (*api.BaseLocations, error) {
	if req.Home != nil {
		if *req.Home == "" {
			if err := s.store.DeleteConfig("home-location"); err != nil {
				return nil, fmt.Errorf("deleting home location: %w", err)
			}
		} else {
			if err := s.store.SetConfig("home-location", *req.Home); err != nil {
				return nil, fmt.Errorf("setting home location: %w", err)
			}
		}
	}
	if req.Work != nil {
		if *req.Work == "" {
			if err := s.store.DeleteConfig("work-location"); err != nil {
				return nil, fmt.Errorf("deleting work location: %w", err)
			}
		} else {
			if err := s.store.SetConfig("work-location", *req.Work); err != nil {
				return nil, fmt.Errorf("setting work location: %w", err)
			}
		}
	}
	return s.GetBaseLocations()
}

// Trip Location operations

// GetTripLocations returns locations for a trip.
func (s *Service) GetTripLocations(tripID uuid.UUID) ([]api.TripDayLocation, error) {
	// Verify trip exists
	trip, err := s.store.GetTrip(tripID)
	if err != nil {
		return nil, fmt.Errorf("getting trip: %w", err)
	}
	if trip == nil {
		return nil, nil // Trip not found
	}

	locations, err := s.store.GetTripLocations(tripID)
	if err != nil {
		return nil, fmt.Errorf("getting trip locations: %w", err)
	}

	return entity.ToAPITripDayLocations(locations), nil
}

// SetTripLocations sets locations for a trip.
func (s *Service) SetTripLocations(tripID uuid.UUID, req *api.SetTripLocationsRequest) ([]api.TripDayLocation, error) {
	// Verify trip exists and get dates
	trip, err := s.store.GetTrip(tripID)
	if err != nil {
		return nil, fmt.Errorf("getting trip: %w", err)
	}
	if trip == nil {
		return nil, nil // Trip not found
	}

	// Create location entities
	locations := entity.TripLocationsFromRequest(tripID, trip.StartDate, trip.EndDate, req)

	// Save to database
	if err := s.store.SetTripLocations(tripID, locations); err != nil {
		return nil, fmt.Errorf("setting trip locations: %w", err)
	}

	return s.GetTripLocations(tripID)
}

// Location Query operations

// GetLocationOnDate returns the user's location on a specific date.
func (s *Service) GetLocationOnDate(date time.Time) (*api.LocationOnDateResponse, error) {
	// Find trips that span this date
	trips, err := s.store.GetTripsForDateRange(date, date)
	if err != nil {
		return nil, fmt.Errorf("getting trips for date: %w", err)
	}

	// Collect all locations from all overlapping trips
	var allLocations []string
	var sources []tripSource

	for _, trip := range trips {
		locations, err := s.store.GetTripLocationsForDateRange(trip.ID, date, date)
		if err != nil {
			return nil, fmt.Errorf("getting trip locations: %w", err)
		}

		if len(locations) > 0 {
			for _, loc := range locations {
				allLocations = append(allLocations, loc.Location)
			}
		} else {
			// Trip exists but no explicit location - default to "Away"
			allLocations = append(allLocations, "Away")
		}

		sources = append(sources, tripSource{
			id:   trip.ID,
			name: trip.Name,
		})
	}

	// If we found trip locations, return them
	if len(allLocations) > 0 {
		// Use the first trip as the source (could be enhanced to return multiple)
		source := sources[0]
		tripID := openapi_types.UUID(source.id)
		return &api.LocationOnDateResponse{
			Date:      openapi_types.Date{Time: date},
			Locations: unique(allLocations),
			Source: api.LocationSource{
				Type:     api.LocationSourceTypeTrip,
				TripId:   &tripID,
				TripName: &source.name,
			},
		}, nil
	}

	// Not on a trip - return home location
	home, err := s.store.GetConfig("home-location")
	if err != nil {
		return nil, fmt.Errorf("getting home location: %w", err)
	}
	homeLocation := "Home"
	if home != nil {
		homeLocation = *home
	}

	return &api.LocationOnDateResponse{
		Date:      openapi_types.Date{Time: date},
		Locations: []string{homeLocation},
		Source: api.LocationSource{
			Type: api.LocationSourceTypeHome,
		},
	}, nil
}

// GetLocationRange returns the user's locations for a date range.
func (s *Service) GetLocationRange(from, to time.Time) ([]api.LocationRangeSegment, error) {
	// Build day-by-day location list
	type dayLocation struct {
		date      time.Time
		locations []string
		source    api.LocationSource
	}

	var days []dayLocation

	// Get all trips that overlap with the range
	trips, err := s.store.GetTripsForDateRange(from, to)
	if err != nil {
		return nil, fmt.Errorf("getting trips for range: %w", err)
	}

	// Build a map of date -> trip locations
	tripLocations := make(map[string][]struct {
		location string
		tripID   uuid.UUID
		tripName string
	})

	for _, trip := range trips {
		locations, err := s.store.GetTripLocations(trip.ID)
		if err != nil {
			return nil, fmt.Errorf("getting trip locations: %w", err)
		}

		// For each day of the trip within our range
		start := *trip.StartDate
		if start.Before(from) {
			start = from
		}
		end := *trip.EndDate
		if end.After(to) {
			end = to
		}

		locMap := entity.GroupByDate(locations)

		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			if locs, ok := locMap[dateStr]; ok {
				for _, loc := range locs {
					tripLocations[dateStr] = append(tripLocations[dateStr], struct {
						location string
						tripID   uuid.UUID
						tripName string
					}{loc, trip.ID, trip.Name})
				}
			} else {
				// Trip day without explicit location - use "Away"
				tripLocations[dateStr] = append(tripLocations[dateStr], struct {
					location string
					tripID   uuid.UUID
					tripName string
				}{"Away", trip.ID, trip.Name})
			}
		}
	}

	// Get home location
	home, err := s.store.GetConfig("home-location")
	if err != nil {
		return nil, fmt.Errorf("getting home location: %w", err)
	}
	homeLocation := "Home"
	if home != nil {
		homeLocation = *home
	}

	// Build day-by-day list
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if tripLocs, ok := tripLocations[dateStr]; ok {
			// Get unique locations
			var locs []string
			for _, tl := range tripLocs {
				locs = append(locs, tl.location)
			}
			// Use first trip as source
			tripID := openapi_types.UUID(tripLocs[0].tripID)
			tripName := tripLocs[0].tripName
			days = append(days, dayLocation{
				date:      d,
				locations: unique(locs),
				source: api.LocationSource{
					Type:     api.LocationSourceTypeTrip,
					TripId:   &tripID,
					TripName: &tripName,
				},
			})
		} else {
			days = append(days, dayLocation{
				date:      d,
				locations: []string{homeLocation},
				source: api.LocationSource{
					Type: api.LocationSourceTypeHome,
				},
			})
		}
	}

	// Group consecutive days with same location and source
	var segments []api.LocationRangeSegment
	if len(days) == 0 {
		return segments, nil
	}

	current := days[0]
	segmentStart := current.date

	for i := 1; i < len(days); i++ {
		day := days[i]
		// Check if this day continues the current segment
		sameLocations := slicesEqual(current.locations, day.locations)
		sameSource := current.source.Type == day.source.Type

		if !sameLocations || !sameSource {
			// End current segment
			segments = append(segments, api.LocationRangeSegment{
				StartDate: openapi_types.Date{Time: segmentStart},
				EndDate:   openapi_types.Date{Time: current.date},
				Locations: current.locations,
				Source:    current.source,
			})
			// Start new segment
			segmentStart = day.date
		}
		current = day
	}

	// Don't forget the last segment
	segments = append(segments, api.LocationRangeSegment{
		StartDate: openapi_types.Date{Time: segmentStart},
		EndDate:   openapi_types.Date{Time: current.date},
		Locations: current.locations,
		Source:    current.source,
	})

	return segments, nil
}

// Helper types and functions

type tripSource struct {
	id   uuid.UUID
	name string
}

func unique(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(strs))
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	sort.Strings(result)
	return result
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
