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

## API Endpoints

| Method | Endpoint | Description | Use Cases |
|--------|----------|-------------|-----------|
| POST | `/api/trips` | Create trip | UC-TRP-001 |
| GET | `/api/trips` | List trips (with filters) | UC-TRP-002 |
| GET | `/api/trips/:id` | Get trip with items | UC-TRP-003 |
| PATCH | `/api/trips/:id` | Update trip | UC-TRP-004 |
| DELETE | `/api/trips/:id` | Delete trip | UC-TRP-005 |
| GET | `/api/trips/search` | Search trips | UC-TRP-006 |

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

---

## UI Components

| Component | Views | Use Cases |
|-----------|-------|-----------|
| TripCard | List item | UC-TRP-002 |
| TripChip | Inline reference | - |
| TripDetail | Full page | UC-TRP-003 |
| TripForm | Create/edit | UC-TRP-001, UC-TRP-004 |
| TripCalendarBar | Calendar view | UC-TRP-002 |

---

## Acceptance Criteria

- [ ] All use case CLI tests pass
- [ ] API endpoints return correct HTTP status codes
- [ ] MCP tools tested with inspector
- [ ] UI components render correctly
- [ ] Search is case-insensitive and matches partial strings
- [ ] Delete cascades to items but not documents

---

## Out of Scope

- Trip templates (recurring trips)
- Trip sharing/collaboration
- Trip merging
