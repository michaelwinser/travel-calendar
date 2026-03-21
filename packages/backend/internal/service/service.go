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
	store store.StoreInterface
}

// New creates a new Service with the given store.
func New(s store.StoreInterface) *Service {
	return &Service{store: s}
}

// Trip operations

// ListTrips returns all trips, optionally filtered.
func (s *Service) ListTrips(userID string, upcoming, past *bool, purpose *api.TripPurpose) ([]api.Trip, error) {
	var purposeStr *string
	if purpose != nil {
		p := string(*purpose)
		purposeStr = &p
	}

	trips, err := s.store.ListTrips(userID, upcoming, past, purposeStr)
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
func (s *Service) GetTrip(userID string, id uuid.UUID) (*api.Trip, error) {
	trip, err := s.store.GetTrip(userID, id)
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
func (s *Service) CreateTrip(userID string, req *api.CreateTripRequest) (*api.Trip, error) {
	trip := entity.TripFromCreateRequest(req)
	trip.UserID = userID
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
func (s *Service) UpdateTrip(userID string, id uuid.UUID, req *api.UpdateTripRequest) (*api.Trip, error) {
	trip, err := s.store.GetTrip(userID, id)
	if err != nil {
		return nil, fmt.Errorf("getting trip: %w", err)
	}
	if trip == nil {
		return nil, nil
	}

	trip.ApplyUpdate(req)
	if err := s.store.UpdateTrip(userID, trip); err != nil {
		return nil, fmt.Errorf("updating trip: %w", err)
	}

	result := trip.ToAPI()
	return &result, nil
}

// DeleteTrip deletes a trip by ID.
func (s *Service) DeleteTrip(userID string, id uuid.UUID) error {
	return s.store.DeleteTrip(userID, id)
}

// SearchTrips searches trips by query string.
func (s *Service) SearchTrips(userID string, q string) ([]api.Trip, error) {
	trips, err := s.store.SearchTrips(userID, q)
	if err != nil {
		return nil, fmt.Errorf("searching trips: %w", err)
	}

	result := make([]api.Trip, len(trips))
	for i, trip := range trips {
		result[i] = trip.ToAPI()
	}
	return result, nil
}

// Trip Organization operations

// MergeTrips merges a source trip into a target trip.
// All items are moved from source to target, dates are extended if needed,
// notes are concatenated if mergeNotes is true, and source is deleted.
func (s *Service) MergeTrips(userID string, sourceID, targetID uuid.UUID, req *api.MergeTripsRequest) (*api.Trip, error) {
	// Validate both trips exist
	source, err := s.store.GetTrip(userID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("getting source trip: %w", err)
	}
	if source == nil {
		return nil, nil // Not found
	}

	target, err := s.store.GetTrip(userID, targetID)
	if err != nil {
		return nil, fmt.Errorf("getting target trip: %w", err)
	}
	if target == nil {
		return nil, nil // Not found
	}

	// Cannot merge trip into itself
	if sourceID == targetID {
		return nil, fmt.Errorf("cannot merge trip into itself")
	}

	// Extend target dates if source dates are outside target range
	datesChanged := false
	if source.StartDate != nil {
		if target.StartDate == nil || source.StartDate.Before(*target.StartDate) {
			target.StartDate = source.StartDate
			datesChanged = true
		}
	}
	if source.EndDate != nil {
		if target.EndDate == nil || source.EndDate.After(*target.EndDate) {
			target.EndDate = source.EndDate
			datesChanged = true
		}
	}

	// Concatenate notes if requested (defaults to true)
	mergeNotes := req.MergeNotes == nil || *req.MergeNotes
	if mergeNotes && source.Notes != nil && *source.Notes != "" {
		if target.Notes != nil && *target.Notes != "" {
			combined := *target.Notes + "\n\n---\n\n" + *source.Notes
			target.Notes = &combined
		} else {
			target.Notes = source.Notes
		}
	}

	// Update target trip with extended dates/notes if changed
	if datesChanged || (mergeNotes && source.Notes != nil) {
		target.UpdatedAt = time.Now()
		if err := s.store.UpdateTrip(userID, target); err != nil {
			return nil, fmt.Errorf("updating target trip: %w", err)
		}
	}

	// Execute merge transaction (moves items, merges locations, deletes source)
	deleteSource := req.DeleteSource == nil || *req.DeleteSource
	if deleteSource {
		if err := s.store.MergeTripsTransaction(sourceID, targetID); err != nil {
			return nil, fmt.Errorf("executing merge: %w", err)
		}
	}

	// Return updated target trip with items
	return s.GetTrip(userID, targetID)
}

// MoveItem moves an item to a different trip.
// If targetTripId is provided, moves to that trip.
// If newTrip is provided, creates a new trip and moves item to it.
func (s *Service) MoveItem(userID string, itemID uuid.UUID, req *api.MoveItemRequest) (*api.MoveItemResponse, error) {
	// Get the item to verify it exists
	item, err := s.store.GetItem(itemID)
	if err != nil {
		return nil, fmt.Errorf("getting item: %w", err)
	}
	if item == nil {
		return nil, nil // Not found
	}

	var targetTripID uuid.UUID
	var createdTrip *api.Trip

	// Determine target trip
	if req.TargetTripId != nil {
		// Move to existing trip
		targetTripID = uuid.UUID(*req.TargetTripId)

		// Verify target trip exists
		targetTrip, err := s.store.GetTrip(userID, targetTripID)
		if err != nil {
			return nil, fmt.Errorf("getting target trip: %w", err)
		}
		if targetTrip == nil {
			return nil, fmt.Errorf("target trip not found")
		}

		// Cannot move to same trip
		if targetTripID == item.TripID {
			return nil, fmt.Errorf("item is already on this trip")
		}
	} else if req.NewTrip != nil {
		// Create new trip
		trip, err := s.CreateTrip(userID, req.NewTrip)
		if err != nil {
			return nil, fmt.Errorf("creating new trip: %w", err)
		}
		targetTripID = uuid.UUID(trip.Id)
		createdTrip = trip
	} else {
		return nil, fmt.Errorf("must provide targetTripId or newTrip")
	}

	// Update item's trip assignment
	if err := s.store.UpdateItemTrip(itemID, targetTripID); err != nil {
		return nil, fmt.Errorf("moving item: %w", err)
	}

	// Get updated item
	item, err = s.store.GetItem(itemID)
	if err != nil {
		return nil, fmt.Errorf("getting moved item: %w", err)
	}

	apiItem := item.ToAPI()
	return &api.MoveItemResponse{
		Item: apiItem,
		Trip: createdTrip,
	}, nil
}

// Item operations

// ListTripItems returns all items for a trip.
func (s *Service) ListTripItems(userID string, tripID uuid.UUID) ([]api.Item, error) {
	// Verify trip exists
	trip, err := s.store.GetTrip(userID, tripID)
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
func (s *Service) CreateTripItem(userID string, tripID uuid.UUID, req *api.CreateItemRequest) (*api.Item, error) {
	// Verify trip exists
	trip, err := s.store.GetTrip(userID, tripID)
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
func (s *Service) GetTripLocations(userID string, tripID uuid.UUID) ([]api.TripDayLocation, error) {
	// Verify trip exists
	trip, err := s.store.GetTrip(userID, tripID)
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
func (s *Service) SetTripLocations(userID string, tripID uuid.UUID, req *api.SetTripLocationsRequest) ([]api.TripDayLocation, error) {
	// Verify trip exists and get dates
	trip, err := s.store.GetTrip(userID, tripID)
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

	return s.GetTripLocations(userID, tripID)
}

// Location Query operations

// GetLocationOnDate returns the user's location on a specific date.
func (s *Service) GetLocationOnDate(userID string, date time.Time) (*api.LocationOnDateResponse, error) {
	// Find trips that span this date
	trips, err := s.store.GetTripsForDateRange(userID, date, date)
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
func (s *Service) GetLocationRange(userID string, from, to time.Time) ([]api.LocationRangeSegment, error) {
	// Build day-by-day location list
	type dayLocation struct {
		date      time.Time
		locations []string
		source    api.LocationSource
	}

	var days []dayLocation

	// Get all trips that overlap with the range
	trips, err := s.store.GetTripsForDateRange(userID, from, to)
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

// Day Entry operations

// ListDayEntries returns day entries for a date range.
func (s *Service) ListDayEntries(userID string, from, to time.Time) ([]api.DayEntry, error) {
	entries, err := s.store.ListDayEntries(userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("listing day entries: %w", err)
	}
	result := make([]api.DayEntry, len(entries))
	for i, e := range entries {
		result[i] = e.ToAPI()
	}
	return result, nil
}

// CreateDayEntry creates a new day entry.
func (s *Service) CreateDayEntry(userID string, req *api.CreateDayEntryRequest) (*api.DayEntry, error) {
	var tripID *uuid.UUID
	if req.TripId != nil {
		tid := uuid.UUID(*req.TripId)
		tripID = &tid
	}

	entry := entity.NewDayEntry(userID, req.Date.Time, req.Location, req.Description, tripID)
	if err := s.store.CreateDayEntry(&entry); err != nil {
		return nil, fmt.Errorf("creating day entry: %w", err)
	}

	result := entry.ToAPI()
	return &result, nil
}

// UpdateDayEntry updates an existing day entry.
func (s *Service) UpdateDayEntry(userID string, id uuid.UUID, req *api.UpdateDayEntryRequest) (*api.DayEntry, error) {
	entry, err := s.store.GetDayEntry(userID, id)
	if err != nil {
		return nil, fmt.Errorf("getting day entry: %w", err)
	}
	if entry == nil {
		return nil, nil
	}

	if req.Location != nil {
		entry.Location = *req.Location
	}
	if req.Description != nil {
		entry.Description = req.Description
	}
	if req.TripId != nil {
		tid := uuid.UUID(*req.TripId)
		entry.TripID = &tid
	}

	if err := s.store.UpdateDayEntry(userID, entry); err != nil {
		return nil, fmt.Errorf("updating day entry: %w", err)
	}

	result := entry.ToAPI()
	return &result, nil
}

// DeleteDayEntry deletes a day entry.
func (s *Service) DeleteDayEntry(userID string, id uuid.UUID) error {
	return s.store.DeleteDayEntry(userID, id)
}

// CreateDayEntriesForTrip creates day entries for all dates in a trip's range.
func (s *Service) CreateDayEntriesForTrip(userID string, tripID uuid.UUID, location string, startDate, endDate time.Time) error {
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		entry := entity.NewDayEntry(userID, d, location, nil, &tripID)
		if err := s.store.CreateDayEntry(&entry); err != nil {
			return fmt.Errorf("creating day entry for %s: %w", d.Format("2006-01-02"), err)
		}
	}
	return nil
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
