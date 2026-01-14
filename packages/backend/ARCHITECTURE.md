# Backend Architecture

**Read this file completely before making any changes to the backend.**

## Overview

The backend is a REST API built with Go:
- **Chi** - lightweight HTTP router
- **oapi-codegen** - OpenAPI code generation for types and router
- **go-sqlite3** - SQLite database driver (CGO)
- **google/uuid** - UUID generation

## Source of Truth

The OpenAPI specification (`packages/api/openapi.yaml`) is the single source of truth for:
- Handler interfaces (generated server types)
- Request/response types
- API documentation

When the API changes, regenerate the types:

```bash
./tc exec sh -c "cd packages/backend && go generate ./..."
```

## Directory Structure

```
packages/backend/
├── ARCHITECTURE.md           # This file - read first!
├── go.mod
├── go.sum
├── cmd/
│   └── server/
│       └── main.go           # Entry point, server setup
├── internal/
│   ├── api/
│   │   └── openapi.gen.go    # Generated types (DO NOT EDIT)
│   ├── entity/
│   │   ├── trip.go           # Trip struct with DB tags
│   │   ├── item.go           # Item struct
│   │   └── document.go       # Document struct
│   ├── store/
│   │   └── sqlite.go         # Database access layer
│   ├── service/
│   │   └── service.go        # Business logic
│   └── handler/
│       └── handler.go        # HTTP handlers
└── data/
    └── travel.db             # SQLite database (gitignored)
```

## Core Principles

### 1. OpenAPI-First Development

The API spec is the contract. Handlers implement the generated `ServerInterface`:

```go
// internal/api/openapi.gen.go (generated)
type ServerInterface interface {
    ListTrips(w http.ResponseWriter, r *http.Request, params ListTripsParams)
    CreateTrip(w http.ResponseWriter, r *http.Request)
    GetTrip(w http.ResponseWriter, r *http.Request, tripId string)
    // ...
}
```

```go
// internal/handler/handler.go
type Handler struct {
    svc *service.Service
}

var _ api.ServerInterface = (*Handler)(nil)  // Compile-time check
```

### 2. Resource-Oriented REST

Every endpoint maps to a resource. No RPC-style endpoints.

```
✓ GET    /api/trips              # List trips
✓ POST   /api/trips              # Create trip
✓ GET    /api/trips/{tripId}     # Get trip
✓ PATCH  /api/trips/{tripId}     # Update trip
✓ DELETE /api/trips/{tripId}     # Delete trip
✓ GET    /api/trips/{tripId}/items    # List items for trip

✗ POST   /api/trips/{tripId}/addFlight    # NO - RPC style
✗ GET    /api/getUpcomingTrips            # NO - RPC style
```

### 3. Entity Design

Entities are internal structs with database tags. They convert to/from API types:

```go
// internal/entity/trip.go
type Trip struct {
    ID        uuid.UUID  `db:"id"`
    Name      string     `db:"name"`
    Purpose   string     `db:"purpose"`
    StartDate *time.Time `db:"start_date"`
    EndDate   *time.Time `db:"end_date"`
    Status    string     `db:"status"`
    Notes     *string    `db:"notes"`
    CreatedAt time.Time  `db:"created_at"`
    UpdatedAt time.Time  `db:"updated_at"`
}

func (t *Trip) ToAPI() api.Trip {
    // Convert internal entity to API type
}

func TripFromCreateRequest(req api.CreateTripRequest) *Trip {
    // Convert API request to internal entity
}
```

### 4. Layered Architecture

```
HTTP Request
    │
    ▼
┌─────────────┐
│   Handler   │  Parse request, call service, return response
└─────────────┘
    │
    ▼
┌─────────────┐
│   Service   │  Business logic, validation, orchestration
└─────────────┘
    │
    ▼
┌─────────────┐
│    Store    │  Database access, SQL queries
└─────────────┘
    │
    ▼
┌─────────────┐
│   Entity    │  Data structures with DB tags
└─────────────┘
```

### 5. Handlers Are Thin

Handlers parse requests and call services. No business logic:

```go
// internal/handler/handler.go - THIN
func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request) {
    var req api.CreateTripRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    trip, err := h.svc.CreateTrip(r.Context(), req)
    if err != nil {
        respondError(w, http.StatusInternalServerError, err.Error())
        return
    }

    respondJSON(w, http.StatusCreated, trip)
}
```

### 6. Services Own Business Logic

```go
// internal/service/service.go - LOGIC HERE
func (s *Service) CreateTrip(ctx context.Context, req api.CreateTripRequest) (*api.Trip, error) {
    // Validate business rules
    // Create entity
    // Persist to database
    // Return API type
}
```

### 7. Consistent Error Responses

All errors follow this format:

```json
{
    "error": "Trip not found"
}
```

Or with details:

```json
{
    "error": "Validation failed",
    "details": {
        "name": "required field"
    }
}
```

## Regenerating API Types

When the OpenAPI spec changes:

```bash
# From inside container
./tc exec sh -c "cd packages/backend && oapi-codegen -generate types,chi-server -package api ../api/openapi.yaml > internal/api/openapi.gen.go"
```

## Adding a New Resource

1. **Update OpenAPI spec** in `packages/api/openapi.yaml`
2. **Regenerate types** with oapi-codegen
3. **Create entity** in `internal/entity/`
4. **Add store methods** in `internal/store/sqlite.go`
5. **Add service methods** in `internal/service/service.go`
6. **Implement handlers** (satisfy new interface methods)
7. **Regenerate TypeScript types** in `packages/shared/`

## Testing

Run tests inside container:

```bash
./tc exec sh -c "cd packages/backend && go test ./..."
```

Manual API testing:

```bash
# List trips
./tc curl backend:3000/api/trips

# Create trip
./tc curl backend:3000/api/trips -X POST -H "Content-Type: application/json" \
  -d '{"name":"Test","purpose":"vacation","status":"planning"}'

# Health check
./tc curl backend:3000/health
```

## Forbidden Patterns

- Editing `internal/api/openapi.gen.go` directly
- Importing from `frontend` or `mcp-server`
- UI-related code
- Direct database queries in handlers (use service layer)
- Business logic in entities
- RPC-style endpoints
- Returning database entities directly (convert to API types)
