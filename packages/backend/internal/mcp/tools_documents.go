package mcp

import (
	"fmt"

	"github.com/google/uuid"
)

func (h *Handler) handleGetDocuments(args map[string]interface{}) (interface{}, error) {
	var tripID *uuid.UUID
	var unassociated *bool

	if v, ok := args["trip_id"].(string); ok && v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			tripID = &id
		}
	}
	if v, ok := args["tripId"].(string); ok && v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			tripID = &id
		}
	}
	if v, ok := args["unassociated"].(bool); ok {
		unassociated = &v
	}

	docs, err := h.svc.ListDocuments(tripID, unassociated)
	if err != nil {
		return errorResult(fmt.Errorf("failed to list documents: %w", err)), nil
	}

	return textResult(FormatDocuments(docs)), nil
}
