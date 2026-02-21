package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Document represents a travel document.
type Document struct {
	ID        uuid.UUID  `firestore:"-"`
	TripID    *uuid.UUID `firestore:"tripId"`
	Name      string     `firestore:"name"`
	Type      string     `firestore:"type"`
	URL       *string    `firestore:"url"`
	CreatedAt time.Time  `firestore:"createdAt"`
	UpdatedAt time.Time  `firestore:"updatedAt"`
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
