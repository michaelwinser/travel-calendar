package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Item represents a trip item (flight, hotel, etc.) in the database.
type Item struct {
	ID           uuid.UUID  `db:"id" firestore:"-"`
	TripID       uuid.UUID  `db:"trip_id" firestore:"tripId"`
	Type         string     `db:"type" firestore:"type"`
	Date         *time.Time `db:"date" firestore:"date"`
	Time         *string    `db:"time" firestore:"time"`
	Confirmation *string    `db:"confirmation" firestore:"confirmation"`
	Notes        *string    `db:"notes" firestore:"notes"`
	// Transport fields
	From         *string `db:"from_location" firestore:"from"`
	To           *string `db:"to_location" firestore:"to"`
	Carrier      *string `db:"carrier" firestore:"carrier"`
	FlightNumber *string `db:"flight_number" firestore:"flightNumber"`
	// Hotel/Event fields
	Name     *string    `db:"name" firestore:"name"`
	Location *string    `db:"location" firestore:"location"`
	CheckIn  *time.Time `db:"check_in" firestore:"checkIn"`
	CheckOut *time.Time `db:"check_out" firestore:"checkOut"`
	// Timestamps
	CreatedAt time.Time `db:"created_at" firestore:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" firestore:"updatedAt"`
}

// ToAPI converts an Item entity to an API Item response.
func (i *Item) ToAPI() api.Item {
	item := api.Item{
		Id:        openapi_types.UUID(i.ID),
		TripId:    openapi_types.UUID(i.TripID),
		Type:      api.ItemType(i.Type),
		CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt,
	}
	if i.Date != nil {
		date := openapi_types.Date{Time: *i.Date}
		item.Date = &date
	}
	if i.Time != nil {
		item.Time = i.Time
	}
	if i.Confirmation != nil {
		item.Confirmation = i.Confirmation
	}
	if i.Notes != nil {
		item.Notes = i.Notes
	}
	if i.From != nil {
		item.From = i.From
	}
	if i.To != nil {
		item.To = i.To
	}
	if i.Carrier != nil {
		item.Carrier = i.Carrier
	}
	if i.FlightNumber != nil {
		item.FlightNumber = i.FlightNumber
	}
	if i.Name != nil {
		item.Name = i.Name
	}
	if i.Location != nil {
		item.Location = i.Location
	}
	if i.CheckIn != nil {
		date := openapi_types.Date{Time: *i.CheckIn}
		item.CheckIn = &date
	}
	if i.CheckOut != nil {
		date := openapi_types.Date{Time: *i.CheckOut}
		item.CheckOut = &date
	}
	return item
}

// ItemFromCreateRequest creates an Item entity from an API CreateItemRequest.
func ItemFromCreateRequest(tripID uuid.UUID, req *api.CreateItemRequest) Item {
	item := Item{
		ID:           uuid.New(),
		TripID:       tripID,
		Type:         string(req.Type),
		Time:         req.Time,
		Confirmation: req.Confirmation,
		Notes:        req.Notes,
		From:         req.From,
		To:           req.To,
		Carrier:      req.Carrier,
		FlightNumber: req.FlightNumber,
		Name:         req.Name,
		Location:     req.Location,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if req.Date != nil {
		t := req.Date.Time
		item.Date = &t
	}
	if req.CheckIn != nil {
		t := req.CheckIn.Time
		item.CheckIn = &t
	}
	if req.CheckOut != nil {
		t := req.CheckOut.Time
		item.CheckOut = &t
	}
	return item
}
