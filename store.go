package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/michaelwinser/appbase/db"
	"github.com/michaelwinser/appbase/store"
)

// Activity is the primary planning entity — a span of time with a purpose and location.
type Activity struct {
	ID        string `json:"id"        store:"id,pk"`
	UserID    string `json:"userId"    store:"user_id,index"`
	Title     string `json:"title"     store:"title"`
	Type      string `json:"type"      store:"type"`
	StartDate string `json:"startDate" store:"start_date,index"`
	EndDate   string `json:"endDate"   store:"end_date"`
	Location  string `json:"location"  store:"location"`
	Notes     string `json:"notes"     store:"notes"`
	TripName  string `json:"tripName"  store:"trip_name"`
	Source    string `json:"source"    store:"source"`
	CreatedAt string `json:"createdAt" store:"created_at"`
}

// Activity types.
const (
	TypeTravel     = "travel"
	TypeStay       = "stay"
	TypeConference = "conference"
	TypeVacation   = "vacation"
	TypeCommitment = "commitment"
)

// ValidTypes lists all accepted activity types.
var ValidTypes = []string{TypeTravel, TypeStay, TypeConference, TypeVacation, TypeCommitment}

// ActivityStore handles activity persistence.
type ActivityStore struct {
	coll *store.Collection[Activity]
}

// NewActivityStore creates a store backed by the given database.
func NewActivityStore(d *db.DB) (*ActivityStore, error) {
	coll, err := store.NewCollection[Activity](d, "activities")
	if err != nil {
		return nil, err
	}
	return &ActivityStore{coll: coll}, nil
}

// Create adds a new activity.
func (s *ActivityStore) Create(userID, title, actType, startDate, endDate, location, notes, tripName string) (*Activity, error) {
	if err := validateType(actType); err != nil {
		return nil, err
	}
	if err := validateDateRange(startDate, endDate); err != nil {
		return nil, err
	}
	a := &Activity{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     title,
		Type:      actType,
		StartDate: startDate,
		EndDate:   endDate,
		Location:  location,
		Notes:     notes,
		TripName:  tripName,
		Source:    "manual",
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	if err := s.coll.Create(a); err != nil {
		return nil, err
	}
	return a, nil
}

// Get retrieves an activity by ID.
func (s *ActivityStore) Get(id string) (*Activity, error) {
	return s.coll.Get(id)
}

// List returns all activities for a user, ordered by start date.
func (s *ActivityStore) List(userID string) ([]Activity, error) {
	return s.coll.Where("user_id", "==", userID).OrderBy("start_date", store.Asc).All()
}

// ListRange returns activities overlapping a date range (inclusive).
func (s *ActivityStore) ListRange(userID, from, to string) ([]Activity, error) {
	// Activities that overlap: start_date <= to AND end_date >= from
	return s.coll.
		Where("user_id", "==", userID).
		Where("start_date", "<=", to).
		Where("end_date", ">=", from).
		OrderBy("start_date", store.Asc).
		All()
}

// ForDate returns all activities spanning a specific date.
func (s *ActivityStore) ForDate(userID, date string) ([]Activity, error) {
	return s.ListRange(userID, date, date)
}

// Delete removes an activity by ID.
func (s *ActivityStore) Delete(id string) error {
	return s.coll.Delete(id)
}

// Update replaces an existing activity.
func (s *ActivityStore) Update(a *Activity) error {
	if err := validateType(a.Type); err != nil {
		return err
	}
	if err := validateDateRange(a.StartDate, a.EndDate); err != nil {
		return err
	}
	return s.coll.Update(a.ID, a)
}

func validateType(t string) error {
	for _, v := range ValidTypes {
		if t == v {
			return nil
		}
	}
	return fmt.Errorf("invalid activity type %q (valid: %v)", t, ValidTypes)
}

func validateDateRange(start, end string) error {
	s, err := time.Parse("2006-01-02", start)
	if err != nil {
		return fmt.Errorf("invalid start date %q (expected YYYY-MM-DD)", start)
	}
	e, err := time.Parse("2006-01-02", end)
	if err != nil {
		return fmt.Errorf("invalid end date %q (expected YYYY-MM-DD)", end)
	}
	if e.Before(s) {
		return fmt.Errorf("end date %s is before start date %s", end, start)
	}
	return nil
}
