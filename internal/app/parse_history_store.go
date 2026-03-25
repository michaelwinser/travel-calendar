package app

import (
	"time"

	"github.com/google/uuid"
	"github.com/michaelwinser/appbase/db"
	"github.com/michaelwinser/appbase/store"
)

// ParseHistory records a parse attempt for analytics and refinement.
type ParseHistory struct {
	ID         string `json:"id"         store:"id,pk"`
	UserID     string `json:"userId"     store:"user_id,index"`
	InputText  string `json:"inputText"  store:"input_text"`
	Today      string `json:"today"      store:"today"`
	ResultJSON string `json:"resultJson" store:"result_json"`
	Accepted   bool   `json:"accepted"   store:"accepted"`
	ActivityID string `json:"activityId" store:"activity_id"`
	CreatedAt  string `json:"createdAt"  store:"created_at"`
}

// ParseHistoryStore handles parse history persistence.
type ParseHistoryStore struct {
	coll *store.Collection[ParseHistory]
}

// NewParseHistoryStore creates a store backed by the given database.
func NewParseHistoryStore(d *db.DB) (*ParseHistoryStore, error) {
	coll, err := store.NewCollection[ParseHistory](d, "parse_history")
	if err != nil {
		return nil, err
	}
	return &ParseHistoryStore{coll: coll}, nil
}

// Create records a parse attempt.
func (s *ParseHistoryStore) Create(userID, inputText, today, resultJSON string) (*ParseHistory, error) {
	h := &ParseHistory{
		ID:         uuid.New().String(),
		UserID:     userID,
		InputText:  inputText,
		Today:      today,
		ResultJSON: resultJSON,
		Accepted:   false,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
	if err := s.coll.Create(h); err != nil {
		return nil, err
	}
	return h, nil
}

// Get retrieves a parse history entry by ID.
func (s *ParseHistoryStore) Get(id string) (*ParseHistory, error) {
	return s.coll.Get(id)
}

// MarkAccepted links a parse history entry to the created activity.
func (s *ParseHistoryStore) MarkAccepted(id, activityID string) error {
	h, err := s.coll.Get(id)
	if err != nil || h == nil {
		return err
	}
	h.Accepted = true
	h.ActivityID = activityID
	return s.coll.Update(id, h)
}
