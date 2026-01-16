# Feature: Trip Management

## Overview

Core CRUD operations for trips - the primary entity in the travel calendar. A trip contains a name, date range, purpose, and a loose collection of items (flights, hotels, events) that are organized by date.

## User Value

Users need to create, view, edit, and delete trips. Trips are the organizing container for all travel-related items. Without trip management, there's no way to plan or track travel.

---

## Use Cases

### [UC-TRP-001] Create a basic trip

**Actor**: User (via UI or CLI)

**Preconditions**:
- None

**Steps**:
1. User provides trip name, purpose, start date, end date
2. System validates input
3. System creates trip with status "planned"

**Expected Result**:
- Trip is created with unique ID
- Trip appears in trip list
- Trip has no items initially

**CLI Test**:
```bash
# Action
RESULT=$(travel trips create \
  --name "FOSDEM 2025" \
  --purpose conference \
  --start 2025-01-29 \
  --end 2025-02-02 \
  --json)

# Verify
echo $RESULT | jq -e '.id != null'
echo $RESULT | jq -e '.name == "FOSDEM 2025"'
echo $RESULT | jq -e '.purpose == "conference"'
echo $RESULT | jq -e '.status == "planned"'

# Cleanup
TRIP_ID=$(echo $RESULT | jq -r '.id')
travel trips delete $TRIP_ID
```

---

### [UC-TRP-002] List upcoming trips

**Actor**: User or LLM

**Preconditions**:
- At least one trip exists with start date in the future

**Steps**:
1. User requests upcoming trips
2. System filters trips where startDate > today
3. System returns trips sorted by start date ascending

**Expected Result**:
- Only future trips returned
- Trips sorted chronologically
- Past trips not included

**CLI Test**:
```bash
# Setup - create past and future trips
PAST=$(travel trips create --name "Past Trip" --purpose work \
  --start 2024-01-01 --end 2024-01-02 --json | jq -r '.id')
FUTURE=$(travel trips create --name "Future Trip" --purpose vacation \
  --start 2099-01-01 --end 2099-01-05 --json | jq -r '.id')

# Action
UPCOMING=$(travel trips list --upcoming --json)

# Verify
echo $UPCOMING | jq -e 'map(.name) | contains(["Future Trip"])'
echo $UPCOMING | jq -e 'map(.name) | contains(["Past Trip"]) | not'

# Cleanup
travel trips delete $PAST
travel trips delete $FUTURE
```

---

### [UC-TRP-003] Get trip with all items

**Actor**: User or LLM

**Preconditions**:
- Trip exists
- Trip has items (flights, hotels, etc.)

**Steps**:
1. User requests trip by ID
2. System fetches trip and all associated items
3. System returns trip with items organized by date

**Expected Result**:
- Trip details returned
- All items included
- Items sorted by date

**CLI Test**:
```bash
# Setup
TRIP_ID=$(travel trips create --name "Test Trip" --purpose vacation \
  --start 2025-03-01 --end 2025-03-05 --json | jq -r '.id')
travel items add $TRIP_ID flight --from EWR --to LAX --date 2025-03-01
travel items add $TRIP_ID hotel --name "LA Hotel" --location "Los Angeles" \
  --checkin 2025-03-01 --checkout 2025-03-05

# Action
TRIP=$(travel trips get $TRIP_ID --json)

# Verify
echo $TRIP | jq -e '.name == "Test Trip"'
echo $TRIP | jq -e '.items | length == 2'
echo $TRIP | jq -e '.items[0].date <= .items[1].date'  # sorted by date

# Cleanup
travel trips delete $TRIP_ID
```

---

### [UC-TRP-004] Update trip details

**Actor**: User

**Preconditions**:
- Trip exists

**Steps**:
1. User provides trip ID and fields to update
2. System validates input
3. System updates trip
4. System returns updated trip

**Expected Result**:
- Trip is updated
- Only specified fields changed
- updatedAt timestamp updated

**CLI Test**:
```bash
# Setup
TRIP_ID=$(travel trips create --name "Original Name" --purpose work \
  --start 2025-03-01 --end 2025-03-05 --json | jq -r '.id')

# Action
UPDATED=$(travel trips update $TRIP_ID --name "New Name" --status confirmed --json)

# Verify
echo $UPDATED | jq -e '.name == "New Name"'
echo $UPDATED | jq -e '.status == "confirmed"'
echo $UPDATED | jq -e '.purpose == "work"'  # unchanged

# Cleanup
travel trips delete $TRIP_ID
```

---

### [UC-TRP-005] Delete trip and all items

**Actor**: User

**Preconditions**:
- Trip exists (with or without items)

**Steps**:
1. User requests trip deletion
2. System deletes trip
3. System cascades delete to all items
4. System returns success

**Expected Result**:
- Trip no longer exists
- All associated items deleted
- Documents become unassociated (not deleted)

**CLI Test**:
```bash
# Setup
TRIP_ID=$(travel trips create --name "To Delete" --purpose work \
  --start 2025-03-01 --end 2025-03-05 --json | jq -r '.id')
travel items add $TRIP_ID flight --from EWR --to LAX --date 2025-03-01

# Action
travel trips delete $TRIP_ID

# Verify - trip should not exist
! travel trips get $TRIP_ID 2>/dev/null
```

---

### [UC-TRP-006] Search trips by name or location

**Actor**: User or LLM

**Preconditions**:
- Trips exist with various names and locations

**Steps**:
1. User provides search query
2. System searches trip names and item locations
3. System returns matching trips

**Expected Result**:
- Trips matching query returned
- Search is case-insensitive
- Partial matches included

**CLI Test**:
```bash
# Setup
TRIP1=$(travel trips create --name "FOSDEM Brussels" --purpose conference \
  --start 2025-01-29 --end 2025-02-02 --json | jq -r '.id')
TRIP2=$(travel trips create --name "NYC Meeting" --purpose work \
  --start 2025-03-01 --end 2025-03-02 --json | jq -r '.id')

# Action
RESULTS=$(travel trips search "brussels" --json)

# Verify
echo $RESULTS | jq -e 'length == 1'
echo $RESULTS | jq -e '.[0].name == "FOSDEM Brussels"'

# Cleanup
travel trips delete $TRIP1
travel trips delete $TRIP2
```

---

### [UC-TRP-007] Ask LLM about next trip

**Actor**: LLM (via MCP)

**Preconditions**:
- At least one upcoming trip exists

**Steps**:
1. User asks LLM "What's my next trip?"
2. LLM calls `get_trips` tool with `upcoming: true`
3. LLM identifies nearest trip
4. LLM responds with trip details

**Expected Result**:
- LLM correctly identifies next trip
- Response includes name, dates, purpose
- Response is natural language

**MCP Tool Call**:
```json
{
  "tool": "get_trips",
  "arguments": { "upcoming": true }
}
```

---

## Location Awareness

A key capability of the travel calendar is knowing where the user is on any given day. Location is a semantic concept (city, country, "Away") rather than GPS coordinates.

### Location Data Model

**Base Locations** (user configuration):
- `home`: Primary residence (default: "Home")
- `work`: Office location if different from home (optional)

**Trip Locations**:
- Each trip can have one or more locations
- Locations can be assigned to specific dates within the trip
- If no explicit location is set, trips default to "Away" (indicating not-home without specifying where)
- A single day can have multiple locations (e.g., travel day: "London" to "Brussels")

```
Trip
├── locations: [
│     { date: "2025-01-29", locations: ["London, UK"] },
│     { date: "2025-01-30", locations: ["London, UK", "Brussels, Belgium"] },  // Eurostar day
│     { date: "2025-01-31", locations: ["Brussels, Belgium"] },
│     { date: "2025-02-01", locations: ["Brussels, Belgium"] },
│     { date: "2025-02-02", locations: ["Brussels, Belgium"] }
│   ]
└── ...
```

**Location Resolution** (for any date):
1. If date falls within a trip with explicit location(s) for that date, return those
2. If date falls within a trip without explicit location, return "Away"
3. Otherwise, return base location (home or work based on context)

---

### [UC-TRP-008] Configure base locations

**Actor**: User

**Preconditions**:
- None

**Steps**:
1. User sets their home location
2. Optionally, user sets their work location

**Expected Result**:
- Base locations stored in user configuration
- Default "Home" used if not explicitly configured
- Work location is optional

**CLI Test**:
```bash
# Action - set home location
travel config set home-location "New York, USA"

# Verify
HOME=$(travel config get home-location --json)
echo $HOME | jq -e '. == "New York, USA"'

# Action - set work location (optional)
travel config set work-location "Jersey City, USA"

# Verify
WORK=$(travel config get work-location --json)
echo $WORK | jq -e '. == "Jersey City, USA"'

# Verify default when not set
travel config unset home-location
DEFAULT=$(travel config get home-location --json)
echo $DEFAULT | jq -e '. == "Home"'

# Cleanup - restore
travel config set home-location "New York, USA"
```

---

### [UC-TRP-009] Set trip location(s)

**Actor**: User

**Preconditions**:
- Trip exists

**Steps**:
1. User adds location(s) to a trip
2. Location can apply to entire trip or specific dates
3. Multiple locations can exist for the same date

**Expected Result**:
- Location(s) associated with trip
- If no dates specified, location applies to all trip dates
- Multiple locations on same date supported (travel days)

**CLI Test**:
```bash
# Setup
TRIP_ID=$(travel trips create --name "Europe Trip" --purpose vacation \
  --start 2025-01-29 --end 2025-02-02 --json | jq -r '.id')

# Action - set location for entire trip
travel trips set-location $TRIP_ID "Brussels, Belgium"

# Verify
TRIP=$(travel trips get $TRIP_ID --json)
echo $TRIP | jq -e '.locations[0].locations[0] == "Brussels, Belgium"'

# Action - override specific date with multiple locations (travel day)
travel trips set-location $TRIP_ID "London, UK" --date 2025-01-29
travel trips add-location $TRIP_ID "Brussels, Belgium" --date 2025-01-29

# Verify travel day has both
TRIP=$(travel trips get $TRIP_ID --json)
echo $TRIP | jq -e '.locations[] | select(.date == "2025-01-29") | .locations | length == 2'

# Cleanup
travel trips delete $TRIP_ID
```

---

### [UC-TRP-010] Query user location on a specific date

**Actor**: User or LLM

**Preconditions**:
- User configuration exists (or defaults apply)
- Zero or more trips exist

**Steps**:
1. User asks "Where will I be on DATE?"
2. System checks if DATE falls within any trip
3. If within a trip: return the trip's location(s) for that date
4. If not within a trip: return base location (home)

**Expected Result**:
- Location(s) returned for the date
- Multiple locations possible (travel day, or home+work)
- Source indicated (trip name or "home"/"work")
- "Away" returned for trips without explicit location

**CLI Test**:
```bash
# Setup
TRIP_ID=$(travel trips create --name "Brussels Conference" --purpose conference \
  --start 2025-01-29 --end 2025-02-02 --json | jq -r '.id')
travel trips set-location $TRIP_ID "Brussels, Belgium"
travel config set home-location "New York, USA"

# Action - query during trip
DURING=$(travel location on 2025-01-30 --json)

# Verify
echo $DURING | jq -e '.locations[0] == "Brussels, Belgium"'
echo $DURING | jq -e '.source.type == "trip"'
echo $DURING | jq -e '.source.tripName == "Brussels Conference"'

# Action - query outside trip
OUTSIDE=$(travel location on 2025-03-15 --json)

# Verify
echo $OUTSIDE | jq -e '.locations[0] == "New York, USA"'
echo $OUTSIDE | jq -e '.source.type == "home"'

# Cleanup
travel trips delete $TRIP_ID
```

---

### [UC-TRP-011] Query user location for a date range

**Actor**: User or LLM

**Preconditions**:
- User configuration exists (or defaults apply)
- Zero or more trips exist

**Steps**:
1. User asks "Where will I be from DATE1 to DATE2?"
2. System builds day-by-day location list
3. System groups consecutive days with same location(s)
4. System returns consolidated timeline

**Expected Result**:
- Timeline of location segments covering the range
- Consecutive days at same location grouped
- Each segment: location(s), start date, end date, source
- Gaps between trips show base location

**CLI Test**:
```bash
# Setup - create trips with gap
TRIP1=$(travel trips create --name "Brussels" --purpose conference \
  --start 2025-01-29 --end 2025-02-02 --json | jq -r '.id')
travel trips set-location $TRIP1 "Brussels, Belgium"

TRIP2=$(travel trips create --name "London" --purpose work \
  --start 2025-02-10 --end 2025-02-12 --json | jq -r '.id')
travel trips set-location $TRIP2 "London, UK"

travel config set home-location "New York, USA"

# Action - query range spanning both trips
RANGE=$(travel location from 2025-01-28 to 2025-02-15 --json)

# Verify - should have 5 segments
echo $RANGE | jq -e 'length == 5'
echo $RANGE | jq -e '.[0].locations[0] == "New York, USA"'   # before trip1
echo $RANGE | jq -e '.[0].startDate == "2025-01-28"'
echo $RANGE | jq -e '.[1].locations[0] == "Brussels, Belgium"'  # trip1
echo $RANGE | jq -e '.[2].locations[0] == "New York, USA"'   # gap
echo $RANGE | jq -e '.[3].locations[0] == "London, UK"'      # trip2
echo $RANGE | jq -e '.[4].locations[0] == "New York, USA"'   # after trip2

# Cleanup
travel trips delete $TRIP1
travel trips delete $TRIP2
```

---

### [UC-TRP-012] Trip defaults to "Away" without explicit location

**Actor**: User

**Preconditions**:
- Trip exists without location set

**Steps**:
1. User creates trip without specifying location
2. User queries location for a date within the trip

**Expected Result**:
- Location returns "Away" (not home, not null)
- Indicates user is traveling but destination unspecified
- Trip is clearly distinguished from being at home

**CLI Test**:
```bash
# Setup - trip without location
TRIP_ID=$(travel trips create --name "Mystery Trip" --purpose vacation \
  --start 2025-04-01 --end 2025-04-05 --json | jq -r '.id')

travel config set home-location "New York, USA"

# Action - query date within trip (no location set)
RESULT=$(travel location on 2025-04-03 --json)

# Verify - should return "Away", not home
echo $RESULT | jq -e '.locations[0] == "Away"'
echo $RESULT | jq -e '.source.type == "trip"'
echo $RESULT | jq -e '.source.tripName == "Mystery Trip"'

# Cleanup
travel trips delete $TRIP_ID
```

---

### [UC-TRP-013] LLM queries user location

**Actor**: LLM (via MCP)

**Preconditions**:
- At least one trip exists with location data

**Steps**:
1. User asks LLM "Where will I be on January 30th?"
2. LLM calls `get_location_on_date` tool
3. LLM responds with location information

**Expected Result**:
- LLM correctly identifies location
- Response is natural language
- Includes trip context if applicable

**MCP Tool Call**:
```json
{
  "tool": "get_location_on_date",
  "arguments": { "date": "2025-01-30" }
}
```

---

### [UC-TRP-014] Detect calendar conflicts with trip locations (Future)

**Actor**: LLM

**Preconditions**:
- Calendar integration enabled (Phase 2+)
- User has calendar events with location data
- User has trips with location data

**Steps**:
1. LLM asks "Does the user have meetings that conflict with their location?"
2. System retrieves calendar events for the date range
3. System determines user location for each event's date
4. System identifies events where event location is far from user location
5. System returns potential conflicts

**Expected Result**:
- Conflicts identified when event location differs significantly from user location
- "Near home" events (dentist in same metro area) not flagged
- Virtual/online meetings not flagged
- Each conflict includes: event details, event location, user location

**Notes**:
- Requires calendar integration (Phase 2+)
- "Significant distance" needs proximity calculation
- Local appointments should be distinguishable from remote meetings

**MCP Tool Call**:
```json
{
  "tool": "detect_location_conflicts",
  "arguments": {
    "startDate": "2025-01-01",
    "endDate": "2025-03-31"
  }
}
```

---

## MVP Scope: Location Awareness

### Included in MVP
- [UC-TRP-008] Configure base locations (home, work)
- [UC-TRP-009] Set trip location(s) - per-trip and per-date
- [UC-TRP-010] Query location on specific date
- [UC-TRP-011] Query location for date range
- [UC-TRP-012] "Away" default for trips without location
- [UC-TRP-013] LLM location queries
- Multiple locations per day supported

### Deferred to Later
- [UC-TRP-014] Calendar conflict detection (requires calendar integration)
- Location proximity/distance calculations
- "Near home" vs "far from home" distinction
- Automatic location inference from flight/hotel items
- Natural language trip creation (`travel NYC next Wed`)
- Simplified CLI (`travel --date 2026-04-13 --location NYC`)

### Future Vision

The location model enables powerful future capabilities:

**Simplified Trip Creation** (beyond MVP):
```bash
# Future: natural defaults
travel --date 2026-04-13 --location NYC
# Creates trip named "NYC" for single day

# Future: natural language
travel NYC next Wed
# Parses and creates appropriate trip
```

**Intelligent Conflict Detection** (Phase 2+):
- Distinguish local appointments (dentist) from remote meetings
- Understand metro areas (Jersey City is "near" NYC)
- Flag only truly conflicting calendar events

**Item-Derived Locations** (future enhancement):
- Infer daily location from hotel bookings
- Track arrival/departure from flight items
- Build location timeline automatically from items

---

## API Endpoints

| Method | Endpoint | Description | Use Cases |
|--------|----------|-------------|-----------|
| POST | `/api/trips` | Create trip | UC-TRP-001 |
| GET | `/api/trips` | List trips (with filters) | UC-TRP-002 |
| GET | `/api/trips/:id` | Get trip with items | UC-TRP-003 |
| PATCH | `/api/trips/:id` | Update trip | UC-TRP-004 |
| DELETE | `/api/trips/:id` | Delete trip | UC-TRP-005 |
| GET | `/api/trips/search` | Search trips | UC-TRP-006 |
| PUT | `/api/trips/:id/locations` | Set trip locations | UC-TRP-009 |
| GET | `/api/location/on/:date` | Get location on date | UC-TRP-010 |
| GET | `/api/location/range` | Get locations for range | UC-TRP-011 |
| GET | `/api/config/locations` | Get base locations | UC-TRP-008 |
| PUT | `/api/config/locations` | Set base locations | UC-TRP-008 |

---

## MCP Tools

| Tool | Description | Use Cases |
|------|-------------|-----------|
| `get_trips` | List/filter trips | UC-TRP-002, UC-TRP-007 |
| `get_trip` | Get single trip | UC-TRP-003 |
| `create_trip` | Create trip | UC-TRP-001 |
| `update_trip` | Update trip | UC-TRP-004 |
| `delete_trip` | Delete trip | UC-TRP-005 |
| `search_trips` | Search trips | UC-TRP-006 |
| `set_trip_locations` | Set locations for trip | UC-TRP-009 |
| `get_location_on_date` | Get location on specific date | UC-TRP-010, UC-TRP-013 |
| `get_location_range` | Get locations for date range | UC-TRP-011 |
| `detect_location_conflicts` | Find calendar conflicts (future) | UC-TRP-014 |

---

## UI Components

| Component | Views | Use Cases |
|-----------|-------|-----------|
| TripCard | List item | UC-TRP-002 |
| TripChip | Inline reference | - |
| TripDetail | Full page | UC-TRP-003 |
| TripForm | Create/edit | UC-TRP-001, UC-TRP-004 |
| TripCalendarBar | Calendar view | UC-TRP-002 |
| LocationEditor | Trip detail, inline | UC-TRP-009 |
| LocationTimeline | Date range view | UC-TRP-011 |
| BaseLocationSettings | Settings page | UC-TRP-008 |

---

## Acceptance Criteria

### Core Trip Management
- [ ] All use case CLI tests pass
- [ ] API endpoints return correct HTTP status codes
- [ ] MCP tools tested with inspector
- [ ] UI components render correctly
- [ ] Search is case-insensitive and matches partial strings
- [ ] Delete cascades to items but not documents

### Location Awareness (MVP)
- [ ] Base locations configurable (home required, work optional)
- [ ] Default "Home" used when home-location not configured
- [ ] Trips without explicit location return "Away"
- [ ] Location query returns trip location when date is within trip
- [ ] Location query returns home location when date is not within any trip
- [ ] Multiple locations per day supported
- [ ] Date range query correctly groups consecutive days at same location
- [ ] LLM can query location via MCP tools

---

## Out of Scope

- Trip templates (recurring trips)
- Trip sharing/collaboration
- GPS coordinates or precise geographic data

**Note**: Trip organization features (merging, converting trips to items, moving items) are covered in [trip-organization.md](./trip-organization.md).
- Automatic location inference from flight/hotel items (future enhancement)
- Time-of-day location changes (morning vs evening on travel days)
- Location history/tracking beyond trip dates
- Natural language trip creation parsing
- Calendar conflict detection (requires Phase 2 calendar integration)
- Proximity calculations for "near home" distinction
