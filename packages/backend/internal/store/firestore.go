// Package store provides database access layer implementations.
package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/entity"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Compile-time check that FirestoreStore implements StoreInterface.
var _ StoreInterface = (*FirestoreStore)(nil)

// FirestoreStore provides database access methods using Google Cloud Firestore.
type FirestoreStore struct {
	client *firestore.Client
	ctx    context.Context
}

// Collection names as constants for consistency.
const (
	collTrips            = "trips"
	collItems            = "items"
	collDocuments        = "documents"
	collTripLocations    = "tripLocations"
	collConfig           = "config"
	collGoogleCreds      = "googleCredentials"
	collUserCalendars    = "userCalendars"
	collCalendarLinks    = "calendarLinks"
	collProcessedEvents  = "processedEvents"
	collSessions         = "sessions"
	collDayEntries       = "dayEntries"
)

// NewFirestore creates a new FirestoreStore with the given project ID.
// The FIRESTORE_EMULATOR_HOST environment variable is auto-detected by the SDK.
func NewFirestore(ctx context.Context, projectID string) (*FirestoreStore, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("creating firestore client: %w", err)
	}
	return &FirestoreStore{
		client: client,
		ctx:    ctx,
	}, nil
}

// Close closes the Firestore client connection.
func (s *FirestoreStore) Close() error {
	return s.client.Close()
}

// --- Trip methods ---

// ListTrips returns all trips, optionally filtered by upcoming/past/purpose.
func (s *FirestoreStore) ListTrips(userID string, upcoming, past *bool, purpose *string) ([]entity.Trip, error) {
	iter := s.client.Collection(collTrips).Where("userId", "==", userID).Documents(s.ctx)
	defer iter.Stop()

	now := time.Now().Format("2006-01-02")
	var trips []entity.Trip

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating trips: %w", err)
		}

		trip, err := docToTrip(doc)
		if err != nil {
			return nil, fmt.Errorf("parsing trip doc: %w", err)
		}

		// Apply filters
		if upcoming != nil && *upcoming {
			if trip.StartDate != nil && trip.StartDate.Format("2006-01-02") < now {
				continue
			}
		}
		if past != nil && *past {
			if trip.EndDate == nil || trip.EndDate.Format("2006-01-02") >= now {
				continue
			}
		}
		if purpose != nil && trip.Purpose != *purpose {
			continue
		}

		trips = append(trips, trip)
	}

	// Sort by start_date ASC, with nil dates last
	sort.Slice(trips, func(i, j int) bool {
		di := "9999-12-31"
		dj := "9999-12-31"
		if trips[i].StartDate != nil {
			di = trips[i].StartDate.Format("2006-01-02")
		}
		if trips[j].StartDate != nil {
			dj = trips[j].StartDate.Format("2006-01-02")
		}
		return di < dj
	})

	return trips, nil
}

// GetTrip returns a single trip by ID, or (nil, nil) if not found.
func (s *FirestoreStore) GetTrip(userID string, id uuid.UUID) (*entity.Trip, error) {
	doc, err := s.client.Collection(collTrips).Doc(id.String()).Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting trip: %w", err)
	}

	trip, err := docToTrip(doc)
	if err != nil {
		return nil, err
	}
	// Verify ownership
	if trip.UserID != userID {
		return nil, nil
	}
	return &trip, nil
}

// CreateTrip inserts a new trip document.
func (s *FirestoreStore) CreateTrip(trip *entity.Trip) error {
	_, err := s.client.Collection(collTrips).Doc(trip.ID.String()).Set(s.ctx, tripToMap(trip))
	if err != nil {
		return fmt.Errorf("creating trip: %w", err)
	}
	return nil
}

// UpdateTrip updates an existing trip. Returns ErrNotFound if trip doesn't exist or doesn't belong to the user.
func (s *FirestoreStore) UpdateTrip(userID string, trip *entity.Trip) error {
	docRef := s.client.Collection(collTrips).Doc(trip.ID.String())

	// Check existence and ownership
	doc, err := docRef.Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("checking trip existence: %w", err)
	}
	existing, err := docToTrip(doc)
	if err != nil {
		return fmt.Errorf("parsing existing trip: %w", err)
	}
	if existing.UserID != userID {
		return ErrNotFound
	}

	_, err = docRef.Set(s.ctx, tripToMap(trip))
	if err != nil {
		return fmt.Errorf("updating trip: %w", err)
	}
	return nil
}

// DeleteTrip deletes a trip and cascades to child items, locations, calendar links.
// Also clears tripId on associated documents (SET NULL equivalent).
func (s *FirestoreStore) DeleteTrip(userID string, id uuid.UUID) error {
	docRef := s.client.Collection(collTrips).Doc(id.String())

	// Check existence and ownership
	doc, err := docRef.Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("checking trip existence: %w", err)
	}
	existing, err := docToTrip(doc)
	if err != nil {
		return fmt.Errorf("parsing existing trip: %w", err)
	}
	if existing.UserID != userID {
		return ErrNotFound
	}

	batch := s.client.Batch()

	// Delete child items
	itemIter := s.client.Collection(collItems).Where("tripId", "==", id.String()).Documents(s.ctx)
	for {
		doc, err := itemIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			itemIter.Stop()
			return fmt.Errorf("iterating items for delete: %w", err)
		}
		batch.Delete(doc.Ref)
	}
	itemIter.Stop()

	// Delete child trip locations
	locIter := s.client.Collection(collTripLocations).Where("tripId", "==", id.String()).Documents(s.ctx)
	for {
		doc, err := locIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			locIter.Stop()
			return fmt.Errorf("iterating locations for delete: %w", err)
		}
		batch.Delete(doc.Ref)
	}
	locIter.Stop()

	// Delete child calendar links
	linkIter := s.client.Collection(collCalendarLinks).Where("tripId", "==", id.String()).Documents(s.ctx)
	for {
		doc, err := linkIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			linkIter.Stop()
			return fmt.Errorf("iterating calendar links for delete: %w", err)
		}
		batch.Delete(doc.Ref)
	}
	linkIter.Stop()

	// Clear tripId on associated documents (ON DELETE SET NULL)
	docIter := s.client.Collection(collDocuments).Where("tripId", "==", id.String()).Documents(s.ctx)
	for {
		doc, err := docIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			docIter.Stop()
			return fmt.Errorf("iterating documents for delete: %w", err)
		}
		batch.Update(doc.Ref, []firestore.Update{
			{Path: "tripId", Value: nil},
		})
	}
	docIter.Stop()

	// Delete the trip itself
	batch.Delete(docRef)

	_, err = batch.Commit(s.ctx)
	if err != nil {
		return fmt.Errorf("committing trip delete batch: %w", err)
	}
	return nil
}

// SearchTrips searches trips by name or notes containing the query string.
func (s *FirestoreStore) SearchTrips(userID string, q string) ([]entity.Trip, error) {
	qLower := strings.ToLower(q)
	iter := s.client.Collection(collTrips).Where("userId", "==", userID).Documents(s.ctx)
	defer iter.Stop()

	var trips []entity.Trip
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating trips for search: %w", err)
		}

		trip, err := docToTrip(doc)
		if err != nil {
			return nil, err
		}

		nameMatch := strings.Contains(strings.ToLower(trip.Name), qLower)
		notesMatch := trip.Notes != nil && strings.Contains(strings.ToLower(*trip.Notes), qLower)
		if nameMatch || notesMatch {
			trips = append(trips, trip)
		}
	}

	// Sort by start_date ASC, nil dates last
	sort.Slice(trips, func(i, j int) bool {
		di := "9999-12-31"
		dj := "9999-12-31"
		if trips[i].StartDate != nil {
			di = trips[i].StartDate.Format("2006-01-02")
		}
		if trips[j].StartDate != nil {
			dj = trips[j].StartDate.Format("2006-01-02")
		}
		return di < dj
	})

	return trips, nil
}

// --- Item methods ---

// ListItems returns all items for a trip, sorted by date then time.
func (s *FirestoreStore) ListItems(tripID uuid.UUID) ([]entity.Item, error) {
	iter := s.client.Collection(collItems).Where("tripId", "==", tripID.String()).Documents(s.ctx)
	defer iter.Stop()

	var items []entity.Item
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating items: %w", err)
		}

		item, err := docToItem(doc)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	// Sort by date ASC (nil last), then time ASC
	sort.Slice(items, func(i, j int) bool {
		di := "9999-12-31"
		dj := "9999-12-31"
		if items[i].Date != nil {
			di = items[i].Date.Format("2006-01-02")
		}
		if items[j].Date != nil {
			dj = items[j].Date.Format("2006-01-02")
		}
		if di != dj {
			return di < dj
		}
		ti := ""
		tj := ""
		if items[i].Time != nil {
			ti = *items[i].Time
		}
		if items[j].Time != nil {
			tj = *items[j].Time
		}
		return ti < tj
	})

	return items, nil
}

// GetItem returns a single item by ID, or (nil, nil) if not found.
func (s *FirestoreStore) GetItem(id uuid.UUID) (*entity.Item, error) {
	doc, err := s.client.Collection(collItems).Doc(id.String()).Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting item: %w", err)
	}

	item, err := docToItem(doc)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateItem inserts a new item document.
func (s *FirestoreStore) CreateItem(item *entity.Item) error {
	_, err := s.client.Collection(collItems).Doc(item.ID.String()).Set(s.ctx, itemToMap(item))
	if err != nil {
		return fmt.Errorf("creating item: %w", err)
	}
	return nil
}

// DeleteItem deletes an item by ID. Returns ErrNotFound if not found.
func (s *FirestoreStore) DeleteItem(id uuid.UUID) error {
	docRef := s.client.Collection(collItems).Doc(id.String())

	_, err := docRef.Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("checking item existence: %w", err)
	}

	_, err = docRef.Delete(s.ctx)
	if err != nil {
		return fmt.Errorf("deleting item: %w", err)
	}
	return nil
}

// UpdateItemTrip updates an item's trip assignment. Returns ErrNotFound if not found.
func (s *FirestoreStore) UpdateItemTrip(itemID, newTripID uuid.UUID) error {
	docRef := s.client.Collection(collItems).Doc(itemID.String())

	_, err := docRef.Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("checking item existence: %w", err)
	}

	_, err = docRef.Update(s.ctx, []firestore.Update{
		{Path: "tripId", Value: newTripID.String()},
		{Path: "updatedAt", Value: time.Now()},
	})
	if err != nil {
		return fmt.Errorf("updating item trip: %w", err)
	}
	return nil
}

// --- Trip Organization methods ---

// MergeTripsTransaction moves items from source to target, merges locations,
// and deletes the source trip, all within a Firestore transaction.
func (s *FirestoreStore) MergeTripsTransaction(sourceID, targetID uuid.UUID) error {
	return s.client.RunTransaction(s.ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		now := time.Now()

		// Move all items from source to target
		itemIter := s.client.Collection(collItems).Where("tripId", "==", sourceID.String()).Documents(s.ctx)
		defer itemIter.Stop()
		for {
			doc, err := itemIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("iterating source items: %w", err)
			}
			err = tx.Update(doc.Ref, []firestore.Update{
				{Path: "tripId", Value: targetID.String()},
				{Path: "updatedAt", Value: now},
			})
			if err != nil {
				return fmt.Errorf("moving item: %w", err)
			}
		}

		// Get target location dates to avoid duplicates
		targetLocIter := s.client.Collection(collTripLocations).Where("tripId", "==", targetID.String()).Documents(s.ctx)
		defer targetLocIter.Stop()
		targetDates := make(map[string]bool)
		for {
			doc, err := targetLocIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("iterating target locations: %w", err)
			}
			data := doc.Data()
			if dateStr, ok := data["date"].(string); ok {
				targetDates[dateStr] = true
			}
		}

		// Copy source locations for dates not in target
		sourceLocIter := s.client.Collection(collTripLocations).Where("tripId", "==", sourceID.String()).Documents(s.ctx)
		defer sourceLocIter.Stop()
		for {
			doc, err := sourceLocIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("iterating source locations: %w", err)
			}
			data := doc.Data()
			dateStr, _ := data["date"].(string)
			if !targetDates[dateStr] {
				newID := uuid.New()
				newRef := s.client.Collection(collTripLocations).Doc(newID.String())
				err = tx.Set(newRef, map[string]interface{}{
					"tripId":    targetID.String(),
					"date":      dateStr,
					"location":  data["location"],
					"createdAt": now,
				})
				if err != nil {
					return fmt.Errorf("inserting merged location: %w", err)
				}
				targetDates[dateStr] = true
			}
			// Delete source location
			err = tx.Delete(doc.Ref)
			if err != nil {
				return fmt.Errorf("deleting source location: %w", err)
			}
		}

		// Delete source calendar links
		linkIter := s.client.Collection(collCalendarLinks).Where("tripId", "==", sourceID.String()).Documents(s.ctx)
		defer linkIter.Stop()
		for {
			doc, err := linkIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("iterating source calendar links: %w", err)
			}
			err = tx.Delete(doc.Ref)
			if err != nil {
				return fmt.Errorf("deleting source calendar link: %w", err)
			}
		}

		// Clear tripId on associated documents
		docIter := s.client.Collection(collDocuments).Where("tripId", "==", sourceID.String()).Documents(s.ctx)
		defer docIter.Stop()
		for {
			doc, err := docIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("iterating source documents: %w", err)
			}
			err = tx.Update(doc.Ref, []firestore.Update{
				{Path: "tripId", Value: nil},
			})
			if err != nil {
				return fmt.Errorf("clearing document tripId: %w", err)
			}
		}

		// Delete the source trip
		sourceRef := s.client.Collection(collTrips).Doc(sourceID.String())
		return tx.Delete(sourceRef)
	})
}

// --- Document methods ---

// ListDocuments returns documents, optionally filtered by tripID or unassociated.
func (s *FirestoreStore) ListDocuments(tripID *uuid.UUID, unassociated *bool) ([]entity.Document, error) {
	var iter *firestore.DocumentIterator

	if tripID != nil {
		iter = s.client.Collection(collDocuments).Where("tripId", "==", tripID.String()).Documents(s.ctx)
	} else {
		iter = s.client.Collection(collDocuments).Documents(s.ctx)
	}
	defer iter.Stop()

	var docs []entity.Document
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating documents: %w", err)
		}

		d, err := docToDocument(doc)
		if err != nil {
			return nil, err
		}

		// Filter unassociated if requested
		if unassociated != nil && *unassociated && d.TripID != nil {
			continue
		}

		docs = append(docs, d)
	}

	// Sort by created_at DESC
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].CreatedAt.After(docs[j].CreatedAt)
	})

	return docs, nil
}

// --- Config methods ---

// GetConfig retrieves a config value by key. Returns (nil, nil) if not found.
func (s *FirestoreStore) GetConfig(key string) (*string, error) {
	doc, err := s.client.Collection(collConfig).Doc(key).Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting config: %w", err)
	}

	data := doc.Data()
	value, ok := data["value"].(string)
	if !ok {
		return nil, nil
	}
	return &value, nil
}

// SetConfig sets a config value (upsert).
func (s *FirestoreStore) SetConfig(key, value string) error {
	_, err := s.client.Collection(collConfig).Doc(key).Set(s.ctx, map[string]interface{}{
		"value":     value,
		"updatedAt": time.Now(),
	})
	if err != nil {
		return fmt.Errorf("setting config: %w", err)
	}
	return nil
}

// DeleteConfig removes a config value.
func (s *FirestoreStore) DeleteConfig(key string) error {
	_, err := s.client.Collection(collConfig).Doc(key).Delete(s.ctx)
	if err != nil {
		return fmt.Errorf("deleting config: %w", err)
	}
	return nil
}

// --- Trip Location methods ---

// GetTripLocations returns all locations for a trip, sorted by date then location.
func (s *FirestoreStore) GetTripLocations(tripID uuid.UUID) ([]entity.TripLocation, error) {
	iter := s.client.Collection(collTripLocations).Where("tripId", "==", tripID.String()).Documents(s.ctx)
	defer iter.Stop()

	var locations []entity.TripLocation
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating trip locations: %w", err)
		}

		loc, err := docToTripLocation(doc)
		if err != nil {
			return nil, err
		}
		locations = append(locations, loc)
	}

	sort.Slice(locations, func(i, j int) bool {
		if !locations[i].Date.Equal(locations[j].Date) {
			return locations[i].Date.Before(locations[j].Date)
		}
		return locations[i].Location < locations[j].Location
	})

	return locations, nil
}

// SetTripLocations replaces all locations for a trip (delete-all then insert-all).
func (s *FirestoreStore) SetTripLocations(tripID uuid.UUID, locations []entity.TripLocation) error {
	batch := s.client.Batch()

	// Delete existing locations for this trip
	iter := s.client.Collection(collTripLocations).Where("tripId", "==", tripID.String()).Documents(s.ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iter.Stop()
			return fmt.Errorf("iterating locations for delete: %w", err)
		}
		batch.Delete(doc.Ref)
	}
	iter.Stop()

	// Insert new locations
	for _, loc := range locations {
		docRef := s.client.Collection(collTripLocations).Doc(loc.ID.String())
		batch.Set(docRef, map[string]interface{}{
			"tripId":    loc.TripID.String(),
			"date":      loc.Date.Format("2006-01-02"),
			"location":  loc.Location,
			"createdAt": loc.CreatedAt,
		})
	}

	_, err := batch.Commit(s.ctx)
	if err != nil {
		return fmt.Errorf("committing set trip locations batch: %w", err)
	}
	return nil
}

// GetTripsForDateRange returns trips that overlap with the given date range.
func (s *FirestoreStore) GetTripsForDateRange(userID string, from, to time.Time) ([]entity.Trip, error) {
	// Firestore can only do inequality on one field, so we fetch trips with startDate <= to
	// and filter endDate >= from in Go.
	toStr := to.Format("2006-01-02")
	fromStr := from.Format("2006-01-02")

	iter := s.client.Collection(collTrips).Where("userId", "==", userID).Documents(s.ctx)
	defer iter.Stop()

	var trips []entity.Trip
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating trips for date range: %w", err)
		}

		trip, err := docToTrip(doc)
		if err != nil {
			return nil, err
		}

		// Both dates must be present
		if trip.StartDate == nil || trip.EndDate == nil {
			continue
		}

		startStr := trip.StartDate.Format("2006-01-02")
		endStr := trip.EndDate.Format("2006-01-02")

		// Overlap: startDate <= to AND endDate >= from
		if startStr <= toStr && endStr >= fromStr {
			trips = append(trips, trip)
		}
	}

	// Sort by start_date ASC
	sort.Slice(trips, func(i, j int) bool {
		return trips[i].StartDate.Before(*trips[j].StartDate)
	})

	return trips, nil
}

// GetTripLocationsForDateRange returns locations for a trip within a date range.
func (s *FirestoreStore) GetTripLocationsForDateRange(tripID uuid.UUID, from, to time.Time) ([]entity.TripLocation, error) {
	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")

	iter := s.client.Collection(collTripLocations).
		Where("tripId", "==", tripID.String()).
		Where("date", ">=", fromStr).
		Where("date", "<=", toStr).
		Documents(s.ctx)
	defer iter.Stop()

	var locations []entity.TripLocation
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating trip locations for date range: %w", err)
		}

		loc, err := docToTripLocation(doc)
		if err != nil {
			return nil, err
		}
		locations = append(locations, loc)
	}

	sort.Slice(locations, func(i, j int) bool {
		if !locations[i].Date.Equal(locations[j].Date) {
			return locations[i].Date.Before(locations[j].Date)
		}
		return locations[i].Location < locations[j].Location
	})

	return locations, nil
}

// --- Google Credentials methods ---

// GetGoogleCredentials retrieves OAuth credentials for a user. Returns (nil, nil) if not found.
func (s *FirestoreStore) GetGoogleCredentials(userID string) (*entity.GoogleCredentials, error) {
	doc, err := s.client.Collection(collGoogleCreds).Doc(userID).Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting google credentials: %w", err)
	}

	creds, err := docToGoogleCredentials(doc, userID)
	if err != nil {
		return nil, err
	}
	return &creds, nil
}

// SaveGoogleCredentials inserts or updates OAuth credentials.
func (s *FirestoreStore) SaveGoogleCredentials(creds *entity.GoogleCredentials) error {
	_, err := s.client.Collection(collGoogleCreds).Doc(creds.UserID).Set(s.ctx, map[string]interface{}{
		"accessToken":  creds.AccessToken,
		"refreshToken": creds.RefreshToken,
		"tokenType":    creds.TokenType,
		"expiresAt":    creds.ExpiresAt,
		"scopes":       creds.Scopes,
		"email":        creds.Email,
		"createdAt":    creds.CreatedAt,
		"updatedAt":    creds.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("saving google credentials: %w", err)
	}
	return nil
}

// DeleteGoogleCredentials removes OAuth credentials for a user. Returns ErrNotFound if not found.
func (s *FirestoreStore) DeleteGoogleCredentials(userID string) error {
	docRef := s.client.Collection(collGoogleCreds).Doc(userID)

	_, err := docRef.Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("checking google credentials existence: %w", err)
	}

	_, err = docRef.Delete(s.ctx)
	if err != nil {
		return fmt.Errorf("deleting google credentials: %w", err)
	}
	return nil
}

// --- User Calendar methods ---

// ListUserCalendars returns all selected calendars for a user, sorted by name.
func (s *FirestoreStore) ListUserCalendars(userID string) ([]entity.UserCalendar, error) {
	iter := s.client.Collection(collUserCalendars).Where("userId", "==", userID).Documents(s.ctx)
	defer iter.Stop()

	var calendars []entity.UserCalendar
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating user calendars: %w", err)
		}

		cal, err := docToUserCalendar(doc)
		if err != nil {
			return nil, err
		}
		calendars = append(calendars, cal)
	}

	sort.Slice(calendars, func(i, j int) bool {
		return calendars[i].Name < calendars[j].Name
	})

	return calendars, nil
}

// GetUserCalendar retrieves a specific user calendar by ID. Returns (nil, nil) if not found.
func (s *FirestoreStore) GetUserCalendar(id uuid.UUID) (*entity.UserCalendar, error) {
	// UserCalendars use composite IDs (userId_calendarId), but this method looks up by UUID.
	// We need to search all user calendars for a matching ID field.
	iter := s.client.Collection(collUserCalendars).Documents(s.ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("iterating user calendars: %w", err)
		}

		cal, err := docToUserCalendar(doc)
		if err != nil {
			return nil, err
		}
		if cal.ID == id {
			return &cal, nil
		}
	}
}

// GetUserCalendarByCalendarID retrieves a user calendar by Google Calendar ID.
// Returns (nil, nil) if not found.
func (s *FirestoreStore) GetUserCalendarByCalendarID(userID, calendarID string) (*entity.UserCalendar, error) {
	docID := userID + "_" + calendarID
	doc, err := s.client.Collection(collUserCalendars).Doc(docID).Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user calendar: %w", err)
	}

	cal, err := docToUserCalendar(doc)
	if err != nil {
		return nil, err
	}
	return &cal, nil
}

// SaveUserCalendar inserts or updates a user calendar.
func (s *FirestoreStore) SaveUserCalendar(cal *entity.UserCalendar) error {
	docID := cal.UserID + "_" + cal.CalendarID
	_, err := s.client.Collection(collUserCalendars).Doc(docID).Set(s.ctx, userCalendarToMap(cal))
	if err != nil {
		return fmt.Errorf("saving user calendar: %w", err)
	}
	return nil
}

// DeleteUserCalendar removes a user calendar by UUID. Returns ErrNotFound if not found.
func (s *FirestoreStore) DeleteUserCalendar(id uuid.UUID) error {
	// Need to find the doc by UUID since doc ID is composite
	iter := s.client.Collection(collUserCalendars).Documents(s.ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("iterating user calendars for delete: %w", err)
		}

		cal, err := docToUserCalendar(doc)
		if err != nil {
			return err
		}
		if cal.ID == id {
			_, err = doc.Ref.Delete(s.ctx)
			if err != nil {
				return fmt.Errorf("deleting user calendar: %w", err)
			}
			return nil
		}
	}
}

// DeleteUserCalendarsByUser removes all calendars for a user.
func (s *FirestoreStore) DeleteUserCalendarsByUser(userID string) error {
	iter := s.client.Collection(collUserCalendars).Where("userId", "==", userID).Documents(s.ctx)
	defer iter.Stop()

	batch := s.client.Batch()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("iterating user calendars for delete: %w", err)
		}
		batch.Delete(doc.Ref)
	}

	_, err := batch.Commit(s.ctx)
	if err != nil {
		return fmt.Errorf("committing delete user calendars batch: %w", err)
	}
	return nil
}

// SetUserCalendars replaces all calendars for a user (delete-all then insert-all).
func (s *FirestoreStore) SetUserCalendars(userID string, calendars []entity.UserCalendar) error {
	batch := s.client.Batch()

	// Delete existing calendars for this user
	iter := s.client.Collection(collUserCalendars).Where("userId", "==", userID).Documents(s.ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iter.Stop()
			return fmt.Errorf("iterating calendars for delete: %w", err)
		}
		batch.Delete(doc.Ref)
	}
	iter.Stop()

	// Insert new calendars
	for i := range calendars {
		cal := &calendars[i]
		docID := cal.UserID + "_" + cal.CalendarID
		docRef := s.client.Collection(collUserCalendars).Doc(docID)
		batch.Set(docRef, userCalendarToMap(cal))
	}

	_, err := batch.Commit(s.ctx)
	if err != nil {
		return fmt.Errorf("committing set user calendars batch: %w", err)
	}
	return nil
}

// --- Calendar Link methods ---

// ListCalendarLinks returns all calendar links for a trip, sorted by synced_at DESC.
func (s *FirestoreStore) ListCalendarLinks(tripID uuid.UUID) ([]entity.CalendarLink, error) {
	iter := s.client.Collection(collCalendarLinks).Where("tripId", "==", tripID.String()).Documents(s.ctx)
	defer iter.Stop()

	var links []entity.CalendarLink
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating calendar links: %w", err)
		}

		link, err := docToCalendarLink(doc)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	sort.Slice(links, func(i, j int) bool {
		return links[i].SyncedAt.After(links[j].SyncedAt)
	})

	return links, nil
}

// GetCalendarLink retrieves a specific calendar link by ID. Returns (nil, nil) if not found.
func (s *FirestoreStore) GetCalendarLink(id uuid.UUID) (*entity.CalendarLink, error) {
	// Calendar links use composite IDs, so we need to search by UUID field
	iter := s.client.Collection(collCalendarLinks).Documents(s.ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("iterating calendar links: %w", err)
		}

		link, err := docToCalendarLink(doc)
		if err != nil {
			return nil, err
		}
		if link.ID == id {
			return &link, nil
		}
	}
}

// GetCalendarLinkByEvent retrieves a calendar link by trip, calendar, and event IDs.
// Returns (nil, nil) if not found.
func (s *FirestoreStore) GetCalendarLinkByEvent(tripID uuid.UUID, calendarID, eventID string) (*entity.CalendarLink, error) {
	docID := tripID.String() + "_" + calendarID + "_" + eventID
	doc, err := s.client.Collection(collCalendarLinks).Doc(docID).Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting calendar link by event: %w", err)
	}

	link, err := docToCalendarLink(doc)
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// CreateCalendarLink inserts a new calendar link.
func (s *FirestoreStore) CreateCalendarLink(link *entity.CalendarLink) error {
	docID := link.TripID.String() + "_" + link.CalendarID + "_" + link.EventID
	_, err := s.client.Collection(collCalendarLinks).Doc(docID).Set(s.ctx, calendarLinkToMap(link))
	if err != nil {
		return fmt.Errorf("creating calendar link: %w", err)
	}
	return nil
}

// DeleteCalendarLink removes a calendar link by ID. Returns ErrNotFound if not found.
func (s *FirestoreStore) DeleteCalendarLink(id uuid.UUID) error {
	// Calendar links use composite IDs, need to search by UUID field
	iter := s.client.Collection(collCalendarLinks).Documents(s.ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("iterating calendar links for delete: %w", err)
		}

		link, err := docToCalendarLink(doc)
		if err != nil {
			return err
		}
		if link.ID == id {
			_, err = doc.Ref.Delete(s.ctx)
			if err != nil {
				return fmt.Errorf("deleting calendar link: %w", err)
			}
			return nil
		}
	}
}

// DeleteCalendarLinksByTrip removes all calendar links for a trip.
func (s *FirestoreStore) DeleteCalendarLinksByTrip(tripID uuid.UUID) error {
	iter := s.client.Collection(collCalendarLinks).Where("tripId", "==", tripID.String()).Documents(s.ctx)
	defer iter.Stop()

	batch := s.client.Batch()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("iterating calendar links for delete: %w", err)
		}
		batch.Delete(doc.Ref)
	}

	_, err := batch.Commit(s.ctx)
	if err != nil {
		return fmt.Errorf("committing delete calendar links batch: %w", err)
	}
	return nil
}

// --- Processed Calendar Events methods ---

// CreateProcessedEvent saves a record of a processed calendar event.
func (s *FirestoreStore) CreateProcessedEvent(event *entity.ProcessedCalendarEvent) error {
	docID := event.CalendarID + "_" + event.CalendarEventID
	data := map[string]interface{}{
		"id":              event.ID.String(),
		"userId":          event.UserID,
		"calendarEventId": event.CalendarEventID,
		"calendarId":      event.CalendarID,
		"action":          event.Action,
		"processedAt":     event.ProcessedAt,
	}
	if event.TripID != nil {
		data["tripId"] = event.TripID.String()
	}
	if event.ItemID != nil {
		data["itemId"] = event.ItemID.String()
	}

	_, err := s.client.Collection(collProcessedEvents).Doc(docID).Set(s.ctx, data)
	if err != nil {
		return fmt.Errorf("creating processed event: %w", err)
	}
	return nil
}

// GetProcessedEventByCalendarEvent retrieves a processed event by its calendar event ID.
// Returns (nil, nil) if not found.
func (s *FirestoreStore) GetProcessedEventByCalendarEvent(userID string, calendarID, eventID string) (*entity.ProcessedCalendarEvent, error) {
	docID := calendarID + "_" + eventID
	doc, err := s.client.Collection(collProcessedEvents).Doc(docID).Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting processed event: %w", err)
	}

	event, err := docToProcessedEvent(doc)
	if err != nil {
		return nil, err
	}
	// Verify ownership
	if event.UserID != userID {
		return nil, nil
	}
	return &event, nil
}

// IsEventProcessed checks if a calendar event has already been processed.
func (s *FirestoreStore) IsEventProcessed(userID string, calendarID, eventID string) (bool, error) {
	docID := calendarID + "_" + eventID
	doc, err := s.client.Collection(collProcessedEvents).Doc(docID).Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking processed event: %w", err)
	}
	// Verify ownership
	event, err := docToProcessedEvent(doc)
	if err != nil {
		return false, err
	}
	if event.UserID != userID {
		return false, nil
	}
	return true, nil
}

// ListProcessedEvents returns all processed events for a calendar, sorted by processedAt DESC.
func (s *FirestoreStore) ListProcessedEvents(userID string, calendarID string) ([]entity.ProcessedCalendarEvent, error) {
	iter := s.client.Collection(collProcessedEvents).Where("userId", "==", userID).Where("calendarId", "==", calendarID).Documents(s.ctx)
	defer iter.Stop()

	var events []entity.ProcessedCalendarEvent
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating processed events: %w", err)
		}

		event, err := docToProcessedEvent(doc)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].ProcessedAt.After(events[j].ProcessedAt)
	})

	return events, nil
}

// DeleteAllProcessedEvents removes all processed event records for a user.
func (s *FirestoreStore) DeleteAllProcessedEvents(userID string) error {
	iter := s.client.Collection(collProcessedEvents).Where("userId", "==", userID).Documents(s.ctx)
	defer iter.Stop()

	batch := s.client.Batch()
	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("iterating processed events for delete: %w", err)
		}
		batch.Delete(doc.Ref)
		count++

		// Firestore batches limited to 500 writes
		if count >= 500 {
			_, err = batch.Commit(s.ctx)
			if err != nil {
				return fmt.Errorf("committing delete batch: %w", err)
			}
			batch = s.client.Batch()
			count = 0
		}
	}

	if count > 0 {
		_, err := batch.Commit(s.ctx)
		if err != nil {
			return fmt.Errorf("committing final delete batch: %w", err)
		}
	}
	return nil
}

// --- Session methods ---

// CreateSession inserts a new session.
func (s *FirestoreStore) CreateSession(session *entity.Session) error {
	_, err := s.client.Collection(collSessions).Doc(session.ID).Set(s.ctx, map[string]interface{}{
		"userId":    session.UserID,
		"email":     session.Email,
		"expiresAt": session.ExpiresAt,
		"createdAt": session.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	return nil
}

// GetSession retrieves a session by ID.
func (s *FirestoreStore) GetSession(id string) (*entity.Session, error) {
	doc, err := s.client.Collection(collSessions).Doc(id).Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}

	data := doc.Data()
	session := &entity.Session{
		ID:        doc.Ref.ID,
		UserID:    data["userId"].(string),
		Email:     data["email"].(string),
		ExpiresAt: toTime(data["expiresAt"]),
		CreatedAt: toTime(data["createdAt"]),
	}
	return session, nil
}

// DeleteSession removes a session by ID.
func (s *FirestoreStore) DeleteSession(id string) error {
	_, err := s.client.Collection(collSessions).Doc(id).Delete(s.ctx)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes all expired sessions.
func (s *FirestoreStore) DeleteExpiredSessions() error {
	iter := s.client.Collection(collSessions).Where("expiresAt", "<", time.Now()).Documents(s.ctx)
	defer iter.Stop()

	batch := s.client.Batch()
	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("iterating expired sessions: %w", err)
		}
		batch.Delete(doc.Ref)
		count++
		if count >= 500 {
			if _, err = batch.Commit(s.ctx); err != nil {
				return fmt.Errorf("committing session delete batch: %w", err)
			}
			batch = s.client.Batch()
			count = 0
		}
	}
	if count > 0 {
		if _, err := batch.Commit(s.ctx); err != nil {
			return fmt.Errorf("committing final session delete batch: %w", err)
		}
	}
	return nil
}

// --- Day Entry methods ---

// ListDayEntries returns day entries for a user within a date range.
func (s *FirestoreStore) ListDayEntries(userID string, from, to time.Time) ([]entity.DayEntry, error) {
	iter := s.client.Collection(collDayEntries).Where("userId", "==", userID).Documents(s.ctx)
	defer iter.Stop()

	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")
	var entries []entity.DayEntry

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating day entries: %w", err)
		}

		entry, err := docToDayEntry(doc)
		if err != nil {
			return nil, fmt.Errorf("parsing day entry doc: %w", err)
		}

		// Filter by date range in Go (Firestore only supports inequality on one field)
		dateStr := entry.Date.Format("2006-01-02")
		if dateStr < fromStr || dateStr > toStr {
			continue
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.Before(entries[j].Date)
	})

	return entries, nil
}

// GetDayEntry retrieves a single day entry by ID, verifying ownership.
func (s *FirestoreStore) GetDayEntry(userID string, id uuid.UUID) (*entity.DayEntry, error) {
	doc, err := s.client.Collection(collDayEntries).Doc(id.String()).Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting day entry: %w", err)
	}

	entry, err := docToDayEntry(doc)
	if err != nil {
		return nil, err
	}
	// Verify ownership
	if entry.UserID != userID {
		return nil, nil
	}
	return &entry, nil
}

// CreateDayEntry inserts a new day entry document.
func (s *FirestoreStore) CreateDayEntry(entry *entity.DayEntry) error {
	_, err := s.client.Collection(collDayEntries).Doc(entry.ID.String()).Set(s.ctx, dayEntryToMap(entry))
	if err != nil {
		return fmt.Errorf("creating day entry: %w", err)
	}
	return nil
}

// UpdateDayEntry updates an existing day entry. Returns ErrNotFound if it doesn't exist or doesn't belong to the user.
func (s *FirestoreStore) UpdateDayEntry(userID string, entry *entity.DayEntry) error {
	docRef := s.client.Collection(collDayEntries).Doc(entry.ID.String())

	// Check existence and ownership
	doc, err := docRef.Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("checking day entry existence: %w", err)
	}
	existing, err := docToDayEntry(doc)
	if err != nil {
		return fmt.Errorf("parsing existing day entry: %w", err)
	}
	if existing.UserID != userID {
		return ErrNotFound
	}

	_, err = docRef.Set(s.ctx, dayEntryToMap(entry))
	if err != nil {
		return fmt.Errorf("updating day entry: %w", err)
	}
	return nil
}

// DeleteDayEntry deletes a day entry by ID, verifying ownership.
func (s *FirestoreStore) DeleteDayEntry(userID string, id uuid.UUID) error {
	docRef := s.client.Collection(collDayEntries).Doc(id.String())

	// Check existence and ownership
	doc, err := docRef.Get(s.ctx)
	if status.Code(err) == codes.NotFound {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("checking day entry existence: %w", err)
	}
	existing, err := docToDayEntry(doc)
	if err != nil {
		return fmt.Errorf("parsing existing day entry: %w", err)
	}
	if existing.UserID != userID {
		return ErrNotFound
	}

	_, err = docRef.Delete(s.ctx)
	if err != nil {
		return fmt.Errorf("deleting day entry: %w", err)
	}
	return nil
}

// GetDayEntriesForTrip returns all day entries associated with a trip.
func (s *FirestoreStore) GetDayEntriesForTrip(userID string, tripID uuid.UUID) ([]entity.DayEntry, error) {
	iter := s.client.Collection(collDayEntries).
		Where("userId", "==", userID).
		Where("tripId", "==", tripID.String()).
		Documents(s.ctx)
	defer iter.Stop()

	var entries []entity.DayEntry
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterating day entries for trip: %w", err)
		}

		entry, err := docToDayEntry(doc)
		if err != nil {
			return nil, fmt.Errorf("parsing day entry doc: %w", err)
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.Before(entries[j].Date)
	})

	return entries, nil
}

// DeleteDayEntriesByTrip deletes all day entries associated with a trip.
func (s *FirestoreStore) DeleteDayEntriesByTrip(tripID uuid.UUID) error {
	iter := s.client.Collection(collDayEntries).Where("tripId", "==", tripID.String()).Documents(s.ctx)
	defer iter.Stop()

	batch := s.client.Batch()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("iterating day entries for delete: %w", err)
		}
		batch.Delete(doc.Ref)
	}

	_, err := batch.Commit(s.ctx)
	if err != nil {
		return fmt.Errorf("committing delete day entries batch: %w", err)
	}
	return nil
}

// --- Helper functions for document/entity conversion ---

func tripToMap(trip *entity.Trip) map[string]interface{} {
	m := map[string]interface{}{
		"userId":    trip.UserID,
		"name":      trip.Name,
		"purpose":   trip.Purpose,
		"status":    trip.Status,
		"createdAt": trip.CreatedAt,
		"updatedAt": trip.UpdatedAt,
	}
	if trip.StartDate != nil {
		m["startDate"] = trip.StartDate.Format("2006-01-02")
	} else {
		m["startDate"] = nil
	}
	if trip.EndDate != nil {
		m["endDate"] = trip.EndDate.Format("2006-01-02")
	} else {
		m["endDate"] = nil
	}
	if trip.Notes != nil {
		m["notes"] = *trip.Notes
	} else {
		m["notes"] = nil
	}
	return m
}

func docToTrip(doc *firestore.DocumentSnapshot) (entity.Trip, error) {
	var trip entity.Trip
	data := doc.Data()

	trip.ID, _ = uuid.Parse(doc.Ref.ID)
	trip.UserID, _ = data["userId"].(string)
	trip.Name, _ = data["name"].(string)
	trip.Purpose, _ = data["purpose"].(string)
	trip.Status, _ = data["status"].(string)
	if sd, ok := data["startDate"].(string); ok && sd != "" {
		t, err := time.Parse("2006-01-02", sd)
		if err == nil {
			trip.StartDate = &t
		}
	}
	if ed, ok := data["endDate"].(string); ok && ed != "" {
		t, err := time.Parse("2006-01-02", ed)
		if err == nil {
			trip.EndDate = &t
		}
	}
	if n, ok := data["notes"].(string); ok && n != "" {
		trip.Notes = &n
	}

	trip.CreatedAt = toTime(data["createdAt"])
	trip.UpdatedAt = toTime(data["updatedAt"])

	return trip, nil
}

func itemToMap(item *entity.Item) map[string]interface{} {
	m := map[string]interface{}{
		"tripId":    item.TripID.String(),
		"type":      item.Type,
		"createdAt": item.CreatedAt,
		"updatedAt": item.UpdatedAt,
	}
	if item.Date != nil {
		m["date"] = item.Date.Format("2006-01-02")
	} else {
		m["date"] = nil
	}
	setOptionalString(m, "time", item.Time)
	setOptionalString(m, "confirmation", item.Confirmation)
	setOptionalString(m, "notes", item.Notes)
	setOptionalString(m, "from", item.From)
	setOptionalString(m, "to", item.To)
	setOptionalString(m, "carrier", item.Carrier)
	setOptionalString(m, "flightNumber", item.FlightNumber)
	setOptionalString(m, "name", item.Name)
	setOptionalString(m, "location", item.Location)
	if item.CheckIn != nil {
		m["checkIn"] = item.CheckIn.Format("2006-01-02")
	} else {
		m["checkIn"] = nil
	}
	if item.CheckOut != nil {
		m["checkOut"] = item.CheckOut.Format("2006-01-02")
	} else {
		m["checkOut"] = nil
	}
	return m
}

func docToItem(doc *firestore.DocumentSnapshot) (entity.Item, error) {
	var item entity.Item
	data := doc.Data()

	item.ID, _ = uuid.Parse(doc.Ref.ID)
	if tid, ok := data["tripId"].(string); ok {
		item.TripID, _ = uuid.Parse(tid)
	}
	item.Type, _ = data["type"].(string)

	if d, ok := data["date"].(string); ok && d != "" {
		t, err := time.Parse("2006-01-02", d)
		if err == nil {
			item.Date = &t
		}
	}

	item.Time = getOptionalString(data, "time")
	item.Confirmation = getOptionalString(data, "confirmation")
	item.Notes = getOptionalString(data, "notes")
	item.From = getOptionalString(data, "from")
	item.To = getOptionalString(data, "to")
	item.Carrier = getOptionalString(data, "carrier")
	item.FlightNumber = getOptionalString(data, "flightNumber")
	item.Name = getOptionalString(data, "name")
	item.Location = getOptionalString(data, "location")

	if ci, ok := data["checkIn"].(string); ok && ci != "" {
		t, err := time.Parse("2006-01-02", ci)
		if err == nil {
			item.CheckIn = &t
		}
	}
	if co, ok := data["checkOut"].(string); ok && co != "" {
		t, err := time.Parse("2006-01-02", co)
		if err == nil {
			item.CheckOut = &t
		}
	}

	item.CreatedAt = toTime(data["createdAt"])
	item.UpdatedAt = toTime(data["updatedAt"])

	return item, nil
}

func docToDocument(doc *firestore.DocumentSnapshot) (entity.Document, error) {
	var d entity.Document
	data := doc.Data()

	d.ID, _ = uuid.Parse(doc.Ref.ID)
	d.Name, _ = data["name"].(string)
	d.Type, _ = data["type"].(string)

	if tid, ok := data["tripId"].(string); ok && tid != "" {
		parsed, _ := uuid.Parse(tid)
		d.TripID = &parsed
	}
	d.URL = getOptionalString(data, "url")
	d.CreatedAt = toTime(data["createdAt"])
	d.UpdatedAt = toTime(data["updatedAt"])

	return d, nil
}

func docToTripLocation(doc *firestore.DocumentSnapshot) (entity.TripLocation, error) {
	var loc entity.TripLocation
	data := doc.Data()

	loc.ID, _ = uuid.Parse(doc.Ref.ID)
	if tid, ok := data["tripId"].(string); ok {
		loc.TripID, _ = uuid.Parse(tid)
	}
	if d, ok := data["date"].(string); ok {
		loc.Date, _ = time.Parse("2006-01-02", d)
	}
	loc.Location, _ = data["location"].(string)
	loc.CreatedAt = toTime(data["createdAt"])

	return loc, nil
}

func dayEntryToMap(entry *entity.DayEntry) map[string]interface{} {
	m := map[string]interface{}{
		"userId":    entry.UserID,
		"date":      entry.Date.Format("2006-01-02"),
		"location":  entry.Location,
		"createdAt": entry.CreatedAt,
	}
	if entry.Description != nil {
		m["description"] = *entry.Description
	} else {
		m["description"] = nil
	}
	if entry.TripID != nil {
		m["tripId"] = entry.TripID.String()
	} else {
		m["tripId"] = nil
	}
	return m
}

func docToDayEntry(doc *firestore.DocumentSnapshot) (entity.DayEntry, error) {
	var entry entity.DayEntry
	data := doc.Data()

	entry.ID, _ = uuid.Parse(doc.Ref.ID)
	entry.UserID, _ = data["userId"].(string)
	entry.Location, _ = data["location"].(string)

	if d, ok := data["date"].(string); ok {
		entry.Date, _ = time.Parse("2006-01-02", d)
	}

	if desc, ok := data["description"].(string); ok && desc != "" {
		entry.Description = &desc
	}

	if tid, ok := data["tripId"].(string); ok && tid != "" {
		parsed, _ := uuid.Parse(tid)
		entry.TripID = &parsed
	}

	entry.CreatedAt = toTime(data["createdAt"])

	return entry, nil
}

func docToGoogleCredentials(doc *firestore.DocumentSnapshot, userID string) (entity.GoogleCredentials, error) {
	var creds entity.GoogleCredentials
	data := doc.Data()

	creds.UserID = userID
	creds.AccessToken, _ = data["accessToken"].(string)
	creds.RefreshToken, _ = data["refreshToken"].(string)
	creds.TokenType, _ = data["tokenType"].(string)
	creds.ExpiresAt = toTime(data["expiresAt"])
	creds.Scopes, _ = data["scopes"].(string)
	if e, ok := data["email"].(string); ok && e != "" {
		creds.Email = &e
	}
	creds.CreatedAt = toTime(data["createdAt"])
	creds.UpdatedAt = toTime(data["updatedAt"])

	return creds, nil
}

func userCalendarToMap(cal *entity.UserCalendar) map[string]interface{} {
	return map[string]interface{}{
		"id":         cal.ID.String(),
		"userId":     cal.UserID,
		"calendarId": cal.CalendarID,
		"name":       cal.Name,
		"enabled":    cal.Enabled,
		"createdAt":  cal.CreatedAt,
		"updatedAt":  cal.UpdatedAt,
	}
}

func docToUserCalendar(doc *firestore.DocumentSnapshot) (entity.UserCalendar, error) {
	var cal entity.UserCalendar
	data := doc.Data()

	if idStr, ok := data["id"].(string); ok {
		cal.ID, _ = uuid.Parse(idStr)
	}
	cal.UserID, _ = data["userId"].(string)
	cal.CalendarID, _ = data["calendarId"].(string)
	cal.Name, _ = data["name"].(string)
	cal.Enabled, _ = data["enabled"].(bool)
	cal.CreatedAt = toTime(data["createdAt"])
	cal.UpdatedAt = toTime(data["updatedAt"])

	return cal, nil
}

func calendarLinkToMap(link *entity.CalendarLink) map[string]interface{} {
	m := map[string]interface{}{
		"id":         link.ID.String(),
		"tripId":     link.TripID.String(),
		"calendarId": link.CalendarID,
		"eventId":    link.EventID,
		"syncedAt":   link.SyncedAt,
	}
	if link.ItemID != nil {
		m["itemId"] = link.ItemID.String()
	} else {
		m["itemId"] = nil
	}
	return m
}

func docToCalendarLink(doc *firestore.DocumentSnapshot) (entity.CalendarLink, error) {
	var link entity.CalendarLink
	data := doc.Data()

	if idStr, ok := data["id"].(string); ok {
		link.ID, _ = uuid.Parse(idStr)
	}
	if tid, ok := data["tripId"].(string); ok {
		link.TripID, _ = uuid.Parse(tid)
	}
	if iid, ok := data["itemId"].(string); ok && iid != "" {
		parsed, _ := uuid.Parse(iid)
		link.ItemID = &parsed
	}
	link.CalendarID, _ = data["calendarId"].(string)
	link.EventID, _ = data["eventId"].(string)
	link.SyncedAt = toTime(data["syncedAt"])

	return link, nil
}

func docToProcessedEvent(doc *firestore.DocumentSnapshot) (entity.ProcessedCalendarEvent, error) {
	var event entity.ProcessedCalendarEvent
	data := doc.Data()

	if idStr, ok := data["id"].(string); ok {
		event.ID, _ = uuid.Parse(idStr)
	}
	event.UserID, _ = data["userId"].(string)
	event.CalendarEventID, _ = data["calendarEventId"].(string)
	event.CalendarID, _ = data["calendarId"].(string)
	event.Action, _ = data["action"].(string)
	event.ProcessedAt = toTime(data["processedAt"])

	if tid, ok := data["tripId"].(string); ok && tid != "" {
		parsed, _ := uuid.Parse(tid)
		event.TripID = &parsed
	}
	if iid, ok := data["itemId"].(string); ok && iid != "" {
		parsed, _ := uuid.Parse(iid)
		event.ItemID = &parsed
	}

	return event, nil
}

// --- Generic helper utilities ---

// toTime converts a Firestore value to time.Time.
// Firestore stores timestamps as time.Time values.
func toTime(v interface{}) time.Time {
	if t, ok := v.(time.Time); ok {
		return t
	}
	// Fallback: try parsing as RFC3339 string
	if s, ok := v.(string); ok {
		t, err := time.Parse(time.RFC3339, s)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func setOptionalString(m map[string]interface{}, key string, val *string) {
	if val != nil {
		m[key] = *val
	} else {
		m[key] = nil
	}
}

func getOptionalString(data map[string]interface{}, key string) *string {
	if v, ok := data[key].(string); ok && v != "" {
		return &v
	}
	return nil
}
