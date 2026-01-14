package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/user/travel-calendar/mcp-server/internal/formatter"
)

func (r *Registry) registerLocationTools() {
	// get_location_on_date - Query location for a specific date
	r.register(ToolDefinition{
		Name: "get_location_on_date",
		Description: `Find out where the user will be on a specific date.

Use this to answer questions like:
- "Where will I be on January 30th?"
- "Am I traveling on March 15?"
- "What's my location next Tuesday?"

Returns the location(s) and whether it's from a trip or home.`,
		InputSchema: map[string]interface{}{
			"type":     "object",
			"required": []string{"date"},
			"properties": map[string]interface{}{
				"date": map[string]interface{}{
					"type":        "string",
					"format":      "date",
					"description": "Date to query (YYYY-MM-DD)",
				},
			},
		},
	}, r.handleGetLocationOnDate)

	// get_location_range - Query locations for a date range
	r.register(ToolDefinition{
		Name: "get_location_range",
		Description: `Get a timeline of where the user will be across a date range.

Use this to answer questions like:
- "Where will I be in January?"
- "What's my travel schedule for Q1?"
- "Am I home between the 10th and 20th?"

Returns a timeline of location segments with dates and sources.`,
		InputSchema: map[string]interface{}{
			"type":     "object",
			"required": []string{"from", "to"},
			"properties": map[string]interface{}{
				"from": map[string]interface{}{
					"type":        "string",
					"format":      "date",
					"description": "Start date (YYYY-MM-DD)",
				},
				"to": map[string]interface{}{
					"type":        "string",
					"format":      "date",
					"description": "End date (YYYY-MM-DD)",
				},
			},
		},
	}, r.handleGetLocationRange)

	// set_trip_locations - Set locations for a trip
	r.register(ToolDefinition{
		Name: "set_trip_locations",
		Description: `Set the location(s) for a trip.

Use this when the user says things like:
- "I'll be in Brussels for FOSDEM"
- "Set my location to London for my Europe trip"
- "On Jan 30 I'll travel from London to Brussels"

Can set a default location for all days, or specify locations per day.`,
		InputSchema: map[string]interface{}{
			"type":     "object",
			"required": []string{"tripId"},
			"properties": map[string]interface{}{
				"tripId": map[string]interface{}{
					"type":        "string",
					"format":      "uuid",
					"description": "The trip ID",
				},
				"defaultLocation": map[string]interface{}{
					"type":        "string",
					"description": "Default location for all days not explicitly specified",
				},
				"locations": map[string]interface{}{
					"type":        "array",
					"description": "Per-date location overrides",
					"items": map[string]interface{}{
						"type":     "object",
						"required": []string{"date", "locations"},
						"properties": map[string]interface{}{
							"date": map[string]interface{}{
								"type":        "string",
								"format":      "date",
								"description": "The date (YYYY-MM-DD)",
							},
							"locations": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "string",
								},
								"description": "Location(s) for this date",
							},
						},
					},
				},
			},
		},
	}, r.handleSetTripLocations)

	// get_base_locations - Get configured home/work locations
	r.register(ToolDefinition{
		Name: "get_base_locations",
		Description: `Get the user's configured base locations (home and work).

Use this to understand the user's default locations when not traveling.`,
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}, r.handleGetBaseLocations)

	// set_base_locations - Set home/work locations
	r.register(ToolDefinition{
		Name: "set_base_locations",
		Description: `Set the user's base locations (home and/or work).

Use this when the user says things like:
- "My home is in New York"
- "I work in San Francisco"
- "Set my home location to NYC"`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"home": map[string]interface{}{
					"type":        "string",
					"description": "Home location",
				},
				"work": map[string]interface{}{
					"type":        "string",
					"description": "Work location",
				},
			},
		},
	}, r.handleSetBaseLocations)
}

func (r *Registry) handleGetLocationOnDate(args map[string]interface{}) (*ToolResult, error) {
	date, ok := args["date"].(string)
	if !ok || date == "" {
		return errorResult("Missing required argument: date"), nil
	}

	resp, err := http.Get(r.backendURL + "/api/location/on/" + date)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to fetch location: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errorResult(fmt.Sprintf("Backend error: %s", string(body))), nil
	}

	var location map[string]interface{}
	if err := json.Unmarshal(body, &location); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse location: %v", err)), nil
	}

	return textResult(formatter.FormatLocationOnDate(location)), nil
}

func (r *Registry) handleGetLocationRange(args map[string]interface{}) (*ToolResult, error) {
	from, ok := args["from"].(string)
	if !ok || from == "" {
		return errorResult("Missing required argument: from"), nil
	}
	to, ok := args["to"].(string)
	if !ok || to == "" {
		return errorResult("Missing required argument: to"), nil
	}

	u, _ := url.Parse(r.backendURL + "/api/location/range")
	q := u.Query()
	q.Set("from", from)
	q.Set("to", to)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to fetch locations: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusBadRequest {
		return errorResult("Invalid date range (end must be after start)"), nil
	}
	if resp.StatusCode != http.StatusOK {
		return errorResult(fmt.Sprintf("Backend error: %s", string(body))), nil
	}

	var segments []map[string]interface{}
	if err := json.Unmarshal(body, &segments); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse locations: %v", err)), nil
	}

	return textResult(formatter.FormatLocationRange(segments)), nil
}

func (r *Registry) handleSetTripLocations(args map[string]interface{}) (*ToolResult, error) {
	tripID, ok := args["tripId"].(string)
	if !ok || tripID == "" {
		return errorResult("Missing required argument: tripId"), nil
	}

	// Remove tripId from args for the request body
	delete(args, "tripId")
	body, _ := json.Marshal(args)

	req, _ := http.NewRequest(http.MethodPut, r.backendURL+"/api/trips/"+tripID+"/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to set trip locations: %v", err)), nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return errorResult("Trip not found"), nil
	}
	if resp.StatusCode != http.StatusOK {
		return errorResult(fmt.Sprintf("Backend error: %s", string(respBody))), nil
	}

	var locations []map[string]interface{}
	if err := json.Unmarshal(respBody, &locations); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse locations: %v", err)), nil
	}

	return textResult(fmt.Sprintf("Trip locations updated successfully!\n\n%s", formatter.FormatTripLocations(locations))), nil
}

func (r *Registry) handleGetBaseLocations(args map[string]interface{}) (*ToolResult, error) {
	resp, err := http.Get(r.backendURL + "/api/config/locations")
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to fetch base locations: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errorResult(fmt.Sprintf("Backend error: %s", string(body))), nil
	}

	var locations map[string]interface{}
	if err := json.Unmarshal(body, &locations); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse locations: %v", err)), nil
	}

	return textResult(formatter.FormatBaseLocations(locations)), nil
}

func (r *Registry) handleSetBaseLocations(args map[string]interface{}) (*ToolResult, error) {
	body, _ := json.Marshal(args)

	req, _ := http.NewRequest(http.MethodPut, r.backendURL+"/api/config/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to set base locations: %v", err)), nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errorResult(fmt.Sprintf("Backend error: %s", string(respBody))), nil
	}

	var locations map[string]interface{}
	if err := json.Unmarshal(respBody, &locations); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse locations: %v", err)), nil
	}

	return textResult(fmt.Sprintf("Base locations updated successfully!\n\n%s", formatter.FormatBaseLocations(locations))), nil
}
