package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// TripLocation represents a location for a specific date within a trip.
type TripLocation struct {
	ID        uuid.UUID `db:"id" firestore:"-"`
	TripID    uuid.UUID `db:"trip_id" firestore:"tripId"`
	Date      time.Time `db:"date" firestore:"date"`
	Location  string    `db:"location" firestore:"location"`
	CreatedAt time.Time `db:"created_at" firestore:"createdAt"`
}

// GroupByDate groups trip locations by date, returning a map of date to locations.
func GroupByDate(locations []TripLocation) map[string][]string {
	grouped := make(map[string][]string)
	for _, loc := range locations {
		dateStr := loc.Date.Format("2006-01-02")
		grouped[dateStr] = append(grouped[dateStr], loc.Location)
	}
	return grouped
}

// ToAPITripDayLocations converts a slice of TripLocation entities to API TripDayLocation.
func ToAPITripDayLocations(locations []TripLocation) []api.TripDayLocation {
	grouped := GroupByDate(locations)
	result := make([]api.TripDayLocation, 0, len(grouped))
	for dateStr, locs := range grouped {
		date, _ := time.Parse("2006-01-02", dateStr)
		result = append(result, api.TripDayLocation{
			Date:      openapi_types.Date{Time: date},
			Locations: locs,
		})
	}
	return result
}

// TripLocationsFromRequest creates TripLocation entities from an API SetTripLocationsRequest.
func TripLocationsFromRequest(tripID uuid.UUID, startDate, endDate *time.Time, req *api.SetTripLocationsRequest) []TripLocation {
	var locations []TripLocation
	now := time.Now()

	// If there are explicit per-date locations, use those
	if req.Locations != nil {
		for _, dayLoc := range *req.Locations {
			for _, loc := range dayLoc.Locations {
				locations = append(locations, TripLocation{
					ID:        uuid.New(),
					TripID:    tripID,
					Date:      dayLoc.Date.Time,
					Location:  loc,
					CreatedAt: now,
				})
			}
		}
	}

	// If there's a default location and we have trip dates, fill in missing dates
	if req.DefaultLocation != nil && *req.DefaultLocation != "" && startDate != nil && endDate != nil {
		// Build a set of dates that already have locations
		coveredDates := make(map[string]bool)
		for _, loc := range locations {
			coveredDates[loc.Date.Format("2006-01-02")] = true
		}

		// Fill in dates that don't have explicit locations
		for d := *startDate; !d.After(*endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			if !coveredDates[dateStr] {
				locations = append(locations, TripLocation{
					ID:        uuid.New(),
					TripID:    tripID,
					Date:      d,
					Location:  *req.DefaultLocation,
					CreatedAt: now,
				})
			}
		}
	}

	return locations
}
