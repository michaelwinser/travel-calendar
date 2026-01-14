# MCP Server Architecture

**Read this file completely before making any changes to the MCP server.**

## Overview

The MCP server exposes travel calendar functionality to LLMs via the Model Context Protocol:
- **Tools** - Actions the LLM can take (CRUD operations, queries)
- **HTTP Streamable Transport** - JSON-RPC 2.0 over HTTP

Built with Go:
- Pure standard library (no external MCP SDK)
- JSON-RPC 2.0 request handling
- LLM-friendly markdown formatters

## Directory Structure

```
packages/mcp-server/
├── ARCHITECTURE.md           # This file - read first!
├── go.mod
├── go.sum
├── cmd/
│   └── mcp/
│       └── main.go           # Entry point, HTTP server
└── internal/
    ├── handler/
    │   └── mcp.go            # JSON-RPC router, MCP protocol
    ├── tools/
    │   ├── registry.go       # Tool registry pattern
    │   ├── trips.go          # Trip-related tools
    │   └── documents.go      # Document tools
    └── formatter/
        └── markdown.go       # LLM-friendly formatters
```

## Core Principles

### 1. MCP Server is a Facade

The MCP server does NOT contain business logic. It:
1. Translates MCP tool calls to backend API calls
2. Formats responses for LLM consumption
3. Provides semantic tool descriptions

```go
// CORRECT: Delegate to backend
func (r *Registry) handleGetTrips(args map[string]interface{}) (*ToolResult, error) {
    resp, err := http.Get(r.backendURL + "/api/trips")
    // ... handle response
    trips := parseTrips(resp)
    return &ToolResult{Content: formatter.FormatTrips(trips)}, nil
}

// WRONG: Direct database access
func (r *Registry) handleGetTrips(args map[string]interface{}) (*ToolResult, error) {
    db := getDatabase()
    trips := db.Query("SELECT * FROM trips")  // NO direct DB!
}
```

### 2. HTTP Streamable Transport

Uses JSON-RPC 2.0 over HTTP (not stdio):

```go
// Request format
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
        "name": "get_trips",
        "arguments": {"upcoming": true}
    }
}

// Response format
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "content": [
            {"type": "text", "text": "## Upcoming Trips\n\n..."}
        ]
    }
}
```

### 3. Tools Are Verb-Oriented

Tool names describe actions:

```go
// Tool naming: {verb}_{resource}[_{qualifier}]

// Good tool names
"get_trips"           // List/query trips
"get_trip"            // Get single trip
"create_trip"         // Create new trip
"update_trip"         // Update existing trip
"delete_trip"         // Delete trip
"search_trips"        // Full-text search
"add_item"            // Add item to trip
"delete_item"         // Delete item

// Bad tool names
"trips"               // Not a verb
"fetchAllTrips"       // camelCase
"trip_list"           // Noun-oriented
```

### 4. Tool Descriptions Are for LLMs

Write descriptions that help the LLM choose the right tool:

```go
ToolDefinition{
    Name: "get_trips",
    Description: `List trips with optional filters.

Use this tool to:
- Find upcoming trips: get_trips({ upcoming: true })
- Find past trips: get_trips({ past: true })
- Filter by purpose: get_trips({ purpose: "vacation" })

Returns formatted list of trips with dates and items.`,
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "upcoming": map[string]interface{}{"type": "boolean"},
            "past":     map[string]interface{}{"type": "boolean"},
            "purpose":  map[string]interface{}{"type": "string"},
        },
    },
}
```

### 5. Responses Are LLM-Friendly

Format responses as readable markdown, not raw JSON:

```go
// formatter/markdown.go
func FormatTrips(trips []Trip) string {
    if len(trips) == 0 {
        return "No trips found matching your criteria."
    }

    var sb strings.Builder
    sb.WriteString("## Trips\n\n")

    for _, trip := range trips {
        sb.WriteString(fmt.Sprintf("### %s\n", trip.Name))
        sb.WriteString(fmt.Sprintf("- **Purpose**: %s\n", trip.Purpose))
        if trip.StartDate != "" {
            sb.WriteString(fmt.Sprintf("- **Dates**: %s to %s\n",
                trip.StartDate, trip.EndDate))
        }
        sb.WriteString(fmt.Sprintf("- **Status**: %s\n", trip.Status))
        sb.WriteString("\n")
    }

    return sb.String()
}
```

### 6. Tool Registry Pattern

Tools are registered in a central registry:

```go
// tools/registry.go
type Registry struct {
    tools      map[string]ToolDefinition
    handlers   map[string]func(map[string]interface{}) (*ToolResult, error)
    backendURL string
}

func NewRegistry(backendURL string) *Registry {
    r := &Registry{
        tools:      make(map[string]ToolDefinition),
        handlers:   make(map[string]func(map[string]interface{}) (*ToolResult, error)),
        backendURL: backendURL,
    }
    r.registerTripTools()
    r.registerDocumentTools()
    return r
}
```

## MCP Protocol Implementation

The handler implements these JSON-RPC methods:

| Method | Description |
|--------|-------------|
| `initialize` | Protocol handshake, returns server info |
| `tools/list` | Returns available tools |
| `tools/call` | Execute a tool |
| `resources/list` | Returns available resources (currently empty) |

## Testing

Manual testing with curl:

```bash
# List available tools
./tc curl mcp:3001/mcp -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# Call get_trips tool
./tc curl mcp:3001/mcp -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_trips","arguments":{}}}'

# Health check
./tc curl mcp:3001/health
```

## Adding a New Tool

1. **Add tool definition** in `tools/trips.go` or appropriate file
2. **Implement handler** that calls backend API
3. **Add formatter** in `formatter/markdown.go` if needed
4. **Register tool** in registry's init method

Example:

```go
// In tools/trips.go
func (r *Registry) registerTripTools() {
    // Add tool definition
    r.tools["my_new_tool"] = ToolDefinition{
        Name:        "my_new_tool",
        Description: "Does something useful...",
        InputSchema: map[string]interface{}{...},
    }

    // Add handler
    r.handlers["my_new_tool"] = r.handleMyNewTool
}

func (r *Registry) handleMyNewTool(args map[string]interface{}) (*ToolResult, error) {
    // Call backend API
    // Format response
    // Return result
}
```

## Forbidden Patterns

- Direct database access (use backend API)
- Business logic (belongs in backend)
- UI code
- Modifying data without going through backend
- Raw JSON responses (format for LLM readability)
- Tools that combine multiple unrelated operations
