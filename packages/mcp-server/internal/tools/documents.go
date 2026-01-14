package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/user/travel-calendar/mcp-server/internal/formatter"
)

func (r *Registry) registerDocumentTools() {
	// get_documents - List documents
	r.register(ToolDefinition{
		Name:        "get_documents",
		Description: "Get a list of documents. Can filter by trip or unassociated documents.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tripId": map[string]interface{}{
					"type":        "string",
					"format":      "uuid",
					"description": "Filter by trip ID",
				},
				"unassociated": map[string]interface{}{
					"type":        "boolean",
					"description": "Show only documents not associated with any trip",
				},
			},
		},
	}, r.handleGetDocuments)
}

func (r *Registry) handleGetDocuments(args map[string]interface{}) (*ToolResult, error) {
	u, _ := url.Parse(r.backendURL + "/api/documents")
	q := u.Query()
	if tripID, ok := args["tripId"].(string); ok && tripID != "" {
		q.Set("tripId", tripID)
	}
	if unassociated, ok := args["unassociated"].(bool); ok && unassociated {
		q.Set("unassociated", "true")
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to fetch documents: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errorResult(fmt.Sprintf("Backend error: %s", string(body))), nil
	}

	var documents []map[string]interface{}
	if err := json.Unmarshal(body, &documents); err != nil {
		return errorResult(fmt.Sprintf("Failed to parse documents: %v", err)), nil
	}

	return textResult(formatter.FormatDocuments(documents)), nil
}
