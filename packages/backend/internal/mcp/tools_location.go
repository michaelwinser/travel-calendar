package mcp

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/user/travel-calendar/backend/internal/api"
)

func (h *Handler) handleGetBaseLocations(args map[string]interface{}) (interface{}, error) {
	locs, err := h.svc.GetBaseLocations()
	if err != nil {
		return errorResult(fmt.Errorf("failed to get base locations: %w", err)), nil
	}

	return textResult(FormatBaseLocations(*locs)), nil
}

func (h *Handler) handleSetBaseLocations(args map[string]interface{}) (interface{}, error) {
	req := &api.SetBaseLocationsRequest{}

	if v, ok := args["home"].(string); ok && v != "" {
		req.Home = &v
	}
	if v, ok := args["work"].(string); ok && v != "" {
		req.Work = &v
	}

	locs, err := h.svc.SetBaseLocations(req)
	if err != nil {
		return errorResult(fmt.Errorf("failed to set base locations: %w", err)), nil
	}

	return textResult(fmt.Sprintf("Base locations updated successfully!\n\n%s", FormatBaseLocations(*locs))), nil
}

func (h *Handler) handleGetTripLocations(args map[string]interface{}) (interface{}, error) {
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

	locs, err := h.svc.GetTripLocations("", tripID)
	if err != nil {
		return errorResult(fmt.Errorf("failed to get trip locations: %w", err)), nil
	}

	return textResult(FormatTripLocations(locs)), nil
}

func (h *Handler) handleSetTripLocations(args map[string]interface{}) (interface{}, error) {
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

	req := &api.SetTripLocationsRequest{}

	if v, ok := args["default_location"].(string); ok && v != "" {
		req.DefaultLocation = &v
	}
	if v, ok := args["defaultLocation"].(string); ok && v != "" {
		req.DefaultLocation = &v
	}

	// Parse locations array
	if locsRaw, ok := args["locations"].([]interface{}); ok {
		locs := make([]api.TripDayLocation, 0, len(locsRaw))
		for _, locRaw := range locsRaw {
			if locMap, ok := locRaw.(map[string]interface{}); ok {
				dateStr, _ := locMap["date"].(string)
				if dateStr == "" {
					continue
				}
				dateTime, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					continue
				}

				var locations []string
				if locsArr, ok := locMap["locations"].([]interface{}); ok {
					for _, l := range locsArr {
						if s, ok := l.(string); ok {
							locations = append(locations, s)
						}
					}
				}

				if len(locations) > 0 {
					locs = append(locs, api.TripDayLocation{
						Date:      openapi_types.Date{Time: dateTime},
						Locations: locations,
					})
				}
			}
		}
		if len(locs) > 0 {
			req.Locations = &locs
		}
	}

	result, err := h.svc.SetTripLocations("", tripID, req)
	if err != nil {
		return errorResult(fmt.Errorf("failed to set trip locations: %w", err)), nil
	}

	return textResult(fmt.Sprintf("Trip locations updated successfully!\n\n%s", FormatTripLocations(result))), nil
}

func (h *Handler) handleGetLocationOnDate(args map[string]interface{}) (interface{}, error) {
	dateStr, ok := args["date"].(string)
	if !ok || dateStr == "" {
		return errorResult(fmt.Errorf("missing required argument: date")), nil
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return errorResult(fmt.Errorf("invalid date format (use YYYY-MM-DD): %v", err)), nil
	}

	loc, err := h.svc.GetLocationOnDate("", date)
	if err != nil {
		return errorResult(fmt.Errorf("failed to get location: %w", err)), nil
	}

	return textResult(FormatLocationOnDate(*loc)), nil
}

func (h *Handler) handleGetLocationRange(args map[string]interface{}) (interface{}, error) {
	fromStr, ok := args["from"].(string)
	if !ok || fromStr == "" {
		return errorResult(fmt.Errorf("missing required argument: from")), nil
	}
	toStr, ok := args["to"].(string)
	if !ok || toStr == "" {
		return errorResult(fmt.Errorf("missing required argument: to")), nil
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return errorResult(fmt.Errorf("invalid from date (use YYYY-MM-DD): %v", err)), nil
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return errorResult(fmt.Errorf("invalid to date (use YYYY-MM-DD): %v", err)), nil
	}

	segments, err := h.svc.GetLocationRange("", from, to)
	if err != nil {
		return errorResult(fmt.Errorf("failed to get location range: %w", err)), nil
	}

	return textResult(FormatLocationRange(segments)), nil
}
