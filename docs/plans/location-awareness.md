# Plan: Location Awareness Feature

Issue: Location Awareness MVP (UC-TRP-008 through UC-TRP-013)
Components: api, backend, cli, mcp-server, frontend

## Summary

Implement location awareness for trips, enabling users to:
1. Configure base locations (home, work)
2. Set locations for trips (per-trip default or per-date)
3. Query where they will be on any given date or date range
4. Support multiple locations per day (travel days)
5. Default trips without explicit location to "Away"

This feature enables the core use case: "Where will I be on January 30th?"

---

## Phase 1: Data Model & API Foundation

### OpenAPI Changes

Add to `packages/api/openapi.yaml`:

- [ ] Add `Config` tag for configuration endpoints
- [ ] Add `Location` tag for location query endpoints

**New Schemas:**

```yaml
# Base location configuration
BaseLocations:
  type: object
  properties:
    home:
      type: string
      description: Home location (default "Home" if not set)
    work:
      type: string
      description: Work location (optional)

SetBaseLocationsRequest:
  type: object
  properties:
    home:
      type: string
      maxLength: 255
    work:
      type: string
      maxLength: 255

# Trip location data
TripDayLocation:
  type: object
  required: [date, locations]
  properties:
    date:
      type: string
      format: date
    locations:
      type: array
      items:
        type: string
      minItems: 1
      description: One or more locations for this date

SetTripLocationsRequest:
  type: object
  properties:
    locations:
      type: array
      items:
        $ref: '#/components/schemas/TripDayLocation'
      description: Per-date locations. If empty, trip uses default location.
    defaultLocation:
      type: string
      description: Default location for all dates not explicitly specified

# Location query responses
LocationOnDateResponse:
  type: object
  required: [date, locations, source]
  properties:
    date:
      type: string
      format: date
    locations:
      type: array
      items:
        type: string
      description: Location(s) on this date
    source:
      $ref: '#/components/schemas/LocationSource'

LocationSource:
  type: object
  required: [type]
  properties:
    type:
      type: string
      enum: [home, work, trip]
    tripId:
      type: string
      format: uuid
      description: Present when type is "trip"
    tripName:
      type: string
      description: Present when type is "trip"

LocationRangeSegment:
  type: object
  required: [startDate, endDate, locations, source]
  properties:
    startDate:
      type: string
      format: date
    endDate:
      type: string
      format: date
    locations:
      type: array
      items:
        type: string
    source:
      $ref: '#/components/schemas/LocationSource'
```

**New Endpoints:**

```yaml
# Configuration endpoints
/api/config/locations:
  get:
    operationId: getBaseLocations
    tags: [Config]
    summary: Get base locations
    responses:
      '200':
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/BaseLocations'
  put:
    operationId: setBaseLocations
    tags: [Config]
    summary: Set base locations
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/SetBaseLocationsRequest'
    responses:
      '200':
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/BaseLocations'

# Trip location endpoints
/api/trips/{tripId}/locations:
  get:
    operationId: getTripLocations
    tags: [Trips]
    summary: Get locations for a trip
    responses:
      '200':
        content:
          application/json:
            schema:
              type: array
              items:
                $ref: '#/components/schemas/TripDayLocation'
  put:
    operationId: setTripLocations
    tags: [Trips]
    summary: Set locations for a trip
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/SetTripLocationsRequest'
    responses:
      '200':
        content:
          application/json:
            schema:
              type: array
              items:
                $ref: '#/components/schemas/TripDayLocation'

# Location query endpoints
/api/location/on/{date}:
  get:
    operationId: getLocationOnDate
    tags: [Location]
    summary: Get user location on a specific date
    parameters:
      - name: date
        in: path
        required: true
        schema:
          type: string
          format: date
    responses:
      '200':
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/LocationOnDateResponse'

/api/location/range:
  get:
    operationId: getLocationRange
    tags: [Location]
    summary: Get user locations for a date range
    parameters:
      - name: from
        in: query
        required: true
        schema:
          type: string
          format: date
      - name: to
        in: query
        required: true
        schema:
          type: string
          format: date
    responses:
      '200':
        content:
          application/json:
            schema:
              type: array
              items:
                $ref: '#/components/schemas/LocationRangeSegment'
```

**Modify Trip schema:**

```yaml
# Add to Trip schema
Trip:
  properties:
    # ... existing properties ...
    locations:
      type: array
      items:
        $ref: '#/components/schemas/TripDayLocation'
      description: Location data for this trip (only included when fetching single trip)
```

**Modify CreateTripRequest schema:**

```yaml
# Add optional location to CreateTripRequest
CreateTripRequest:
  properties:
    # ... existing properties (name, purpose, startDate, endDate, status, notes) ...
    location:
      type: string
      maxLength: 255
      description: Default location for all days of this trip. Sets all dates to this location on creation.
```

This enables the convenience of:
```bash
travel trips create --name "FOSDEM" --start 2025-01-29 --end 2025-02-02 --purpose conference --location "Brussels"
```
Which creates the trip AND sets all days to "Brussels" in one command. User can refine day-by-day later.

---

## Phase 2: Backend Implementation (Go)

### Backend Changes

- [ ] Regenerate types from OpenAPI

**New Database Tables:**

```sql
-- User configuration (single row, user_id for future multi-tenancy)
CREATE TABLE IF NOT EXISTS config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Trip locations (per-date)
CREATE TABLE IF NOT EXISTS trip_locations (
    id TEXT PRIMARY KEY,
    trip_id TEXT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    date TEXT NOT NULL,
    location TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(trip_id, date, location)
);

CREATE INDEX IF NOT EXISTS idx_trip_locations_trip_id ON trip_locations(trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_locations_date ON trip_locations(date);
```

**New Entity Files:**

- [ ] Create `internal/entity/config.go` - Config key-value entity
- [ ] Create `internal/entity/trip_location.go` - TripLocation entity with ToAPI conversion

**Store Methods:**

- [ ] Add config methods to `internal/store/sqlite.go`:
  - `GetConfig(key string) (*string, error)`
  - `SetConfig(key, value string) error`
  - `DeleteConfig(key string) error`
- [ ] Add trip location methods:
  - `GetTripLocations(tripID uuid.UUID) ([]entity.TripLocation, error)`
  - `SetTripLocations(tripID uuid.UUID, locations []entity.TripLocation) error`
  - `GetTripsInDateRange(from, to time.Time) ([]entity.Trip, error)` - for location queries

**Service Methods:**

- [ ] Add to `internal/service/service.go`:
  - `GetBaseLocations() (*api.BaseLocations, error)`
  - `SetBaseLocations(req *api.SetBaseLocationsRequest) (*api.BaseLocations, error)`
  - `GetTripLocations(tripID uuid.UUID) ([]api.TripDayLocation, error)`
  - `SetTripLocations(tripID uuid.UUID, req *api.SetTripLocationsRequest) ([]api.TripDayLocation, error)`
  - `GetLocationOnDate(date time.Time) (*api.LocationOnDateResponse, error)`
  - `GetLocationRange(from, to time.Time) ([]api.LocationRangeSegment, error)`

**Location Resolution Logic:**

```go
// GetLocationOnDate resolves location for a specific date
func (s *Service) GetLocationOnDate(date time.Time) (*api.LocationOnDateResponse, error) {
    // 1. Find any trip that spans this date
    trip := s.store.GetTripForDate(date)

    if trip != nil {
        // 2. Check for explicit location on this date
        locations := s.store.GetTripLocationsForDate(trip.ID, date)
        if len(locations) > 0 {
            return &api.LocationOnDateResponse{
                Date: date,
                Locations: locations,
                Source: api.LocationSource{
                    Type: "trip",
                    TripId: trip.ID,
                    TripName: trip.Name,
                },
            }, nil
        }

        // 3. No explicit location - return "Away"
        return &api.LocationOnDateResponse{
            Date: date,
            Locations: []string{"Away"},
            Source: api.LocationSource{
                Type: "trip",
                TripId: trip.ID,
                TripName: trip.Name,
            },
        }, nil
    }

    // 4. Not on a trip - return home location
    home := s.GetConfigOrDefault("home-location", "Home")
    return &api.LocationOnDateResponse{
        Date: date,
        Locations: []string{home},
        Source: api.LocationSource{Type: "home"},
    }, nil
}
```

**Handler Methods:**

- [ ] Implement all new ServerInterface methods in `internal/handler/handler.go`

---

## Phase 3: CLI Implementation (Go)

### CLI Changes

- [ ] Regenerate client from OpenAPI

**New Commands:**

```
travel config
  get <key>              # Get a config value
  set <key> <value>      # Set a config value
  unset <key>            # Remove a config value
  list                   # List all config

travel trips create ... --location <location>
  # New optional flag: sets all days to this location on creation

travel trips set-location <trip-id> <location> [--date YYYY-MM-DD] [--start YYYY-MM-DD --end YYYY-MM-DD]
  # Set location for: all days (no flags), single day (--date), or range (--start/--end)

travel trips add-location <trip-id> <location> --date YYYY-MM-DD
travel trips add-location <trip-id> <location> --start YYYY-MM-DD --end YYYY-MM-DD
  # Add additional location to existing day(s) - for travel days with multiple locations

travel location
  on <date>              # Where am I on this date?
  from <date> to <date>  # Where am I in this range?
```

**Examples:**

```bash
# Create trip with location in one command
travel trips create --name "FOSDEM" --start 2025-01-29 --end 2025-02-02 \
  --purpose conference --location "Brussels"

# Later, refine the travel day
travel trips set-location $ID "London" --date 2025-01-29        # Overwrite day 1
travel trips add-location $ID "Brussels" --date 2025-01-29      # Add second location (travel day)

# Set location for a range within the trip
travel trips set-location $ID "Paris" --start 2025-01-31 --end 2025-02-01  # Side trip
```

**New Files:**

- [ ] Create `internal/cmd/config.go` - Config commands
- [ ] Create `internal/cmd/location.go` - Location query commands
- [ ] Update `internal/cmd/trips.go` - Add set-location, add-location subcommands

**Output Formatters:**

- [ ] Add to `internal/output/output.go`:
  - `PrintConfig(key, value string)`
  - `PrintLocationOnDate(loc api.LocationOnDateResponse)`
  - `PrintLocationRange(segments []api.LocationRangeSegment)`

---

## Phase 4: MCP Server Implementation (Go)

### MCP Server Changes

**New Tools:**

- [ ] Create `internal/tools/location.go`:

```go
// get_location_on_date - Query location for a specific date
ToolDefinition{
    Name: "get_location_on_date",
    Description: `Find out where the user will be on a specific date.

Use this to answer questions like:
- "Where will I be on January 30th?"
- "Am I traveling on March 15?"
- "What's my location next Tuesday?"

Returns the location(s) and whether it's from a trip or home.`,
    InputSchema: map[string]interface{}{
        "type": "object",
        "required": []string{"date"},
        "properties": map[string]interface{}{
            "date": map[string]interface{}{
                "type": "string",
                "format": "date",
                "description": "Date to query (YYYY-MM-DD)",
            },
        },
    },
}

// get_location_range - Query locations for a date range
ToolDefinition{
    Name: "get_location_range",
    Description: `Get a timeline of where the user will be across a date range.

Use this to answer questions like:
- "Where will I be in January?"
- "What's my travel schedule for Q1?"
- "Am I home between the 10th and 20th?"

Returns a timeline of location segments with dates and sources.`,
    InputSchema: map[string]interface{}{
        "type": "object",
        "required": []string{"from", "to"},
        "properties": map[string]interface{}{
            "from": map[string]interface{}{
                "type": "string",
                "format": "date",
            },
            "to": map[string]interface{}{
                "type": "string",
                "format": "date",
            },
        },
    },
}

// set_trip_locations - Set locations for a trip
ToolDefinition{
    Name: "set_trip_locations",
    Description: `Set the location(s) for a trip.

Use this when the user says things like:
- "I'll be in Brussels for FOSDEM"
- "Add London to my Europe trip"
- "On Jan 30 I'll travel from London to Brussels"`,
    InputSchema: map[string]interface{}{
        "type": "object",
        "required": []string{"tripId"},
        "properties": map[string]interface{}{
            "tripId": {...},
            "defaultLocation": {...},
            "locations": {...},  // per-date overrides
        },
    },
}
```

**Formatters:**

- [ ] Add to `internal/formatter/markdown.go`:
  - `FormatLocationOnDate(resp map[string]interface{}) string`
  - `FormatLocationRange(segments []map[string]interface{}) string`

---

## Phase 5: Frontend Implementation

### Frontend Changes

**Components:**

- [ ] Create `src/lib/components/location/LocationBadge.svelte` - Shows location inline
- [ ] Create `src/lib/components/location/LocationEditor.svelte` - Edit trip locations (see UX below)
- [ ] Create `src/lib/components/location/LocationTimeline.svelte` - Date range view
- [ ] Create `src/lib/components/settings/BaseLocationSettings.svelte` - Configure home/work

**LocationEditor UX - Grid/Spreadsheet View:**

The LocationEditor should display trip days as a grid, enabling intuitive day-by-day editing:

```
┌─────────────────────────────────────────────────────────┐
│  FOSDEM Brussels Conference                             │
│  Jan 29 - Feb 2, 2025                                   │
├─────────────────────────────────────────────────────────┤
│  Date       │ Location(s)                               │
├─────────────┼───────────────────────────────────────────┤
│  Wed Jan 29 │ [London] [Brussels]        [+ Add]        │
│  Thu Jan 30 │ [Brussels]                 [+ Add]        │
│  Fri Jan 31 │ [Brussels]                 [+ Add]        │
│  Sat Feb 1  │ [Brussels]                 [+ Add]        │
│  Sun Feb 2  │ [Brussels]                 [+ Add]        │
└─────────────┴───────────────────────────────────────────┘
```

Key interactions:
- Click location badge to edit inline
- Click [+ Add] to add another location (travel days)
- Multi-select days (shift-click or drag) then set location for all
- Autocomplete from previously used locations
- Visual indicator for travel days (multiple locations)

This grid view makes it easy to:
1. See the whole trip at a glance
2. Edit individual days quickly
3. Handle travel days with multiple locations
4. Bulk-edit ranges (select Jan 30 - Feb 2, set all to "Brussels")

**Stores:**

- [ ] Create `src/lib/stores/config.ts` - Configuration store

**Routes:**

- [ ] Add settings page section for base locations
- [ ] Update trip detail page to show/edit locations

---

## Phase 6: E2E Tests

### E2E Test Scripts

- [ ] `tests/e2e/uc-008-configure-base-locations.sh`
- [ ] `tests/e2e/uc-009-set-trip-locations.sh`
- [ ] `tests/e2e/uc-010-query-location-on-date.sh`
- [ ] `tests/e2e/uc-011-query-location-range.sh`
- [ ] `tests/e2e/uc-012-trip-defaults-to-away.sh`
- [ ] `tests/e2e/uc-013-llm-queries-location.sh` (MCP tool test)

---

## Integration Points

| Component | Interface | Description |
|-----------|-----------|-------------|
| API | `PUT /api/config/locations` | Set home/work locations |
| API | `GET /api/config/locations` | Get base locations |
| API | `POST /api/trips` | Create trip (now with optional `location` field) |
| API | `PUT /api/trips/{id}/locations` | Set trip locations |
| API | `GET /api/location/on/{date}` | Query single date |
| API | `GET /api/location/range?from=&to=` | Query date range |
| CLI | `travel config set home-location "NYC"` | Configure home |
| CLI | `travel trips create ... --location "Brussels"` | Create trip with location |
| CLI | `travel trips set-location <id> "Brussels"` | Set trip location (all days) |
| CLI | `travel trips set-location <id> "Paris" --start X --end Y` | Set location for date range |
| CLI | `travel trips add-location <id> "Brussels" --date X` | Add location to day |
| CLI | `travel location on 2025-01-30` | Query location |
| MCP | `get_location_on_date` | LLM queries location |
| MCP | `get_location_range` | LLM queries timeline |
| MCP | `set_trip_locations` | LLM sets locations |

---

## Testing Strategy

1. **Backend Go tests** - Unit tests for location resolution logic
   ```bash
   ./tc exec sh -c "cd packages/backend && go test ./..."
   ```

2. **CLI build and manual test**
   ```bash
   cd packages/cli && go build -o travel ./cmd/travel
   ./travel config set home-location "New York, USA"
   ./travel location on 2025-01-30 --json
   ```

3. **MCP tool curl test**
   ```bash
   ./tc curl mcp:3001/mcp -X POST -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_location_on_date","arguments":{"date":"2025-01-30"}}}'
   ```

4. **Frontend build**
   ```bash
   ./tc exec pnpm --filter @travel-calendar/frontend build
   ```

5. **E2E tests** - Run shell scripts in `tests/e2e/`

---

## Design Decisions (Resolved)

1. **Work location behavior**: Home is always the default when not traveling. The subtleties of commuting (auto-selecting work on weekdays) are beyond MVP scope. Work location is stored but never auto-selected.

2. **Overlapping trips**: Return all locations from all trips that span the queried date. The model already supports multiple locations per day, so overlapping trips aren't special - just more locations. Future enhancement: organize results by trip status (confirmed, completed, etc).

3. **Location format validation**: No validation for MVP. Locations are free-form semantic text (e.g., "NYC", "New York, USA", "Mom's house"). Future enhancement: UI can autocomplete against previously entered locations.

4. **Trips without dates**: The current schema allows trips without dates (only name and purpose are required). For MVP: location queries only consider trips that have both start and end dates. Trips without dates won't participate in location results - they're in a "planning" state with no timeline yet. This supports a future feature of planning trips without precise dates.

---

## Execution Order

1. **api** - OpenAPI spec changes (this document's Phase 1)
2. **shared** - Regenerate TypeScript types
3. **backend** - Database, entities, service, handlers (Phase 2)
4. **cli** - Commands and output formatters (Phase 3)
5. **mcp-server** - Tools and formatters (Phase 4)
6. **frontend** - Components and routes (Phase 5)
7. **e2e tests** - Verification scripts (Phase 6)

---

## Approval

- [x] Plan reviewed and approved (2026-01-14)
