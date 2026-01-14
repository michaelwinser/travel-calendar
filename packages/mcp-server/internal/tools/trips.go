package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/user/travel-calendar/mcp-server/internal/formatter"
)

func (r *Registry) registerTripTools() {
	// get_trips - List trips with optional filters
	r.register(ToolDefinition{
		Name:        "get_trips",
		Description: "Get a list of trips. Can filter by upcoming, past, or purpose.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"upcoming": map[string]interface{}{
					"type":        "boolean",
					"description": "Filter to show only upcoming trips",
				},
				"past": map[string]interface{}{
					"type":        "boolean",
					"description": "Filter to show only past trips",
				},
				"purpose": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"business", "vacation", "conference", "family", "other"},
					"description": "Filter by trip purpose",
				},
			},
		},
	}, r.handleGetTrips)

	// get_trip - Get a single trip by ID
	r.register(ToolDefinition{
		Name:        "get_trip",
		Description: "Get detailed information about a specific trip including all its items.",
		InputSchema: map[string]interface{}{
			"type":     "object",
			"required": []string{"id"},
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"format":      "uuid",
					"description": "The trip ID",
				},
			},
		},
	}, r.handleGetTrip)

	// create_trip - Create a new trip
	r.register(ToolDefinition{
		Name:        "create_trip",
		Description: "Create a new trip.",
		InputSchema: map[string]interface{}{
			"type":     "object",
			"required": []string{"name", "purpose"},
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Trip name",
				},
				"purpose": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"business", "vacation", "conference", "family", "other"},
					"description": "Trip purpose",
				},
				"startDate": map[string]interface{}{
					"type":        "string",
					"format":      "date",
					"description": "Start date (YYYY-MM-DD)",
				},
				"endDate": map[string]interface{}{
					"type":        "string",
					"format":      "date",
					"description": "End date (YYYY-MM-DD)",
				},
				"notes": map[string]interface{}{
					"type":        "string",
					"description": "Additional notes",
				},
			},
		},
	}, r.handleCreateTrip)

	// update_trip - Update an existing trip
	r.register(ToolDefinition{
		Name:        "update_trip",
		Description: "Update an existing trip.",
		InputSchema: map[string]interface{}{
			"type":     "object",
			"required": []string{"id"},
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"format":      "uuid",
					"description": "The trip ID",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "New trip name",
				},
				"purpose": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"business", "vacation", "conference", "family", "other"},
					"description": "New trip purpose",
				},
				"startDate": map[string]interface{}{
					"type":        "string",
					"format":      "date",
					"description": "New start date (YYYY-MM-DD)",
				},
				"endDate": map[string]interface{}{
					"type":        "string",
					"format":      "date",
					"description": "New end date (YYYY-MM-DD)",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"planning", "confirmed", "in_progress", "completed", "cancelled"},
					"description": "Trip status",
				},
				"notes": map[string]interface{}{
					"type":        "string",
					"description": "Additional notes",
				},
			},
		},
	}, r.handleUpdateTrip)

	// delete_trip - Delete a trip
	r.register(ToolDefinition{
		Name:        "delete_trip",
		Description: "Delete a trip and all its items.",
		InputSchema: map[string]interface{}{
			"type":     "object",
			"required": []string{"id"},
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"format":      "uuid",
					"description": "The trip ID",
				},
			},
		},
	}, r.handleDeleteTrip)

	// search_trips - Search trips
	r.register(ToolDefinition{
		Name:        "search_trips",
		Description: "Search trips by name or notes.",
		InputSchema: map[string]interface{}{
			"type":     "object",
			"required": []string{"query"},
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
			},
		},
	}, r.handleSearchTrips)

	// add_item - Add an item to a trip
	r.register(ToolDefinition{
		Name:        "add_item",
		Description: "Add an item (flight, hotel, train, drive, or event) to a trip.",
		InputSchema: map[string]interface{}{
			"type":     "object",
			"required": []string{"tripId", "type"},
			"properties": map[string]interface{}{
				"tripId": map[string]interface{}{
					"type":        "string",
					"format":      "uuid",
					"description": "The trip ID",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"flight", "hotel", "train", "drive", "event"},
					"description": "Item type",
				},
				"date": map[string]interface{}{
					"type":        "string",
					"format":      "date",
					"description": "Date (YYYY-MM-DD)",
				},
				"time": map[string]interface{}{
					"type":        "string",
					"pattern":     "^([01]\\d|2[0-3]):([0-5]\\d)$",
					"description": "Time (HH:MM)",
				},
				"from": map[string]interface{}{
					"type":        "string",
					"description": "Origin (for transport items)",
				},
				"to": map[string]interface{}{
					"type":        "string",
					"description": "Destination (for transport items)",
				},
				"carrier": map[string]interface{}{
					"type":        "string",
					"description": "Carrier/airline (for transport items)",
				},
				"flightNumber": map[string]interface{}{
					"type":        "string",
					"description": "Flight number (for flights)",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name (for hotels/events)",
				},
				"location": map[string]interface{}{
					"type":        "string",
					"description": "Location (for hotels/events)",
				},
				"checkIn": map[string]interface{}{
					"type":        "string",
					"format":      "date",
					"description": "Check-in date (for hotels)",
				},
				"checkOut": map[string]interface{}{
					"type":        "string",
					"format":      "date",
					"description": "Check-out date (for hotels)",
				},
				"confirmation": map[string]interface{}{
					"type":        "string",
					"description": "Confirmation number",
				},
				"notes": map[string]interface{}{
					"type":        "string",
					"description": "Additional notes",
				},
			},
		},
	}, r.handleAddItem)

	// delete_item - Delete an item
	r.register(ToolDefinition{
		Name:        "delete_item",
		Description: "Delete an item from a trip.",
		InputSchema: map[string]interface{}{
			"type":     "object",
			"required": []string{"id"},
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"format":      "uuid",
					"description": "The item ID",
				},
			},
		},
	}, r.handleDeleteItem)
}

func (r *Registry) handleGetTrips(args map[string]interface{}) (*ToolResult, error) {
	u, _ := url.Parse(r.backendURL + "/api/trips")
	q := u.Query()
	if upcoming, ok := args["upcoming"].(bool); ok && upcoming {
		q.Set("upcoming", "true")
	}
	if past, ok := args["past"].(bool); ok && past {
		q.Set("past", "true")
	}
	if purpose, ok := args["purpose"].(string); ok && purpose != "" {
		q.Set("purpose", purpose)
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to fetch trips: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errorResult(fmt.Sprintf("Backend error: %s", string(body))), nil
	}

	var trips []map[string]interface{}
	if err := json.Unmarshal(body, &trips); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse trips: %v", err)), nil
	}

	return textResult(formatter.FormatTrips(trips)), nil
}

func (r *Registry) handleGetTrip(args map[string]interface{}) (*ToolResult, error) {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return errorResult("Missing required argument: id"), nil
	}

	resp, err := http.Get(r.backendURL + "/api/trips/" + id)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to fetch trip: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return errorResult("Trip not found"), nil
	}
	if resp.StatusCode != http.StatusOK {
		return errorResult(fmt.Sprintf("Backend error: %s", string(body))), nil
	}

	var trip map[string]interface{}
	if err := json.Unmarshal(body, &trip); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse trip: %v", err)), nil
	}

	return textResult(formatter.FormatTrip(trip)), nil
}

func (r *Registry) handleCreateTrip(args map[string]interface{}) (*ToolResult, error) {
	body, _ := json.Marshal(args)
	resp, err := http.Post(r.backendURL+"/api/trips", "application/json", bytes.NewReader(body))
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to create trip: %v", err)), nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return errorResult(fmt.Sprintf("Backend error: %s", string(respBody))), nil
	}

	var trip map[string]interface{}
	if err := json.Unmarshal(respBody, &trip); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse trip: %v", err)), nil
	}

	return textResult(fmt.Sprintf("Trip created successfully!\n\n%s", formatter.FormatTrip(trip))), nil
}

func (r *Registry) handleUpdateTrip(args map[string]interface{}) (*ToolResult, error) {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return errorResult("Missing required argument: id"), nil
	}

	// Remove id from args for the request body
	delete(args, "id")
	body, _ := json.Marshal(args)

	req, _ := http.NewRequest(http.MethodPatch, r.backendURL+"/api/trips/"+id, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to update trip: %v", err)), nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return errorResult("Trip not found"), nil
	}
	if resp.StatusCode != http.StatusOK {
		return errorResult(fmt.Sprintf("Backend error: %s", string(respBody))), nil
	}

	var trip map[string]interface{}
	if err := json.Unmarshal(respBody, &trip); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse trip: %v", err)), nil
	}

	return textResult(fmt.Sprintf("Trip updated successfully!\n\n%s", formatter.FormatTrip(trip))), nil
}

func (r *Registry) handleDeleteTrip(args map[string]interface{}) (*ToolResult, error) {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return errorResult("Missing required argument: id"), nil
	}

	req, _ := http.NewRequest(http.MethodDelete, r.backendURL+"/api/trips/"+id, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to delete trip: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errorResult("Trip not found"), nil
	}
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return errorResult(fmt.Sprintf("Backend error: %s", string(body))), nil
	}

	return textResult("Trip deleted successfully."), nil
}

func (r *Registry) handleSearchTrips(args map[string]interface{}) (*ToolResult, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return errorResult("Missing required argument: query"), nil
	}

	u, _ := url.Parse(r.backendURL + "/api/trips/search")
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to search trips: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errorResult(fmt.Sprintf("Backend error: %s", string(body))), nil
	}

	var trips []map[string]interface{}
	if err := json.Unmarshal(body, &trips); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse trips: %v", err)), nil
	}

	if len(trips) == 0 {
		return textResult(fmt.Sprintf("No trips found matching '%s'.", query)), nil
	}

	return textResult(fmt.Sprintf("Found %d trip(s) matching '%s':\n\n%s", len(trips), query, formatter.FormatTrips(trips))), nil
}

func (r *Registry) handleAddItem(args map[string]interface{}) (*ToolResult, error) {
	tripID, ok := args["tripId"].(string)
	if !ok || tripID == "" {
		return errorResult("Missing required argument: tripId"), nil
	}

	// Remove tripId from args for the request body
	delete(args, "tripId")
	body, _ := json.Marshal(args)

	resp, err := http.Post(r.backendURL+"/api/trips/"+tripID+"/items", "application/json", bytes.NewReader(body))
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to add item: %v", err)), nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return errorResult("Trip not found"), nil
	}
	if resp.StatusCode != http.StatusCreated {
		return errorResult(fmt.Sprintf("Backend error: %s", string(respBody))), nil
	}

	var item map[string]interface{}
	if err := json.Unmarshal(respBody, &item); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse item: %v", err)), nil
	}

	return textResult(fmt.Sprintf("Item added successfully!\n\n%s", formatter.FormatItem(item))), nil
}

func (r *Registry) handleDeleteItem(args map[string]interface{}) (*ToolResult, error) {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return errorResult("Missing required argument: id"), nil
	}

	req, _ := http.NewRequest(http.MethodDelete, r.backendURL+"/api/items/"+id, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to delete item: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errorResult("Item not found"), nil
	}
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return errorResult(fmt.Sprintf("Backend error: %s", string(body))), nil
	}

	return textResult("Item deleted successfully."), nil
}

func textResult(text string) *ToolResult {
	return &ToolResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func errorResult(message string) *ToolResult {
	return &ToolResult{
		Content: []ContentBlock{{Type: "text", Text: message}},
		IsError: true,
	}
}

// Silence unused import
var _ = strings.TrimSpace
