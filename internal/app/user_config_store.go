package app

import (
	"github.com/michaelwinser/appbase/db"
	"github.com/michaelwinser/appbase/store"
)

// UserConfig stores per-user key-value configuration.
type UserConfig struct {
	ID     string `json:"id"     store:"id,pk"`
	UserID string `json:"userId" store:"user_id,index"`
	Key    string `json:"key"    store:"key"`
	Value  string `json:"value"  store:"value"`
}

// UserConfigStore handles user configuration persistence.
type UserConfigStore struct {
	coll *store.Collection[UserConfig]
}

func NewUserConfigStore(d *db.DB) (*UserConfigStore, error) {
	coll, err := store.NewCollection[UserConfig](d, "user_configs")
	if err != nil {
		return nil, err
	}
	return &UserConfigStore{coll: coll}, nil
}

// Get returns the value for a user's config key, or empty string if not set.
func (s *UserConfigStore) Get(userID, key string) string {
	configs, err := s.coll.Where("user_id", "==", userID).Where("key", "==", key).All()
	if err != nil || len(configs) == 0 {
		return ""
	}
	return configs[0].Value
}

// Set creates or updates a config value for a user.
func (s *UserConfigStore) Set(userID, key, value string) {
	configs, err := s.coll.Where("user_id", "==", userID).Where("key", "==", key).All()
	if err == nil && len(configs) > 0 {
		configs[0].Value = value
		s.coll.Update(configs[0].ID, &configs[0])
		return
	}
	s.coll.Create(&UserConfig{
		ID:     userID + ":" + key,
		UserID: userID,
		Key:    key,
		Value:  value,
	})
}
