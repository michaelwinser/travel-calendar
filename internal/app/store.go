package app

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/michaelwinser/appbase/db"
	"github.com/michaelwinser/appbase/store"
)

// Trip groups related activities into a journey.
type Trip struct {
	ID        string `json:"id"        store:"id,pk"`
	UserID    string `json:"userId"    store:"user_id,index"`
	Name      string `json:"name"      store:"name"`
	Color     string `json:"color"     store:"color"`
	CreatedAt string `json:"createdAt" store:"created_at"`
}

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
	TripID    string `json:"tripId"    store:"trip_id"`
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

// TripStore handles trip persistence.
type TripStore struct {
	coll *store.Collection[Trip]
}

// NewTripStore creates a store backed by the given database.
func NewTripStore(d *db.DB) (*TripStore, error) {
	coll, err := store.NewCollection[Trip](d, "trips")
	if err != nil {
		return nil, err
	}
	return &TripStore{coll: coll}, nil
}

func (s *TripStore) Create(userID, name, color string) (*Trip, error) {
	t := &Trip{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      name,
		Color:     color,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	if err := s.coll.Create(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TripStore) Get(id string) (*Trip, error) { return s.coll.Get(id) }

func (s *TripStore) List(userID string) ([]Trip, error) {
	return s.coll.Where("user_id", "==", userID).OrderBy("created_at", store.Asc).All()
}

func (s *TripStore) Update(t *Trip) error { return s.coll.Update(t.ID, t) }

func (s *TripStore) Delete(id string) error { return s.coll.Delete(id) }

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
func (s *ActivityStore) Create(userID, title, actType, startDate, endDate, location, notes, tripID string) (*Activity, error) {
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
		TripID:    tripID,
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

// ShareLink enables token-based calendar sharing with optional filters.
type ShareLink struct {
	ID         string `json:"id"         store:"id,pk"`
	UserID     string `json:"userId"     store:"user_id,index"`
	OwnerEmail string `json:"ownerEmail" store:"owner_email"`
	Token      string `json:"token"      store:"token,index"`
	Label      string `json:"label"      store:"label"`
	ExpiresAt  string `json:"expiresAt"  store:"expires_at"`
	FromDate   string `json:"fromDate"   store:"from_date"`
	ToDate     string `json:"toDate"     store:"to_date"`
	TripIDs    string `json:"tripIds"    store:"trip_ids"`
	ShowTitle  bool   `json:"showTitle"  store:"show_title"`
	CreatedAt  string `json:"createdAt"  store:"created_at"`
}

// ShareLinkStore handles share link persistence.
type ShareLinkStore struct {
	coll *store.Collection[ShareLink]
}

// NewShareLinkStore creates a store backed by the given database.
func NewShareLinkStore(d *db.DB) (*ShareLinkStore, error) {
	coll, err := store.NewCollection[ShareLink](d, "share_links")
	if err != nil {
		return nil, err
	}
	return &ShareLinkStore{coll: coll}, nil
}

// generateToken creates a cryptographically random URL-safe token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Create adds a new share link with a random token.
func (s *ShareLinkStore) Create(userID, ownerEmail, label, expiresAt, fromDate, toDate, tripIDs string, showTitle bool) (*ShareLink, error) {
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}
	if label == "" {
		label = "Shared calendar"
	}
	link := &ShareLink{
		ID:         uuid.New().String(),
		UserID:     userID,
		OwnerEmail: ownerEmail,
		Token:      token,
		Label:      label,
		ExpiresAt:  expiresAt,
		FromDate:   fromDate,
		ToDate:     toDate,
		TripIDs:    tripIDs,
		ShowTitle:  showTitle,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
	if err := s.coll.Create(link); err != nil {
		return nil, err
	}
	return link, nil
}

// Get retrieves a share link by ID.
func (s *ShareLinkStore) Get(id string) (*ShareLink, error) {
	return s.coll.Get(id)
}

// GetByToken retrieves a share link by its public token.
func (s *ShareLinkStore) GetByToken(token string) (*ShareLink, error) {
	links, err := s.coll.Where("token", "==", token).All()
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, nil
	}
	return &links[0], nil
}

// ListByUser returns all share links for a user.
func (s *ShareLinkStore) ListByUser(userID string) ([]ShareLink, error) {
	return s.coll.Where("user_id", "==", userID).OrderBy("created_at", store.Asc).All()
}

// Delete removes a share link by ID.
func (s *ShareLinkStore) Delete(id string) error {
	return s.coll.Delete(id)
}

// Share represents a user-to-user calendar sharing relationship.
type Share struct {
	ID          string `json:"id"          store:"id,pk"`
	OwnerUserID string `json:"ownerUserId" store:"owner_user_id,index"`
	OwnerEmail  string `json:"ownerEmail"  store:"owner_email"`
	SharedWith  string `json:"sharedWith"  store:"shared_with,index"`
	ShowTitle   bool   `json:"showTitle"   store:"show_title"`
	CreatedAt   string `json:"createdAt"   store:"created_at"`
}

// ShareStore handles user-to-user share persistence.
type ShareStore struct {
	coll *store.Collection[Share]
}

// NewShareStore creates a store backed by the given database.
func NewShareStore(d *db.DB) (*ShareStore, error) {
	coll, err := store.NewCollection[Share](d, "shares")
	if err != nil {
		return nil, err
	}
	return &ShareStore{coll: coll}, nil
}

// Create adds a new share.
func (s *ShareStore) Create(ownerUserID, ownerEmail, sharedWith string, showTitle bool) (*Share, error) {
	share := &Share{
		ID:          uuid.New().String(),
		OwnerUserID: ownerUserID,
		OwnerEmail:  ownerEmail,
		SharedWith:  sharedWith,
		ShowTitle:   showTitle,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	if err := s.coll.Create(share); err != nil {
		return nil, err
	}
	return share, nil
}

// Get retrieves a share by ID.
func (s *ShareStore) Get(id string) (*Share, error) {
	return s.coll.Get(id)
}

// ListByOwner returns all shares created by a user.
func (s *ShareStore) ListByOwner(ownerUserID string) ([]Share, error) {
	return s.coll.Where("owner_user_id", "==", ownerUserID).OrderBy("created_at", store.Asc).All()
}

// ListByRecipient returns all shares where the given email is the recipient.
func (s *ShareStore) ListByRecipient(email string) ([]Share, error) {
	return s.coll.Where("shared_with", "==", email).OrderBy("created_at", store.Asc).All()
}

// FindByOwnerAndRecipient looks up a specific share relationship.
func (s *ShareStore) FindByOwnerAndRecipient(ownerEmail, recipientEmail string) (*Share, error) {
	shares, err := s.coll.Where("owner_email", "==", ownerEmail).Where("shared_with", "==", recipientEmail).All()
	if err != nil {
		return nil, err
	}
	if len(shares) == 0 {
		return nil, nil
	}
	return &shares[0], nil
}

// Delete removes a share by ID.
func (s *ShareStore) Delete(id string) error {
	return s.coll.Delete(id)
}

// PublicProfile controls a user's public dashboard.
type PublicProfile struct {
	ID        string `json:"id"        store:"id,pk"`
	UserID    string `json:"userId"    store:"user_id,index"`
	Handle    string `json:"handle"    store:"handle,index"`
	Enabled   bool   `json:"enabled"   store:"enabled"`
	CreatedAt string `json:"createdAt" store:"created_at"`
}

// PublicProfileStore handles public profile persistence.
type PublicProfileStore struct {
	coll *store.Collection[PublicProfile]
}

// NewPublicProfileStore creates a store backed by the given database.
func NewPublicProfileStore(d *db.DB) (*PublicProfileStore, error) {
	coll, err := store.NewCollection[PublicProfile](d, "public_profiles")
	if err != nil {
		return nil, err
	}
	return &PublicProfileStore{coll: coll}, nil
}

// GetByUserID returns the profile for a user, or nil if none.
func (s *PublicProfileStore) GetByUserID(userID string) (*PublicProfile, error) {
	profiles, err := s.coll.Where("user_id", "==", userID).All()
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}
	return &profiles[0], nil
}

// GetByHandle returns the profile for a handle, or nil if none.
func (s *PublicProfileStore) GetByHandle(handle string) (*PublicProfile, error) {
	profiles, err := s.coll.Where("handle", "==", handle).All()
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}
	return &profiles[0], nil
}

// Create adds a new public profile.
func (s *PublicProfileStore) Create(p *PublicProfile) error {
	return s.coll.Create(p)
}

// Update saves changes to an existing profile.
func (s *PublicProfileStore) Update(p *PublicProfile) error {
	return s.coll.Update(p.ID, p)
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
