package mcp

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/user/travel-calendar/backend/internal/api"
)

func (h *Handler) handleGetTrips(args map[string]interface{}) (interface{}, error) {
	var upcoming, past *bool
	var purpose *api.TripPurpose

	if v, ok := args["upcoming"].(bool); ok {
		upcoming = &v
	}
	if v, ok := args["past"].(bool); ok {
		past = &v
	}
	if v, ok := args["purpose"].(string); ok && v != "" {
		p := api.TripPurpose(v)
		purpose = &p
	}

	trips, err := h.svc.ListTrips(upcoming, past, purpose)
	if err != nil {
		return errorResult(fmt.Errorf("failed to list trips: %w", err)), nil
	}

	return textResult(FormatTrips(trips)), nil
}

func (h *Handler) handleGetTrip(args map[string]interface{}) (interface{}, error) {
	idStr, ok := args["trip_id"].(string)
	if !ok {
		// Also try "id" for backwards compatibility
		idStr, ok = args["id"].(string)
	}
	if !ok || idStr == "" {
		return errorResult(fmt.Errorf("missing required argument: trip_id")), nil
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return errorResult(fmt.Errorf("invalid trip_id: %v", err)), nil
	}

	trip, err := h.svc.GetTrip(id)
	if err != nil {
		return errorResult(fmt.Errorf("failed to get trip: %w", err)), nil
	}
	if trip == nil {
		return errorResult(fmt.Errorf("trip not found")), nil
	}

	return textResult(FormatTrip(*trip)), nil
}

func (h *Handler) handleCreateTrip(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return errorResult(fmt.Errorf("missing required argument: name")), nil
	}

	purposeStr, ok := args["purpose"].(string)
	if !ok || purposeStr == "" {
		return errorResult(fmt.Errorf("missing required argument: purpose")), nil
	}
	purpose := api.TripPurpose(purposeStr)

	req := &api.CreateTripRequest{
		Name:    name,
		Purpose: purpose,
	}

	if v, ok := args["start_date"].(string); ok && v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			d := openapi_types.Date{Time: t}
			req.StartDate = &d
		}
	}
	if v, ok := args["end_date"].(string); ok && v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			d := openapi_types.Date{Time: t}
			req.EndDate = &d
		}
	}
	if v, ok := args["notes"].(string); ok {
		req.Notes = &v
	}
	if v, ok := args["location"].(string); ok {
		req.Location = &v
	}
	if v, ok := args["status"].(string); ok {
		s := api.TripStatus(v)
		req.Status = &s
	}

	trip, err := h.svc.CreateTrip(req)
	if err != nil {
		return errorResult(fmt.Errorf("failed to create trip: %w", err)), nil
	}

	return textResult(fmt.Sprintf("Trip created successfully!\n\n%s", FormatTrip(*trip))), nil
}

func (h *Handler) handleUpdateTrip(args map[string]interface{}) (interface{}, error) {
	idStr, ok := args["trip_id"].(string)
	if !ok {
		idStr, ok = args["id"].(string)
	}
	if !ok || idStr == "" {
		return errorResult(fmt.Errorf("missing required argument: trip_id")), nil
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return errorResult(fmt.Errorf("invalid trip_id: %v", err)), nil
	}

	req := &api.UpdateTripRequest{}

	if v, ok := args["name"].(string); ok && v != "" {
		req.Name = &v
	}
	if v, ok := args["purpose"].(string); ok && v != "" {
		p := api.TripPurpose(v)
		req.Purpose = &p
	}
	if v, ok := args["start_date"].(string); ok && v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			d := openapi_types.Date{Time: t}
			req.StartDate = &d
		}
	}
	if v, ok := args["end_date"].(string); ok && v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			d := openapi_types.Date{Time: t}
			req.EndDate = &d
		}
	}
	if v, ok := args["status"].(string); ok && v != "" {
		s := api.TripStatus(v)
		req.Status = &s
	}
	if v, ok := args["notes"].(string); ok {
		req.Notes = &v
	}

	trip, err := h.svc.UpdateTrip(id, req)
	if err != nil {
		return errorResult(fmt.Errorf("failed to update trip: %w", err)), nil
	}
	if trip == nil {
		return errorResult(fmt.Errorf("trip not found")), nil
	}

	return textResult(fmt.Sprintf("Trip updated successfully!\n\n%s", FormatTrip(*trip))), nil
}

func (h *Handler) handleDeleteTrip(args map[string]interface{}) (interface{}, error) {
	idStr, ok := args["trip_id"].(string)
	if !ok {
		idStr, ok = args["id"].(string)
	}
	if !ok || idStr == "" {
		return errorResult(fmt.Errorf("missing required argument: trip_id")), nil
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return errorResult(fmt.Errorf("invalid trip_id: %v", err)), nil
	}

	if err := h.svc.DeleteTrip(id); err != nil {
		return errorResult(fmt.Errorf("failed to delete trip: %w", err)), nil
	}

	return textResult("Trip deleted successfully."), nil
}

func (h *Handler) handleSearchTrips(args map[string]interface{}) (interface{}, error) {
	query, ok := args["q"].(string)
	if !ok {
		query, ok = args["query"].(string)
	}
	if !ok || query == "" {
		return errorResult(fmt.Errorf("missing required argument: q")), nil
	}

	trips, err := h.svc.SearchTrips(query)
	if err != nil {
		return errorResult(fmt.Errorf("failed to search trips: %w", err)), nil
	}

	if len(trips) == 0 {
		return textResult(fmt.Sprintf("No trips found matching '%s'.", query)), nil
	}

	return textResult(fmt.Sprintf("Found %d trip(s) matching '%s':\n\n%s", len(trips), query, FormatTrips(trips))), nil
}

func (h *Handler) handleAddItem(args map[string]interface{}) (interface{}, error) {
	tripIDStr, ok := args["trip_id"].(string)
	if !ok {
		tripIDStr, ok = args["tripId"].(string)
	}
	if !ok || tripIDStr == "" {
		return errorResult(fmt.Errorf("missing required argument: trip_id")), nil
	}

	tripID, err := uuid.Parse(tripIDStr)
	if err != nil {
		return errorResult(fmt.Errorf("invalid trip_id: %v", err)), nil
	}

	itemTypeStr, ok := args["type"].(string)
	if !ok || itemTypeStr == "" {
		return errorResult(fmt.Errorf("missing required argument: type")), nil
	}
	itemType := api.ItemType(itemTypeStr)

	req := &api.CreateItemRequest{
		Type: itemType,
	}

	if v, ok := args["date"].(string); ok && v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			d := openapi_types.Date{Time: t}
			req.Date = &d
		}
	}
	if v, ok := args["time"].(string); ok {
		req.Time = &v
	}
	if v, ok := args["from"].(string); ok {
		req.From = &v
	}
	if v, ok := args["to"].(string); ok {
		req.To = &v
	}
	if v, ok := args["carrier"].(string); ok {
		req.Carrier = &v
	}
	if v, ok := args["flight_number"].(string); ok {
		req.FlightNumber = &v
	}
	if v, ok := args["flightNumber"].(string); ok {
		req.FlightNumber = &v
	}
	if v, ok := args["name"].(string); ok {
		req.Name = &v
	}
	if v, ok := args["location"].(string); ok {
		req.Location = &v
	}
	if v, ok := args["check_in"].(string); ok && v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			d := openapi_types.Date{Time: t}
			req.CheckIn = &d
		}
	}
	if v, ok := args["checkIn"].(string); ok && v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			d := openapi_types.Date{Time: t}
			req.CheckIn = &d
		}
	}
	if v, ok := args["check_out"].(string); ok && v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			d := openapi_types.Date{Time: t}
			req.CheckOut = &d
		}
	}
	if v, ok := args["checkOut"].(string); ok && v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			d := openapi_types.Date{Time: t}
			req.CheckOut = &d
		}
	}
	if v, ok := args["confirmation"].(string); ok {
		req.Confirmation = &v
	}
	if v, ok := args["notes"].(string); ok {
		req.Notes = &v
	}

	item, err := h.svc.CreateTripItem(tripID, req)
	if err != nil {
		return errorResult(fmt.Errorf("failed to add item: %w", err)), nil
	}

	return textResult(fmt.Sprintf("Item added successfully!\n\n%s", FormatItem(*item))), nil
}

func (h *Handler) handleDeleteItem(args map[string]interface{}) (interface{}, error) {
	idStr, ok := args["item_id"].(string)
	if !ok {
		idStr, ok = args["id"].(string)
	}
	if !ok || idStr == "" {
		return errorResult(fmt.Errorf("missing required argument: item_id")), nil
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return errorResult(fmt.Errorf("invalid item_id: %v", err)), nil
	}

	if err := h.svc.DeleteItem(id); err != nil {
		return errorResult(fmt.Errorf("failed to delete item: %w", err)), nil
	}

	return textResult("Item deleted successfully."), nil
}
