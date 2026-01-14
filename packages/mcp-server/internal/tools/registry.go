// Package tools provides MCP tool definitions and implementations.
package tools

import (
	"fmt"
)

// ToolDefinition describes an MCP tool.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolResult represents the result of a tool call.
type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a content block in a tool result.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Registry holds all registered tools.
type Registry struct {
	tools      map[string]ToolDefinition
	handlers   map[string]func(map[string]interface{}) (*ToolResult, error)
	backendURL string
}

// NewRegistry creates a new tool registry.
func NewRegistry(backendURL string) *Registry {
	r := &Registry{
		tools:      make(map[string]ToolDefinition),
		handlers:   make(map[string]func(map[string]interface{}) (*ToolResult, error)),
		backendURL: backendURL,
	}
	r.registerTools()
	return r
}

// ListTools returns all registered tool definitions.
func (r *Registry) ListTools() []ToolDefinition {
	tools := make([]ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// CallTool executes a tool by name.
func (r *Registry) CallTool(name string, args map[string]interface{}) (*ToolResult, error) {
	handler, ok := r.handlers[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return handler(args)
}

func (r *Registry) register(def ToolDefinition, handler func(map[string]interface{}) (*ToolResult, error)) {
	r.tools[def.Name] = def
	r.handlers[def.Name] = handler
}

func (r *Registry) registerTools() {
	// Register trip tools
	r.registerTripTools()
	// Register document tools
	r.registerDocumentTools()
	// Register location tools
	r.registerLocationTools()
}
