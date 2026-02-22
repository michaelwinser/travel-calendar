package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Document represents a travel document in the database.
type Document struct {
	ID        uuid.UUID  `db:"id" firestore:"-"`
	TripID    *uuid.UUID `db:"trip_id" firestore:"tripId"`
	Name      string     `db:"name" firestore:"name"`
	Type      string     `db:"type" firestore:"type"`
	URL       *string    `db:"url" firestore:"url"`
	CreatedAt time.Time  `db:"created_at" firestore:"createdAt"`
	UpdatedAt time.Time  `db:"updated_at" firestore:"updatedAt"`
}

// ToAPI converts a Document entity to an API Document response.
func (d *Document) ToAPI() api.Document {
	doc := api.Document{
		Id:        openapi_types.UUID(d.ID),
		Name:      d.Name,
		Type:      api.DocumentType(d.Type),
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
	if d.TripID != nil {
		tid := openapi_types.UUID(*d.TripID)
		doc.TripId = &tid
	}
	if d.URL != nil {
		doc.Url = d.URL
	}
	return doc
}
