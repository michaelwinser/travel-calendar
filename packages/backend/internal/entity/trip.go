// Package entity defines the internal domain models with database tags.
package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Trip represents a travel trip.
type Trip struct {
	ID        uuid.UUID  `firestore:"-"`
	Name      string     `firestore:"name"`
	Purpose   string     `firestore:"purpose"`
	StartDate *time.Time `firestore:"startDate"`
	EndDate   *time.Time `firestore:"endDate"`
	Status    string     `firestore:"status"`
	Notes     *string    `firestore:"notes"`
	CreatedAt time.Time  `firestore:"createdAt"`
	UpdatedAt time.Time  `firestore:"updatedAt"`
}

// ToAPI converts a Trip entity to an API Trip response.
func (t *Trip) ToAPI() api.Trip {
	trip := api.Trip{
		Id:        openapi_types.UUID(t.ID),
		Name:      t.Name,
		Purpose:   api.TripPurpose(t.Purpose),
		Status:    api.TripStatus(t.Status),
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
	if t.StartDate != nil {
		date := openapi_types.Date{Time: *t.StartDate}
		trip.StartDate = &date
	}
	if t.EndDate != nil {
		date := openapi_types.Date{Time: *t.EndDate}
		trip.EndDate = &date
	}
	if t.Notes != nil {
		trip.Notes = t.Notes
	}
	return trip
}

// ToAPIWithItems converts a Trip entity to an API Trip response including items.
func (t *Trip) ToAPIWithItems(items []Item) api.Trip {
	trip := t.ToAPI()
	apiItems := make([]api.Item, len(items))
	for i, item := range items {
		apiItems[i] = item.ToAPI()
	}
	trip.Items = &apiItems
	return trip
}

// TripFromCreateRequest creates a Trip entity from an API CreateTripRequest.
func TripFromCreateRequest(req *api.CreateTripRequest) Trip {
	trip := Trip{
		ID:        uuid.New(),
		Name:      req.Name,
		Purpose:   string(req.Purpose),
		Status:    "planning", // default status
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if req.StartDate != nil {
		t := req.StartDate.Time
		trip.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.Time
		trip.EndDate = &t
	}
	if req.Status != nil {
		trip.Status = string(*req.Status)
	}
	if req.Notes != nil {
		trip.Notes = req.Notes
	}
	return trip
}

// ApplyUpdate applies an UpdateTripRequest to an existing Trip.
func (t *Trip) ApplyUpdate(req *api.UpdateTripRequest) {
	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Purpose != nil {
		t.Purpose = string(*req.Purpose)
	}
	if req.StartDate != nil {
		tm := req.StartDate.Time
		t.StartDate = &tm
	}
	if req.EndDate != nil {
		tm := req.EndDate.Time
		t.EndDate = &tm
	}
	if req.Status != nil {
		t.Status = string(*req.Status)
	}
	if req.Notes != nil {
		t.Notes = req.Notes
	}
	t.UpdatedAt = time.Now()
}
