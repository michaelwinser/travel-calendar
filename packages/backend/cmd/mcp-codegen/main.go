// Command mcp-codegen generates MCP tool definitions from OpenAPI spec
//
// Usage:
//
//	go run ./cmd/mcp-codegen ../../api/openapi.yaml
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// MCPConfig represents the x-mcp extension at the info level
type MCPConfig struct {
	Name         string        `json:"name"`
	Version      string        `json:"version"`
	Instructions string        `json:"instructions"`
	Resources    []MCPResource `json:"resources"`
}

// MCPResource represents an MCP resource definition
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

// MCPTool represents an MCP tool definition
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPOperationExt represents the x-mcp extension on an operation
type MCPOperationExt struct {
	Tool              string `json:"tool,omitempty"`
	Description       string `json:"description,omitempty"`
	PathParamsAsInput bool   `json:"path_params_as_input,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <openapi-spec.yaml>\n", os.Args[0])
		os.Exit(1)
	}

	specPath := os.Args[1]
	outputPath := "internal/mcp/tools.gen.go"

	if len(os.Args) > 2 {
		outputPath = os.Args[2]
	}

	// Load OpenAPI spec
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	// Extract x-mcp config from info
	var mcpConfig MCPConfig
	if ext, ok := doc.Info.Extensions["x-mcp"]; ok {
		data, err := json.Marshal(ext)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling x-mcp config: %v\n", err)
			os.Exit(1)
		}
		if err := json.Unmarshal(data, &mcpConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error unmarshaling x-mcp config: %v\n", err)
			os.Exit(1)
		}
	}

	// Extract tools from operations
	tools := extractOperationTools(doc)

	// Sort tools by name for deterministic output
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	// Generate code
	code := generateCode(mcpConfig, tools)

	// Write output
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d tools and %d resources\n", outputPath, len(tools), len(mcpConfig.Resources))
}

func extractOperationTools(doc *openapi3.T) []MCPTool {
	var tools []MCPTool

	for _, pathItem := range doc.Paths.Map() {
		for _, op := range pathItem.Operations() {
			if op == nil {
				continue
			}

			ext, ok := op.Extensions["x-mcp"]
			if !ok {
				continue
			}

			var mcpExt MCPOperationExt
			data, err := json.Marshal(ext)
			if err != nil {
				continue
			}
			if err := json.Unmarshal(data, &mcpExt); err != nil {
				continue
			}

			if mcpExt.Tool != "" {
				tool := buildTool(mcpExt.Tool, mcpExt.Description, op, mcpExt.PathParamsAsInput)
				tools = append(tools, tool)
			}
		}
	}

	return tools
}

func buildTool(name, description string, op *openapi3.Operation, pathParamsAsInput bool) MCPTool {
	inputSchema := buildInputSchema(op, pathParamsAsInput)

	return MCPTool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
}

func buildInputSchema(op *openapi3.Operation, pathParamsAsInput bool) map[string]interface{} {
	properties := make(map[string]interface{})
	var required []string

	// Add parameters from operation
	for _, paramRef := range op.Parameters {
		if paramRef == nil || paramRef.Value == nil {
			continue
		}
		param := paramRef.Value

		// Skip path params unless explicitly included
		if param.In == "path" && !pathParamsAsInput {
			continue
		}

		propSchema := schemaToMap(param.Schema)
		if param.Description != "" {
			propSchema["description"] = param.Description
		}

		// Convert param name to snake_case
		propName := toSnakeCase(param.Name)

		// For path params, rename generic 'id' params
		if param.In == "path" && pathParamsAsInput {
			if propName == "trip_id" {
				propName = "trip_id"
			} else if propName == "item_id" {
				propName = "item_id"
			}
		}

		properties[propName] = propSchema

		if param.Required {
			required = append(required, propName)
		}
	}

	// Add request body properties
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		if content, ok := op.RequestBody.Value.Content["application/json"]; ok && content.Schema != nil {
			bodySchema := content.Schema.Value
			if bodySchema != nil && bodySchema.Type != nil && bodySchema.Type.Is("object") {
				for propName, propRef := range bodySchema.Properties {
					if propRef == nil || propRef.Value == nil {
						continue
					}
					propSchema := schemaToMap(propRef)
					snakeName := toSnakeCase(propName)
					properties[snakeName] = propSchema
				}
				for _, req := range bodySchema.Required {
					snakeName := toSnakeCase(req)
					if !contains(required, snakeName) {
						required = append(required, snakeName)
					}
				}
			}
		}
	}

	result := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		sort.Strings(required)
		result["required"] = required
	}

	return result
}

func schemaToMap(schemaRef *openapi3.SchemaRef) map[string]interface{} {
	if schemaRef == nil || schemaRef.Value == nil {
		return map[string]interface{}{"type": "string"}
	}

	schema := schemaRef.Value
	result := make(map[string]interface{})

	// Handle type
	if schema.Type != nil {
		types := schema.Type.Slice()
		if len(types) == 1 {
			result["type"] = types[0]
		} else if len(types) > 1 {
			result["type"] = types
		}
	}

	// Handle description
	if schema.Description != "" {
		result["description"] = schema.Description
	}

	// Handle enum
	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}

	// Handle default
	if schema.Default != nil {
		result["default"] = schema.Default
	}

	// Handle array items
	if schema.Items != nil && schema.Items.Value != nil {
		result["items"] = schemaToMap(schema.Items)
	}

	// Handle object properties
	if len(schema.Properties) > 0 {
		props := make(map[string]interface{})
		for name, propRef := range schema.Properties {
			props[toSnakeCase(name)] = schemaToMap(propRef)
		}
		result["properties"] = props
	}

	return result
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func generateCode(config MCPConfig, tools []MCPTool) string {
	var sb strings.Builder

	sb.WriteString(`// Code generated by mcp-codegen from openapi.yaml. DO NOT EDIT.

package mcp

import "encoding/json"

// ServerInfo contains MCP server metadata
type ServerInfo struct {
	Name         string
	Version      string
	Instructions string
}

// GetServerInfo returns the MCP server metadata
func GetServerInfo() ServerInfo {
	return ServerInfo{
		Name:         `)
	sb.WriteString(fmt.Sprintf("%q", config.Name))
	sb.WriteString(`,
		Version:      `)
	sb.WriteString(fmt.Sprintf("%q", config.Version))
	sb.WriteString(`,
		Instructions: `)
	sb.WriteString(fmt.Sprintf("%q", config.Instructions))
	sb.WriteString(`,
	}
}

// Resource represents an MCP resource
type Resource struct {
	URI         string
	Name        string
	Description string
	MimeType    string
}

// GetResources returns all MCP resources
func GetResources() []Resource {
	return []Resource{
`)

	for _, r := range config.Resources {
		sb.WriteString(fmt.Sprintf(`		{
			URI:         %q,
			Name:        %q,
			Description: %q,
			MimeType:    %q,
		},
`, r.URI, r.Name, r.Description, r.MimeType))
	}

	sb.WriteString(`	}
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

// GetTools returns all MCP tool definitions
func GetTools() []Tool {
	return []Tool{
`)

	for _, t := range tools {
		inputSchemaJSON, _ := json.MarshalIndent(t.InputSchema, "\t\t\t", "\t")
		schemaStr := "parseSchema(`" + strings.ReplaceAll(string(inputSchemaJSON), "`", "` + \"`\" + `") + "`)"

		sb.WriteString(fmt.Sprintf(`		{
			Name:        %q,
			Description: %q,
			InputSchema: %s,
		},
`, t.Name, t.Description, schemaStr))
	}

	sb.WriteString(`	}
}

// parseSchema parses a JSON schema string into a map
func parseSchema(s string) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		panic("invalid schema: " + err.Error())
	}
	return result
}

// ToolNames returns a list of all tool names
func ToolNames() []string {
	tools := GetTools()
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}
`)

	return sb.String()
}
