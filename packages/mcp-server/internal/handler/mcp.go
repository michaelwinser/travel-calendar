// Package handler provides the MCP JSON-RPC handler.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/user/travel-calendar/mcp-server/internal/tools"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP handler implements the Model Context Protocol.
type MCPHandler struct {
	registry *tools.Registry
}

// NewMCPHandler creates a new MCP handler.
func NewMCPHandler(backendURL string) *MCPHandler {
	return &MCPHandler{
		registry: tools.NewRegistry(backendURL),
	}
}

// ServeHTTP handles MCP JSON-RPC requests.
func (h *MCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

func (h *MCPHandler) handleInitialize(w http.ResponseWriter, req JSONRPCRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]string{
			"name":    "travel-calendar-mcp",
			"version": "1.0.0",
		},
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"instructions": "Travel Calendar MCP Server. Use these tools to manage your trips, items, and documents.",
	}
	respondRPCResult(w, req.ID, result)
}

func (h *MCPHandler) handleToolsList(w http.ResponseWriter, req JSONRPCRequest) {
	result := map[string]interface{}{
		"tools": h.registry.ListTools(),
	}
	respondRPCResult(w, req.ID, result)
}

func (h *MCPHandler) handleToolsCall(w http.ResponseWriter, req JSONRPCRequest) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		respondRPCError(w, req.ID, -32602, "Invalid params", nil)
		return
	}

	result, err := h.registry.CallTool(params.Name, params.Arguments)
	if err != nil {
		respondRPCError(w, req.ID, -32000, err.Error(), nil)
		return
	}
	respondRPCResult(w, req.ID, result)
}

func (h *MCPHandler) handleResourcesList(w http.ResponseWriter, req JSONRPCRequest) {
	// No resources currently
	result := map[string]interface{}{
		"resources": []interface{}{},
	}
	respondRPCResult(w, req.ID, result)
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
