package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/travel-calendar/backend/internal/entity"
)

// =============================================================================
// Factory function for test store
// =============================================================================

func newSQLiteStore(t *testing.T) StoreInterface {
	s, err := NewSQLite(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

// =============================================================================
// Entry point: run shared tests against SQLite
// =============================================================================

func TestSQLiteStoreShared(t *testing.T) {
	runStoreTests(t, newSQLiteStore)
}

// =============================================================================
// Shared test helpers (work with StoreInterface)
// =============================================================================

func createTestTripVia(t *testing.T, s StoreInterface, name, purpose string, startDate, endDate *time.Time) entity.Trip {
	trip := entity.Trip{
		ID:        uuid.New(),
		Name:      name,
		Purpose:   purpose,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    "planning",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
	}
	err := s.CreateTrip(&trip)
	require.NoError(t, err)
	return trip
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

// =============================================================================
// Shared test suite
// =============================================================================

func runStoreTests(t *testing.T, newStore func(t *testing.T) StoreInterface) {
	t.Run("Trips", func(t *testing.T) {
		runTripTests(t, newStore)
	})
	t.Run("Items", func(t *testing.T) {
		runItemTests(t, newStore)
	})
	t.Run("Config", func(t *testing.T) {
		runConfigTests(t, newStore)
	})
	t.Run("TripLocations", func(t *testing.T) {
		runTripLocationTests(t, newStore)
	})
	t.Run("GoogleCredentials", func(t *testing.T) {
		runGoogleCredentialsTests(t, newStore)
	})
	t.Run("UserCalendars", func(t *testing.T) {
		runUserCalendarTests(t, newStore)
	})
	t.Run("CalendarLinks", func(t *testing.T) {
		runCalendarLinkTests(t, newStore)
	})
	t.Run("ProcessedEvents", func(t *testing.T) {
		runProcessedEventTests(t, newStore)
	})
}

// =============================================================================
// Trip Tests
// =============================================================================

func runTripTests(t *testing.T, newStore func(t *testing.T) StoreInterface) {
	t.Run("ListTrips_Empty", func(t *testing.T) {
		s := newStore(t)
		trips, err := s.ListTrips(nil, nil, nil)
		require.NoError(t, err)
		assert.Empty(t, trips)
	})

	t.Run("ListTrips_ReturnsAll", func(t *testing.T) {
		s := newStore(t)
		createTestTripVia(t, s, "Trip 1", "vacation", nil, nil)
		createTestTripVia(t, s, "Trip 2", "business", nil, nil)

		trips, err := s.ListTrips(nil, nil, nil)
		require.NoError(t, err)
		assert.Len(t, trips, 2)
	})

	t.Run("ListTrips_Upcoming", func(t *testing.T) {
		s := newStore(t)
		futureDate := time.Now().AddDate(0, 1, 0)
		createTestTripVia(t, s, "Future Trip", "vacation", timePtr(futureDate), nil)

		pastDate := time.Now().AddDate(0, -1, 0)
		pastEndDate := time.Now().AddDate(0, 0, -15)
		createTestTripVia(t, s, "Past Trip", "vacation", timePtr(pastDate), timePtr(pastEndDate))

		trips, err := s.ListTrips(boolPtr(true), nil, nil)
		require.NoError(t, err)
		assert.Len(t, trips, 1)
		assert.Equal(t, "Future Trip", trips[0].Name)
	})

	t.Run("ListTrips_Past", func(t *testing.T) {
		s := newStore(t)
		futureDate := time.Now().AddDate(0, 1, 0)
		futureEndDate := time.Now().AddDate(0, 1, 5)
		createTestTripVia(t, s, "Future Trip", "vacation", timePtr(futureDate), timePtr(futureEndDate))

		pastDate := time.Now().AddDate(0, -1, 0)
		pastEndDate := time.Now().AddDate(0, 0, -15)
		createTestTripVia(t, s, "Past Trip", "vacation", timePtr(pastDate), timePtr(pastEndDate))

		trips, err := s.ListTrips(nil, boolPtr(true), nil)
		require.NoError(t, err)
		assert.Len(t, trips, 1)
		assert.Equal(t, "Past Trip", trips[0].Name)
	})

	t.Run("ListTrips_ByPurpose", func(t *testing.T) {
		s := newStore(t)
		createTestTripVia(t, s, "Vacation Trip", "vacation", nil, nil)
		createTestTripVia(t, s, "Business Trip", "business", nil, nil)

		trips, err := s.ListTrips(nil, nil, strPtr("vacation"))
		require.NoError(t, err)
		assert.Len(t, trips, 1)
		assert.Equal(t, "Vacation Trip", trips[0].Name)
	})

	t.Run("ListTrips_SortedByStartDate", func(t *testing.T) {
		s := newStore(t)
		date3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
		date1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)

		createTestTripVia(t, s, "March Trip", "vacation", timePtr(date3), nil)
		createTestTripVia(t, s, "January Trip", "vacation", timePtr(date1), nil)
		createTestTripVia(t, s, "February Trip", "vacation", timePtr(date2), nil)

		trips, err := s.ListTrips(nil, nil, nil)
		require.NoError(t, err)
		require.Len(t, trips, 3)
		assert.Equal(t, "January Trip", trips[0].Name)
		assert.Equal(t, "February Trip", trips[1].Name)
		assert.Equal(t, "March Trip", trips[2].Name)
	})

	t.Run("ListTrips_NullDatesLast", func(t *testing.T) {
		s := newStore(t)
		createTestTripVia(t, s, "Unscheduled Trip", "vacation", nil, nil)
		date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		createTestTripVia(t, s, "Scheduled Trip", "vacation", timePtr(date), nil)

		trips, err := s.ListTrips(nil, nil, nil)
		require.NoError(t, err)
		require.Len(t, trips, 2)
		assert.Equal(t, "Scheduled Trip", trips[0].Name)
		assert.Equal(t, "Unscheduled Trip", trips[1].Name)
	})

	t.Run("GetTrip_Found", func(t *testing.T) {
		s := newStore(t)
		created := createTestTripVia(t, s, "Test Trip", "vacation", nil, nil)

		trip, err := s.GetTrip(created.ID)
		require.NoError(t, err)
		require.NotNil(t, trip)
		assert.Equal(t, "Test Trip", trip.Name)
		assert.Equal(t, "vacation", trip.Purpose)
	})

	t.Run("GetTrip_NotFound", func(t *testing.T) {
		s := newStore(t)
		trip, err := s.GetTrip(uuid.New())
		require.NoError(t, err)
		assert.Nil(t, trip)
	})

	t.Run("CreateTrip_Success", func(t *testing.T) {
		s := newStore(t)
		startDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
		notes := "Test notes"

		trip := entity.Trip{
			ID:        uuid.New(),
			Name:      "Summer Vacation",
			Purpose:   "vacation",
			StartDate: &startDate,
			EndDate:   &endDate,
			Status:    "planning",
			Notes:     &notes,
			CreatedAt: time.Now().UTC().Truncate(time.Second),
			UpdatedAt: time.Now().UTC().Truncate(time.Second),
		}

		err := s.CreateTrip(&trip)
		require.NoError(t, err)

		retrieved, err := s.GetTrip(trip.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, "Summer Vacation", retrieved.Name)
		assert.Equal(t, "vacation", retrieved.Purpose)
		assert.NotNil(t, retrieved.StartDate)
		assert.NotNil(t, retrieved.EndDate)
		assert.NotNil(t, retrieved.Notes)
		assert.Equal(t, "Test notes", *retrieved.Notes)
	})

	t.Run("CreateTrip_SetsTimestamps", func(t *testing.T) {
		s := newStore(t)
		now := time.Now().UTC().Truncate(time.Second)
		trip := entity.Trip{
			ID:        uuid.New(),
			Name:      "Test Trip",
			Purpose:   "vacation",
			Status:    "planning",
			CreatedAt: now,
			UpdatedAt: now,
		}

		err := s.CreateTrip(&trip)
		require.NoError(t, err)

		retrieved, err := s.GetTrip(trip.ID)
		require.NoError(t, err)
		assert.False(t, retrieved.CreatedAt.IsZero())
		assert.False(t, retrieved.UpdatedAt.IsZero())
	})

	t.Run("UpdateTrip_Success", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Original Name", "vacation", nil, nil)

		trip.Name = "Updated Name"
		trip.Purpose = "business"
		trip.UpdatedAt = time.Now().UTC().Truncate(time.Second)

		err := s.UpdateTrip(&trip)
		require.NoError(t, err)

		retrieved, err := s.GetTrip(trip.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", retrieved.Name)
		assert.Equal(t, "business", retrieved.Purpose)
	})

	t.Run("UpdateTrip_NotFound", func(t *testing.T) {
		s := newStore(t)
		trip := entity.Trip{
			ID:        uuid.New(),
			Name:      "Nonexistent",
			Purpose:   "vacation",
			Status:    "planning",
			UpdatedAt: time.Now(),
		}
		err := s.UpdateTrip(&trip)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("DeleteTrip_Success", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "To Delete", "vacation", nil, nil)

		err := s.DeleteTrip(trip.ID)
		require.NoError(t, err)

		retrieved, err := s.GetTrip(trip.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("DeleteTrip_NotFound", func(t *testing.T) {
		s := newStore(t)
		err := s.DeleteTrip(uuid.New())
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("DeleteTrip_CascadesItems", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip with Items", "vacation", nil, nil)

		item := entity.Item{
			ID:        uuid.New(),
			TripID:    trip.ID,
			Type:      "flight",
			CreatedAt: time.Now().UTC().Truncate(time.Second),
			UpdatedAt: time.Now().UTC().Truncate(time.Second),
		}
		err := s.CreateItem(&item)
		require.NoError(t, err)

		items, err := s.ListItems(trip.ID)
		require.NoError(t, err)
		assert.Len(t, items, 1)

		err = s.DeleteTrip(trip.ID)
		require.NoError(t, err)

		items, err = s.ListItems(trip.ID)
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("SearchTrips_ByName", func(t *testing.T) {
		s := newStore(t)
		createTestTripVia(t, s, "FOSDEM Conference", "conference", nil, nil)
		createTestTripVia(t, s, "Beach Vacation", "vacation", nil, nil)

		trips, err := s.SearchTrips("FOSDEM")
		require.NoError(t, err)
		assert.Len(t, trips, 1)
		assert.Equal(t, "FOSDEM Conference", trips[0].Name)
	})

	t.Run("SearchTrips_ByNotes", func(t *testing.T) {
		s := newStore(t)
		notes := "Meeting with important clients"
		trip := entity.Trip{
			ID:        uuid.New(),
			Name:      "Business Trip",
			Purpose:   "business",
			Status:    "planning",
			Notes:     &notes,
			CreatedAt: time.Now().UTC().Truncate(time.Second),
			UpdatedAt: time.Now().UTC().Truncate(time.Second),
		}
		err := s.CreateTrip(&trip)
		require.NoError(t, err)

		createTestTripVia(t, s, "Other Trip", "vacation", nil, nil)

		trips, err := s.SearchTrips("clients")
		require.NoError(t, err)
		assert.Len(t, trips, 1)
		assert.Equal(t, "Business Trip", trips[0].Name)
	})

	t.Run("SearchTrips_CaseInsensitive", func(t *testing.T) {
		s := newStore(t)
		createTestTripVia(t, s, "FOSDEM Conference", "conference", nil, nil)

		trips, err := s.SearchTrips("fosdem")
		require.NoError(t, err)
		assert.Len(t, trips, 1)
		assert.Equal(t, "FOSDEM Conference", trips[0].Name)
	})

	t.Run("SearchTrips_PartialMatch", func(t *testing.T) {
		s := newStore(t)
		createTestTripVia(t, s, "Summer Vacation 2025", "vacation", nil, nil)

		trips, err := s.SearchTrips("Vacat")
		require.NoError(t, err)
		assert.Len(t, trips, 1)
		assert.Equal(t, "Summer Vacation 2025", trips[0].Name)
	})
}

// =============================================================================
// Item Tests
// =============================================================================

func runItemTests(t *testing.T, newStore func(t *testing.T) StoreInterface) {
	t.Run("ListItems_Empty", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Empty Trip", "vacation", nil, nil)

		items, err := s.ListItems(trip.ID)
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("ListItems_SortedByDateTime", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip", "vacation", nil, nil)

		date2 := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
		date1 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

		item2 := entity.Item{
			ID:        uuid.New(),
			TripID:    trip.ID,
			Type:      "event",
			Date:      &date2,
			Time:      strPtr("10:00"),
			CreatedAt: time.Now().UTC().Truncate(time.Second),
			UpdatedAt: time.Now().UTC().Truncate(time.Second),
		}
		err := s.CreateItem(&item2)
		require.NoError(t, err)

		item1 := entity.Item{
			ID:        uuid.New(),
			TripID:    trip.ID,
			Type:      "flight",
			Date:      &date1,
			Time:      strPtr("08:00"),
			CreatedAt: time.Now().UTC().Truncate(time.Second),
			UpdatedAt: time.Now().UTC().Truncate(time.Second),
		}
		err = s.CreateItem(&item1)
		require.NoError(t, err)

		items, err := s.ListItems(trip.ID)
		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, "flight", items[0].Type)
		assert.Equal(t, "event", items[1].Type)
	})

	t.Run("GetItem_Found", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip", "vacation", nil, nil)

		item := entity.Item{
			ID:        uuid.New(),
			TripID:    trip.ID,
			Type:      "flight",
			From:      strPtr("JFK"),
			To:        strPtr("LAX"),
			CreatedAt: time.Now().UTC().Truncate(time.Second),
			UpdatedAt: time.Now().UTC().Truncate(time.Second),
		}
		err := s.CreateItem(&item)
		require.NoError(t, err)

		retrieved, err := s.GetItem(item.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, "flight", retrieved.Type)
		assert.Equal(t, "JFK", *retrieved.From)
		assert.Equal(t, "LAX", *retrieved.To)
	})

	t.Run("GetItem_NotFound", func(t *testing.T) {
		s := newStore(t)
		item, err := s.GetItem(uuid.New())
		require.NoError(t, err)
		assert.Nil(t, item)
	})

	t.Run("CreateItem_Success", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip", "vacation", nil, nil)
		date := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

		item := entity.Item{
			ID:           uuid.New(),
			TripID:       trip.ID,
			Type:         "flight",
			Date:         &date,
			Time:         strPtr("08:00"),
			From:         strPtr("JFK"),
			To:           strPtr("LAX"),
			Carrier:      strPtr("Delta"),
			FlightNumber: strPtr("DL123"),
			CreatedAt:    time.Now().UTC().Truncate(time.Second),
			UpdatedAt:    time.Now().UTC().Truncate(time.Second),
		}
		err := s.CreateItem(&item)
		require.NoError(t, err)

		retrieved, err := s.GetItem(item.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, "flight", retrieved.Type)
		assert.Equal(t, "JFK", *retrieved.From)
		assert.Equal(t, "LAX", *retrieved.To)
		assert.Equal(t, "Delta", *retrieved.Carrier)
		assert.Equal(t, "DL123", *retrieved.FlightNumber)
	})

	t.Run("CreateItem_OptionalFieldsNull", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip", "vacation", nil, nil)

		item := entity.Item{
			ID:        uuid.New(),
			TripID:    trip.ID,
			Type:      "event",
			CreatedAt: time.Now().UTC().Truncate(time.Second),
			UpdatedAt: time.Now().UTC().Truncate(time.Second),
		}
		err := s.CreateItem(&item)
		require.NoError(t, err)

		retrieved, err := s.GetItem(item.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, "event", retrieved.Type)
		assert.Nil(t, retrieved.Date)
		assert.Nil(t, retrieved.Time)
		assert.Nil(t, retrieved.From)
		assert.Nil(t, retrieved.To)
	})

	t.Run("DeleteItem_Success", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip", "vacation", nil, nil)

		item := entity.Item{
			ID:        uuid.New(),
			TripID:    trip.ID,
			Type:      "flight",
			CreatedAt: time.Now().UTC().Truncate(time.Second),
			UpdatedAt: time.Now().UTC().Truncate(time.Second),
		}
		err := s.CreateItem(&item)
		require.NoError(t, err)

		err = s.DeleteItem(item.ID)
		require.NoError(t, err)

		retrieved, err := s.GetItem(item.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("DeleteItem_NotFound", func(t *testing.T) {
		s := newStore(t)
		err := s.DeleteItem(uuid.New())
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("UpdateItemTrip", func(t *testing.T) {
		s := newStore(t)
		trip1 := createTestTripVia(t, s, "Trip 1", "vacation", nil, nil)
		trip2 := createTestTripVia(t, s, "Trip 2", "business", nil, nil)

		item := entity.Item{
			ID:        uuid.New(),
			TripID:    trip1.ID,
			Type:      "flight",
			CreatedAt: time.Now().UTC().Truncate(time.Second),
			UpdatedAt: time.Now().UTC().Truncate(time.Second),
		}
		err := s.CreateItem(&item)
		require.NoError(t, err)

		err = s.UpdateItemTrip(item.ID, trip2.ID)
		require.NoError(t, err)

		// Item should now be under trip2
		items, err := s.ListItems(trip2.ID)
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, item.ID, items[0].ID)

		// trip1 should have no items
		items, err = s.ListItems(trip1.ID)
		require.NoError(t, err)
		assert.Empty(t, items)
	})
}

// =============================================================================
// Config Tests
// =============================================================================

func runConfigTests(t *testing.T, newStore func(t *testing.T) StoreInterface) {
	t.Run("GetConfig_NotFound", func(t *testing.T) {
		s := newStore(t)
		val, err := s.GetConfig("nonexistent")
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("SetConfig_And_GetConfig", func(t *testing.T) {
		s := newStore(t)
		err := s.SetConfig("test_key", "test_value")
		require.NoError(t, err)

		val, err := s.GetConfig("test_key")
		require.NoError(t, err)
		require.NotNil(t, val)
		assert.Equal(t, "test_value", *val)
	})

	t.Run("SetConfig_Overwrite", func(t *testing.T) {
		s := newStore(t)
		err := s.SetConfig("key", "value1")
		require.NoError(t, err)

		err = s.SetConfig("key", "value2")
		require.NoError(t, err)

		val, err := s.GetConfig("key")
		require.NoError(t, err)
		require.NotNil(t, val)
		assert.Equal(t, "value2", *val)
	})

	t.Run("DeleteConfig", func(t *testing.T) {
		s := newStore(t)
		err := s.SetConfig("to_delete", "value")
		require.NoError(t, err)

		err = s.DeleteConfig("to_delete")
		require.NoError(t, err)

		val, err := s.GetConfig("to_delete")
		require.NoError(t, err)
		assert.Nil(t, val)
	})
}

// =============================================================================
// Trip Location Tests
// =============================================================================

func runTripLocationTests(t *testing.T, newStore func(t *testing.T) StoreInterface) {
	t.Run("GetTripLocations_Empty", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip", "vacation", nil, nil)

		locs, err := s.GetTripLocations(trip.ID)
		require.NoError(t, err)
		assert.Empty(t, locs)
	})

	t.Run("SetTripLocations_And_Get", func(t *testing.T) {
		s := newStore(t)
		start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
		trip := createTestTripVia(t, s, "Trip", "vacation", timePtr(start), timePtr(end))

		now := time.Now().UTC().Truncate(time.Second)
		locations := []entity.TripLocation{
			{
				ID:        uuid.New(),
				TripID:    trip.ID,
				Date:      time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
				Location:  "Paris",
				CreatedAt: now,
			},
			{
				ID:        uuid.New(),
				TripID:    trip.ID,
				Date:      time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC),
				Location:  "London",
				CreatedAt: now,
			},
		}

		err := s.SetTripLocations(trip.ID, locations)
		require.NoError(t, err)

		retrieved, err := s.GetTripLocations(trip.ID)
		require.NoError(t, err)
		require.Len(t, retrieved, 2)
		// Check both locations exist (order may vary by implementation)
		locNames := map[string]bool{}
		for _, loc := range retrieved {
			locNames[loc.Location] = true
		}
		assert.True(t, locNames["Paris"])
		assert.True(t, locNames["London"])
	})

	t.Run("SetTripLocations_Replaces", func(t *testing.T) {
		s := newStore(t)
		start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
		trip := createTestTripVia(t, s, "Trip", "vacation", timePtr(start), timePtr(end))

		now := time.Now().UTC().Truncate(time.Second)

		// Set initial locations
		err := s.SetTripLocations(trip.ID, []entity.TripLocation{
			{ID: uuid.New(), TripID: trip.ID, Date: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), Location: "Paris", CreatedAt: now},
		})
		require.NoError(t, err)

		// Replace with new locations
		err = s.SetTripLocations(trip.ID, []entity.TripLocation{
			{ID: uuid.New(), TripID: trip.ID, Date: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), Location: "Berlin", CreatedAt: now},
			{ID: uuid.New(), TripID: trip.ID, Date: time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC), Location: "Munich", CreatedAt: now},
		})
		require.NoError(t, err)

		retrieved, err := s.GetTripLocations(trip.ID)
		require.NoError(t, err)
		require.Len(t, retrieved, 2)
		locNames := map[string]bool{}
		for _, loc := range retrieved {
			locNames[loc.Location] = true
		}
		assert.True(t, locNames["Berlin"])
		assert.True(t, locNames["Munich"])
	})

	t.Run("GetTripsForDateRange", func(t *testing.T) {
		s := newStore(t)
		// Trip within range
		start1 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		end1 := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
		createTestTripVia(t, s, "June Trip", "vacation", timePtr(start1), timePtr(end1))

		// Trip outside range
		start2 := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
		end2 := time.Date(2025, 9, 10, 0, 0, 0, 0, time.UTC)
		createTestTripVia(t, s, "Sept Trip", "vacation", timePtr(start2), timePtr(end2))

		from := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
		trips, err := s.GetTripsForDateRange(from, to)
		require.NoError(t, err)
		assert.Len(t, trips, 1)
		assert.Equal(t, "June Trip", trips[0].Name)
	})
}

// =============================================================================
// Google Credentials Tests
// =============================================================================

func runGoogleCredentialsTests(t *testing.T, newStore func(t *testing.T) StoreInterface) {
	t.Run("GetGoogleCredentials_NotFound", func(t *testing.T) {
		s := newStore(t)
		creds, err := s.GetGoogleCredentials("nonexistent")
		require.NoError(t, err)
		assert.Nil(t, creds)
	})

	t.Run("SaveAndGetGoogleCredentials", func(t *testing.T) {
		s := newStore(t)
		now := time.Now().UTC().Truncate(time.Second)
		creds := &entity.GoogleCredentials{
			UserID:       "user1",
			AccessToken:  "access-token-123",
			RefreshToken: "refresh-token-456",
			TokenType:    "Bearer",
			ExpiresAt:    now.Add(time.Hour),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		err := s.SaveGoogleCredentials(creds)
		require.NoError(t, err)

		retrieved, err := s.GetGoogleCredentials("user1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, "access-token-123", retrieved.AccessToken)
		assert.Equal(t, "refresh-token-456", retrieved.RefreshToken)
	})

	t.Run("DeleteGoogleCredentials", func(t *testing.T) {
		s := newStore(t)
		now := time.Now().UTC().Truncate(time.Second)
		creds := &entity.GoogleCredentials{
			UserID:       "user2",
			AccessToken:  "token",
			RefreshToken: "refresh",
			TokenType:    "Bearer",
			ExpiresAt:    now.Add(time.Hour),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		err := s.SaveGoogleCredentials(creds)
		require.NoError(t, err)

		err = s.DeleteGoogleCredentials("user2")
		require.NoError(t, err)

		retrieved, err := s.GetGoogleCredentials("user2")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

// =============================================================================
// User Calendar Tests
// =============================================================================

func runUserCalendarTests(t *testing.T, newStore func(t *testing.T) StoreInterface) {
	t.Run("ListUserCalendars_Empty", func(t *testing.T) {
		s := newStore(t)
		cals, err := s.ListUserCalendars("user1")
		require.NoError(t, err)
		assert.Empty(t, cals)
	})

	t.Run("SaveAndListUserCalendars", func(t *testing.T) {
		s := newStore(t)
		now := time.Now().UTC().Truncate(time.Second)
		cal := &entity.UserCalendar{
			ID:         uuid.New(),
			UserID:     "user1",
			CalendarID: "cal-123",
			Name:       "Work Calendar",
			Enabled:    true,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		err := s.SaveUserCalendar(cal)
		require.NoError(t, err)

		cals, err := s.ListUserCalendars("user1")
		require.NoError(t, err)
		require.Len(t, cals, 1)
		assert.Equal(t, "Work Calendar", cals[0].Name)
		assert.True(t, cals[0].Enabled)
	})

	t.Run("GetUserCalendarByCalendarID", func(t *testing.T) {
		s := newStore(t)
		now := time.Now().UTC().Truncate(time.Second)
		cal := &entity.UserCalendar{
			ID:         uuid.New(),
			UserID:     "user1",
			CalendarID: "unique-cal-id",
			Name:       "My Calendar",
			Enabled:    true,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		err := s.SaveUserCalendar(cal)
		require.NoError(t, err)

		retrieved, err := s.GetUserCalendarByCalendarID("user1", "unique-cal-id")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, "My Calendar", retrieved.Name)
	})

	t.Run("DeleteUserCalendar", func(t *testing.T) {
		s := newStore(t)
		now := time.Now().UTC().Truncate(time.Second)
		cal := &entity.UserCalendar{
			ID:         uuid.New(),
			UserID:     "user1",
			CalendarID: "to-delete",
			Name:       "Delete Me",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		err := s.SaveUserCalendar(cal)
		require.NoError(t, err)

		err = s.DeleteUserCalendar(cal.ID)
		require.NoError(t, err)

		retrieved, err := s.GetUserCalendar(cal.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("SetUserCalendars_Replaces", func(t *testing.T) {
		s := newStore(t)
		now := time.Now().UTC().Truncate(time.Second)
		// Save initial
		err := s.SaveUserCalendar(&entity.UserCalendar{
			ID:         uuid.New(),
			UserID:     "user1",
			CalendarID: "old-cal",
			Name:       "Old",
			CreatedAt:  now,
			UpdatedAt:  now,
		})
		require.NoError(t, err)

		// Replace with SetUserCalendars
		newCals := []entity.UserCalendar{
			{
				ID:         uuid.New(),
				UserID:     "user1",
				CalendarID: "new-cal-1",
				Name:       "New 1",
				Enabled:    true,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
			{
				ID:         uuid.New(),
				UserID:     "user1",
				CalendarID: "new-cal-2",
				Name:       "New 2",
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		}
		err = s.SetUserCalendars("user1", newCals)
		require.NoError(t, err)

		cals, err := s.ListUserCalendars("user1")
		require.NoError(t, err)
		assert.Len(t, cals, 2)
	})
}

// =============================================================================
// Calendar Link Tests
// =============================================================================

func runCalendarLinkTests(t *testing.T, newStore func(t *testing.T) StoreInterface) {
	t.Run("ListCalendarLinks_Empty", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip", "vacation", nil, nil)

		links, err := s.ListCalendarLinks(trip.ID)
		require.NoError(t, err)
		assert.Empty(t, links)
	})

	t.Run("CreateAndListCalendarLinks", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip", "vacation", nil, nil)

		link := &entity.CalendarLink{
			ID:         uuid.New(),
			TripID:     trip.ID,
			CalendarID: "cal-123",
			EventID:    "event-456",
			SyncedAt:   time.Now().UTC().Truncate(time.Second),
		}
		err := s.CreateCalendarLink(link)
		require.NoError(t, err)

		links, err := s.ListCalendarLinks(trip.ID)
		require.NoError(t, err)
		require.Len(t, links, 1)
		assert.Equal(t, "event-456", links[0].EventID)
	})

	t.Run("GetCalendarLinkByEvent", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip", "vacation", nil, nil)

		link := &entity.CalendarLink{
			ID:         uuid.New(),
			TripID:     trip.ID,
			CalendarID: "cal-abc",
			EventID:    "event-xyz",
			SyncedAt:   time.Now().UTC().Truncate(time.Second),
		}
		err := s.CreateCalendarLink(link)
		require.NoError(t, err)

		retrieved, err := s.GetCalendarLinkByEvent(trip.ID, "cal-abc", "event-xyz")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, "cal-abc", retrieved.CalendarID)
		assert.Equal(t, "event-xyz", retrieved.EventID)
	})

	t.Run("DeleteCalendarLink", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip", "vacation", nil, nil)

		link := &entity.CalendarLink{
			ID:         uuid.New(),
			TripID:     trip.ID,
			CalendarID: "cal-del",
			EventID:    "event-del",
			SyncedAt:   time.Now().UTC().Truncate(time.Second),
		}
		err := s.CreateCalendarLink(link)
		require.NoError(t, err)

		err = s.DeleteCalendarLink(link.ID)
		require.NoError(t, err)

		retrieved, err := s.GetCalendarLink(link.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("DeleteCalendarLinksByTrip", func(t *testing.T) {
		s := newStore(t)
		trip := createTestTripVia(t, s, "Trip", "vacation", nil, nil)

		for i := 0; i < 3; i++ {
			link := &entity.CalendarLink{
				ID:         uuid.New(),
				TripID:     trip.ID,
				CalendarID: "cal",
				EventID:    fmt.Sprintf("event-%d", i),
				SyncedAt:   time.Now().UTC().Truncate(time.Second),
			}
			err := s.CreateCalendarLink(link)
			require.NoError(t, err)
		}

		err := s.DeleteCalendarLinksByTrip(trip.ID)
		require.NoError(t, err)

		links, err := s.ListCalendarLinks(trip.ID)
		require.NoError(t, err)
		assert.Empty(t, links)
	})
}

// =============================================================================
// Processed Event Tests
// =============================================================================

func runProcessedEventTests(t *testing.T, newStore func(t *testing.T) StoreInterface) {
	t.Run("IsEventProcessed_False", func(t *testing.T) {
		s := newStore(t)
		processed, err := s.IsEventProcessed("cal-1", "event-1")
		require.NoError(t, err)
		assert.False(t, processed)
	})

	t.Run("CreateProcessedEvent_And_IsProcessed", func(t *testing.T) {
		s := newStore(t)
		tripID := uuid.New()
		event := &entity.ProcessedCalendarEvent{
			ID:              uuid.New(),
			CalendarID:      "cal-1",
			CalendarEventID: "event-1",
			TripID:          &tripID,
			ProcessedAt:     time.Now().UTC().Truncate(time.Second),
			Action:          "imported",
		}
		err := s.CreateProcessedEvent(event)
		require.NoError(t, err)

		processed, err := s.IsEventProcessed("cal-1", "event-1")
		require.NoError(t, err)
		assert.True(t, processed)
	})

	t.Run("GetProcessedEventByCalendarEvent", func(t *testing.T) {
		s := newStore(t)
		tripID := uuid.New()
		event := &entity.ProcessedCalendarEvent{
			ID:              uuid.New(),
			CalendarID:      "cal-2",
			CalendarEventID: "event-2",
			TripID:          &tripID,
			ProcessedAt:     time.Now().UTC().Truncate(time.Second),
			Action:          "imported",
		}
		err := s.CreateProcessedEvent(event)
		require.NoError(t, err)

		retrieved, err := s.GetProcessedEventByCalendarEvent("cal-2", "event-2")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, "imported", retrieved.Action)
	})

	t.Run("ListProcessedEvents", func(t *testing.T) {
		s := newStore(t)
		tripID := uuid.New()
		for i := 0; i < 3; i++ {
			event := &entity.ProcessedCalendarEvent{
				ID:              uuid.New(),
				CalendarID:      "cal-list",
				CalendarEventID: fmt.Sprintf("event-%d", i),
				TripID:          &tripID,
				ProcessedAt:     time.Now().UTC().Truncate(time.Second),
				Action:          "imported",
			}
			err := s.CreateProcessedEvent(event)
			require.NoError(t, err)
		}

		events, err := s.ListProcessedEvents("cal-list")
		require.NoError(t, err)
		assert.Len(t, events, 3)
	})

	t.Run("DeleteAllProcessedEvents", func(t *testing.T) {
		s := newStore(t)
		tripID := uuid.New()
		event := &entity.ProcessedCalendarEvent{
			ID:              uuid.New(),
			CalendarID:      "cal-del",
			CalendarEventID: "event-del",
			TripID:          &tripID,
			ProcessedAt:     time.Now().UTC().Truncate(time.Second),
			Action:          "imported",
		}
		err := s.CreateProcessedEvent(event)
		require.NoError(t, err)

		err = s.DeleteAllProcessedEvents()
		require.NoError(t, err)

		processed, err := s.IsEventProcessed("cal-del", "event-del")
		require.NoError(t, err)
		assert.False(t, processed)
	})
}
