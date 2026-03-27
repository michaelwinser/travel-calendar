package app

import (
	"time"

	"github.com/google/uuid"
	"github.com/michaelwinser/appbase/db"
	"github.com/michaelwinser/appbase/store"
)

// SyncTarget represents a configured export destination (e.g., a Google Calendar).
type SyncTarget struct {
	ID           string `json:"id"           store:"id,pk"`
	UserID       string `json:"userId"       store:"user_id,index"`
	Type         string `json:"type"         store:"type"`          // "google_calendar"
	CalendarID   string `json:"calendarId"   store:"calendar_id"`   // Google Calendar ID
	CalendarName string `json:"calendarName" store:"calendar_name"` // Display name
	Status       string `json:"status"       store:"status"`        // active, paused
	LastSyncAt   string `json:"lastSyncAt"   store:"last_sync_at"`
	CreatedAt    string `json:"createdAt"    store:"created_at"`
}

// SyncRecord tracks a synced entity (activity or trip) on a target.
type SyncRecord struct {
	ID              string `json:"id"              store:"id,pk"`
	UserID          string `json:"userId"          store:"user_id,index"`
	TargetID        string `json:"targetId"        store:"target_id,index"`
	EntityType      string `json:"entityType"      store:"entity_type"` // "activity" or "trip"
	EntityID        string `json:"entityId"        store:"entity_id"`
	EntityKey       string `json:"entityKey"       store:"entity_key"`
	CalendarEventID string `json:"calendarEventId" store:"calendar_event_id"`
	SyncHash        string `json:"syncHash"        store:"sync_hash"` // hash of synced fields
	LastSyncAt      string `json:"lastSyncAt"      store:"last_sync_at"`
}

// SyncTargetStore handles sync target persistence.
type SyncTargetStore struct {
	coll *store.Collection[SyncTarget]
}

func NewSyncTargetStore(d *db.DB) (*SyncTargetStore, error) {
	coll, err := store.NewCollection[SyncTarget](d, "sync_targets")
	if err != nil {
		return nil, err
	}
	return &SyncTargetStore{coll: coll}, nil
}

func (s *SyncTargetStore) Create(target *SyncTarget) error {
	return s.coll.Create(target)
}

func (s *SyncTargetStore) Get(id string) (*SyncTarget, error) {
	return s.coll.Get(id)
}

func (s *SyncTargetStore) ListByUser(userID string) ([]SyncTarget, error) {
	return s.coll.Where("user_id", "==", userID).All()
}

func (s *SyncTargetStore) Update(target *SyncTarget) error {
	return s.coll.Update(target.ID, target)
}

func (s *SyncTargetStore) Delete(id string) error {
	return s.coll.Delete(id)
}

// SyncRecordStore handles sync record persistence.
type SyncRecordStore struct {
	coll *store.Collection[SyncRecord]
}

func NewSyncRecordStore(d *db.DB) (*SyncRecordStore, error) {
	coll, err := store.NewCollection[SyncRecord](d, "sync_records")
	if err != nil {
		return nil, err
	}
	return &SyncRecordStore{coll: coll}, nil
}

func (s *SyncRecordStore) Create(record *SyncRecord) error {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	return s.coll.Create(record)
}

func (s *SyncRecordStore) ListByTarget(targetID string) ([]SyncRecord, error) {
	return s.coll.Where("target_id", "==", targetID).All()
}

func (s *SyncRecordStore) FindByEntity(targetID, entityType, entityID string) (*SyncRecord, error) {
	records, err := s.coll.Where("target_id", "==", targetID).Where("entity_id", "==", entityID).All()
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		if r.EntityType == entityType {
			return &r, nil
		}
	}
	return nil, nil
}

func (s *SyncRecordStore) Update(record *SyncRecord) error {
	return s.coll.Update(record.ID, record)
}

func (s *SyncRecordStore) Delete(id string) error {
	return s.coll.Delete(id)
}

func (s *SyncRecordStore) DeleteByTarget(targetID string) error {
	records, err := s.ListByTarget(targetID)
	if err != nil {
		return err
	}
	for _, r := range records {
		s.coll.Delete(r.ID)
	}
	return nil
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}
