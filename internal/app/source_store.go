package app

import (
	"time"

	"github.com/google/uuid"
	"github.com/michaelwinser/appbase/db"
	"github.com/michaelwinser/appbase/store"
)

// ImportSource represents a persistent connection to an external calendar.
type ImportSource struct {
	ID           string `json:"id"           store:"id,pk"`
	UserID       string `json:"userId"       store:"user_id,index"`
	Name         string `json:"name"         store:"name"`
	URL          string `json:"url"          store:"url"`
	SourceType   string `json:"sourceType"   store:"source_type"` // ical, json
	FilterConfig string `json:"filterConfig" store:"filter_config"`
	LastSyncAt   string `json:"lastSyncAt"   store:"last_sync_at"`
	Status       string `json:"status"       store:"status"` // active, paused
	CreatedAt    string `json:"createdAt"    store:"created_at"`
}

// StagedEvent represents a candidate activity from an import source.
type StagedEvent struct {
	ID            string `json:"id"            store:"id,pk"`
	UserID        string `json:"userId"        store:"user_id,index"`
	SourceID      string `json:"sourceId"      store:"source_id,index"`
	SourceEventID string `json:"sourceEventId" store:"source_event_id"`
	Title         string `json:"title"         store:"title"`
	Type          string `json:"type"          store:"type"`
	StartDate     string `json:"startDate"     store:"start_date"`
	EndDate       string `json:"endDate"       store:"end_date"`
	Location      string `json:"location"      store:"location"`
	Notes         string `json:"notes"         store:"notes"`
	State         string `json:"state"         store:"state"` // new, imported, hidden
	ActivityID    string `json:"activityId"    store:"activity_id"`
	CreatedAt     string `json:"createdAt"     store:"created_at"`
	UpdatedAt     string `json:"updatedAt"     store:"updated_at"`
}

// ImportSourceStore handles import source persistence.
type ImportSourceStore struct {
	coll *store.Collection[ImportSource]
}

func NewImportSourceStore(d *db.DB) (*ImportSourceStore, error) {
	coll, err := store.NewCollection[ImportSource](d, "import_sources")
	if err != nil {
		return nil, err
	}
	return &ImportSourceStore{coll: coll}, nil
}

func (s *ImportSourceStore) Create(userID, name, url, sourceType, filterConfig string) (*ImportSource, error) {
	src := &ImportSource{
		ID:           uuid.New().String(),
		UserID:       userID,
		Name:         name,
		URL:          url,
		SourceType:   sourceType,
		FilterConfig: filterConfig,
		Status:       "active",
		CreatedAt:    time.Now().Format(time.RFC3339),
	}
	if err := s.coll.Create(src); err != nil {
		return nil, err
	}
	return src, nil
}

func (s *ImportSourceStore) Get(id string) (*ImportSource, error) {
	return s.coll.Get(id)
}

func (s *ImportSourceStore) List(userID string) ([]ImportSource, error) {
	return s.coll.Where("user_id", "==", userID).OrderBy("created_at", store.Asc).All()
}

func (s *ImportSourceStore) Update(src *ImportSource) error {
	return s.coll.Update(src.ID, src)
}

func (s *ImportSourceStore) Delete(id string) error {
	return s.coll.Delete(id)
}

func (s *ImportSourceStore) FindByURL(userID, url string) (*ImportSource, error) {
	sources, err := s.coll.Where("user_id", "==", userID).All()
	if err != nil {
		return nil, err
	}
	for _, src := range sources {
		if src.URL == url {
			return &src, nil
		}
	}
	return nil, nil
}

// StagedEventStore handles staged event persistence.
type StagedEventStore struct {
	coll *store.Collection[StagedEvent]
}

func NewStagedEventStore(d *db.DB) (*StagedEventStore, error) {
	coll, err := store.NewCollection[StagedEvent](d, "staged_events")
	if err != nil {
		return nil, err
	}
	return &StagedEventStore{coll: coll}, nil
}

func (s *StagedEventStore) Create(e *StagedEvent) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	if e.CreatedAt == "" {
		e.CreatedAt = time.Now().Format(time.RFC3339)
	}
	e.UpdatedAt = e.CreatedAt
	return s.coll.Create(e)
}

func (s *StagedEventStore) Get(id string) (*StagedEvent, error) {
	return s.coll.Get(id)
}

func (s *StagedEventStore) ListBySource(sourceID string) ([]StagedEvent, error) {
	return s.coll.Where("source_id", "==", sourceID).OrderBy("start_date", store.Asc).All()
}

func (s *StagedEventStore) ListByUser(userID, stateFilter string) ([]StagedEvent, error) {
	q := s.coll.Where("user_id", "==", userID)
	if stateFilter != "" {
		q = q.Where("state", "==", stateFilter)
	}
	return q.OrderBy("start_date", store.Asc).All()
}

func (s *StagedEventStore) FindBySourceEventID(sourceID, sourceEventID string) (*StagedEvent, error) {
	events, err := s.coll.Where("source_id", "==", sourceID).Where("source_event_id", "==", sourceEventID).All()
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, nil
	}
	return &events[0], nil
}

func (s *StagedEventStore) Update(e *StagedEvent) error {
	e.UpdatedAt = time.Now().Format(time.RFC3339)
	return s.coll.Update(e.ID, e)
}

func (s *StagedEventStore) Delete(id string) error {
	return s.coll.Delete(id)
}

func (s *StagedEventStore) DeleteBySource(sourceID string) error {
	events, err := s.ListBySource(sourceID)
	if err != nil {
		return err
	}
	for _, e := range events {
		s.coll.Delete(e.ID)
	}
	return nil
}
