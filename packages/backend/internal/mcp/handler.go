// Package mcp provides the MCP JSON-RPC handler.
package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/user/travel-calendar/backend/internal/service"
)

const protocolVersion = "2024-11-05"

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Handler implements the Model Context Protocol.
type Handler struct {
	svc *service.Service
}

// NewHandler creates a new MCP handler with the given service.
func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// ServeHTTP handles MCP JSON-RPC requests (POST /mcp).
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondRPCError(w, nil, -32700, "Parse error", nil)
		return
	}

	if req.JSONRPC != "2.0" {
		respondRPCError(w, req.ID, -32600, "Invalid Request: jsonrpc must be '2.0'", nil)
		return
	}

	switch req.Method {
	case "initialize":
		h.handleInitialize(w, req)
	case "tools/list":
		h.handleToolsList(w, req)
	case "tools/call":
		h.handleToolsCall(w, req)
	case "resources/list":
		h.handleResourcesList(w, req)
	default:
		respondRPCError(w, req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}
}

// VersionHandler handles HEAD/GET requests for protocol version discovery.
func (h *Handler) VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("MCP-Protocol-Version", protocolVersion)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleInitialize(w http.ResponseWriter, req JSONRPCRequest) {
	info := GetServerInfo()
	result := map[string]interface{}{
		"protocolVersion": protocolVersion,
		"serverInfo": map[string]string{
			"name":    info.Name,
			"version": info.Version,
		},
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"instructions": info.Instructions,
	}
	respondRPCResult(w, req.ID, result)
}

func (h *Handler) handleToolsList(w http.ResponseWriter, req JSONRPCRequest) {
	tools := GetTools()
	toolList := make([]map[string]interface{}, len(tools))
	for i, t := range tools {
		toolList[i] = map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		}
	}
	result := map[string]interface{}{
		"tools": toolList,
	}
	respondRPCResult(w, req.ID, result)
}

func (h *Handler) handleToolsCall(w http.ResponseWriter, req JSONRPCRequest) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		respondRPCError(w, req.ID, -32602, "Invalid params", nil)
		return
	}

	result, err := h.callTool(params.Name, params.Arguments)
	if err != nil {
		respondRPCError(w, req.ID, -32000, err.Error(), nil)
		return
	}
	respondRPCResult(w, req.ID, result)
}

func (h *Handler) handleResourcesList(w http.ResponseWriter, req JSONRPCRequest) {
	resources := GetResources()
	resourceList := make([]map[string]interface{}, len(resources))
	for i, r := range resources {
		resourceList[i] = map[string]interface{}{
			"uri":         r.URI,
			"name":        r.Name,
			"description": r.Description,
			"mimeType":    r.MimeType,
		}
	}
	result := map[string]interface{}{
		"resources": resourceList,
	}
	respondRPCResult(w, req.ID, result)
}

// callTool dispatches to the appropriate tool handler.
func (h *Handler) callTool(name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	// Trip tools
	case "get_trips":
		return h.handleGetTrips(args)
	case "get_trip":
		return h.handleGetTrip(args)
	case "create_trip":
		return h.handleCreateTrip(args)
	case "update_trip":
		return h.handleUpdateTrip(args)
	case "delete_trip":
		return h.handleDeleteTrip(args)
	case "search_trips":
		return h.handleSearchTrips(args)

	// Item tools
	case "add_item":
		return h.handleAddItem(args)
	case "delete_item":
		return h.handleDeleteItem(args)
	case "move_item":
		return h.handleMoveItem(args)

	// Trip organization tools
	case "merge_trips":
		return h.handleMergeTrips(args)

	// Document tools
	case "get_documents":
		return h.handleGetDocuments(args)

	// Location tools
	case "get_base_locations":
		return h.handleGetBaseLocations(args)
	case "set_base_locations":
		return h.handleSetBaseLocations(args)
	case "get_trip_locations":
		return h.handleGetTripLocations(args)
	case "set_trip_locations":
		return h.handleSetTripLocations(args)
	case "get_location_on_date":
		return h.handleGetLocationOnDate(args)
	case "get_location_range":
		return h.handleGetLocationRange(args)

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// textResult creates a standard MCP tool result with text content.
func textResult(text string) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
	}
}

// errorResult creates an MCP tool result indicating an error.
func errorResult(err error) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": fmt.Sprintf("Error: %s", err.Error())},
		},
		"isError": true,
	}
}

func respondRPCResult(w http.ResponseWriter, id interface{}, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func respondRPCError(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	})
}
