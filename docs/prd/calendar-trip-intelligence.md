# Feature: Calendar Trip Intelligence

## Overview

Enhance the calendar-to-trip import flow with intelligent event classification, TripIt integration, trip merging, and nested trip support. This feature transforms raw calendar events into well-organized trips while avoiding duplicates and maintaining relationships between related trips.

## User Value

Frequent travelers use multiple tools (Google Calendar, TripIt, airline apps) that all write to their calendar. Currently, importing a trip suggestion creates an empty trip shell. This feature:

1. **Imports travel items** - Flights, hotels, and events from calendar become trip items, not just trip metadata
2. **Prevents duplicates** - Merging logic avoids creating redundant trips when importing overlapping date ranges
3. **Handles TripIt** - Parses TripIt's specific calendar format to extract flight details and trip summaries
4. **Distinguishes trips vs items** - All-day events suggest trips; timed events suggest items within trips
5. **Supports complex itineraries** - Nested trips (Paris -> Brussels -> Paris -> Dakhla) become linked related trips

---

## Use Cases

### [UC-CAL-001] Import trip with travel items

**Actor**: User

**Preconditions**:
- Google Calendar connected
- Calendar contains trip-related events (flights, hotels, etc.)
- Trip suggestion generated from calendar

**Steps**:
1. User views trip suggestions
2. User clicks "Import" on a suggestion
3. System creates trip from suggestion
4. System converts source calendar events to trip items

**Expected Result**:
- Trip created with name, dates, location
- Flight events become flight items with from/to/carrier/flight number
- Hotel events become hotel items with name, location, dates
- Timed events become event items with time and location
- Calendar event IDs linked to trip items for deduplication

**CLI Test**:
```bash
# Setup: Requires calendar with travel events
# This test assumes a trip suggestion exists

# Action - import the suggestion
TRIP=$(travel calendar import-suggestion $SUGGESTION_ID --json)

# Verify
echo $TRIP | jq -e '.items | length > 0'
echo $TRIP | jq -e '.items[] | select(.type == "flight") | .from != null'

# Cleanup
travel trips delete $(echo $TRIP | jq -r '.id')
```

---

### [UC-CAL-002] Parse TripIt all-day summary event

**Actor**: System

**Preconditions**:
- TripIt calendar synced to Google Calendar
- TripIt all-day event format: "michaelwinser is in Brussels, Belgium from Jan 23 to Feb 3, 2026"
- Description contains TripIt link

**Steps**:
1. System encounters TripIt all-day summary event
2. System parses location from event title
3. System extracts date range from event title
4. System creates trip suggestion with TripIt source indicator

**Expected Result**:
- Location extracted: "Brussels, Belgium"
- Date range extracted: Jan 23 - Feb 3, 2026
- Source marked as "tripit" for deduplication
- TripIt link preserved in trip notes

**Parsing Examples**:
```
Input: "michaelwinser is in Brussels, Belgium from Jan 23 to Feb 3, 2026"
Output: { location: "Brussels, Belgium", start: "2026-01-23", end: "2026-02-03" }

Input: "michaelwinser is in New York, NY, USA from Mar 1 to Mar 5, 2026"
Output: { location: "New York, NY, USA", start: "2026-03-01", end: "2026-03-05" }
```

---

### [UC-CAL-003] Parse TripIt flight segment event

**Actor**: System

**Preconditions**:
- TripIt calendar synced to Google Calendar
- TripIt flight event format: "UA57 EWR to CDG"
- Description contains flight details (times, terminals, confirmation)

**Steps**:
1. System encounters TripIt flight segment event
2. System parses carrier and flight number from title
3. System extracts origin/destination airports from title
4. System parses departure/arrival times from description
5. System extracts terminal and confirmation from description

**Expected Result**:
- Flight item created with:
  - carrier: "UA"
  - flight_number: "57"
  - from: "EWR"
  - to: "CDG"
  - date: from event start
  - time: from event start time
  - confirmation: from description if present
  - notes: terminal info, full description preserved

**Parsing Examples**:
```
Input Title: "UA57 EWR to CDG"
Input Description: "Departs: 7:30 PM Terminal B\nArrives: 8:45 AM +1 Terminal 1\nConfirmation: ABC123"

Output: {
  type: "flight",
  carrier: "UA",
  flight_number: "57",
  from: "EWR",
  to: "CDG",
  confirmation: "ABC123",
  notes: "Terminal B -> Terminal 1"
}
```

---

### [UC-CAL-004] Merge imported trip with existing trip

**Actor**: User

**Preconditions**:
- Existing trip in the system
- Trip suggestion has same/similar location and overlapping dates

**Steps**:
1. System detects potential merge candidate
2. System presents merge option to user
3. User confirms merge
4. System adds new items to existing trip
5. System extends trip dates if needed

**Expected Result**:
- No duplicate trip created
- New items added to existing trip
- Trip dates expanded to cover all items
- Calendar event IDs added to trip for future deduplication
- Merge history recorded in trip notes

**Merge Criteria**:
- Same location (fuzzy match on city name)
- Dates overlap OR within 2 days of each other
- User confirmation required (no automatic merge)

**CLI Test**:
```bash
# Setup - create existing trip
TRIP_ID=$(travel trips create --name "Brussels Conference" --purpose conference \
  --start 2026-01-29 --end 2026-02-02 --location "Brussels, Belgium" --json | jq -r '.id')

# Action - import suggestion that should merge
# (assumes suggestion exists with similar dates/location)
RESULT=$(travel calendar import-suggestion $SUGGESTION_ID --merge-with $TRIP_ID --json)

# Verify - should return existing trip with new items
echo $RESULT | jq -e '.id == "'$TRIP_ID'"'
echo $RESULT | jq -e '.items | length > 0'

# Cleanup
travel trips delete $TRIP_ID
```

---

### [UC-CAL-005] Detect merge candidates for trip suggestion

**Actor**: System

**Preconditions**:
- Trip suggestions generated
- Existing trips in the system

**Steps**:
1. User requests trip suggestions
2. System generates suggestions from calendar
3. System checks each suggestion against existing trips
4. System annotates suggestions with merge candidates

**Expected Result**:
- Suggestions include `mergeCandidates` array
- Each candidate includes trip ID, name, match reason
- UI can present "Import" vs "Merge with existing" options

**API Response Enhancement**:
```json
{
  "suggestions": [
    {
      "id": "abc123",
      "name": "Brussels Trip",
      "location": "Brussels, Belgium",
      "startDate": "2026-01-29",
      "endDate": "2026-02-02",
      "mergeCandidates": [
        {
          "tripId": "trip-uuid",
          "tripName": "FOSDEM 2026",
          "matchReason": "Same location, overlapping dates"
        }
      ]
    }
  ]
}
```

---

### [UC-CAL-006] Distinguish all-day events (trips) from timed events (items)

**Actor**: System

**Preconditions**:
- Calendar contains mix of all-day and timed events

**Steps**:
1. System analyzes calendar events
2. All-day events with locations suggest entire trips
3. Timed events with duration suggest trip items
4. System groups items under appropriate trip suggestions

**Expected Result**:
- All-day events become trip suggestions (top-level containers)
- Timed events with short duration (< 24h) become item suggestions
- Items are associated with the nearest trip by date/location
- Standalone items (no parent trip) prompt "Create trip for this item?"

**Classification Rules**:
| Event Type | Duration | Has Location | Result |
|------------|----------|--------------|--------|
| All-day | 1+ days | Yes | Trip suggestion |
| All-day | 1+ days | No | Skip (not travel) |
| Timed | < 24h | Yes | Item suggestion |
| Timed | Multi-day | Yes | Trip suggestion |
| TripIt summary | All-day | Parsed | Trip suggestion |
| TripIt flight | Timed | Parsed | Flight item |

---

### [UC-CAL-007] Create nested/related trips from complex itinerary

**Actor**: User

**Preconditions**:
- Calendar shows complex itinerary:
  - Fly NYC -> Paris (Jan 15)
  - Paris events (Jan 15-18)
  - Train Paris -> Brussels (Jan 18)
  - Brussels events (Jan 18-20)
  - Train Brussels -> Paris (Jan 20)
  - Paris events (Jan 20-22)
  - Fly Paris -> Dakhla (Jan 22)
  - Dakhla events (Jan 22-26)
  - Fly Dakhla -> Paris -> NYC (Jan 26)

**Steps**:
1. System detects multiple distinct locations
2. System groups events by location
3. System creates related trip suggestions with links
4. User imports trips individually or as a group
5. System maintains bidirectional links between trips

**Expected Result**:
- Paris Trip 1: Jan 15-18 (linked to: Brussels Trip)
- Brussels Trip: Jan 18-20 (linked to: Paris Trip 1, Paris Trip 2)
- Paris Trip 2: Jan 20-22 (linked to: Brussels Trip, Dakhla Trip)
- Dakhla Trip: Jan 22-26 (linked to: Paris Trip 2)
- Each trip shows "Related Trips" section with links
- User can optionally merge Paris 1 + Paris 2 into single trip

**Data Model**:
```
Trip
├── relatedTrips: [
│     { tripId: "uuid", relationship: "before" | "after" | "during" }
│   ]
└── ...
```

---

### [UC-CAL-008] Show related trips in UI

**Actor**: User

**Preconditions**:
- Trips exist with relatedTrips links

**Steps**:
1. User views trip detail
2. UI shows "Related Trips" section
3. Related trips displayed with dates and relationship type
4. User can click to navigate to related trip

**Expected Result**:
- Related trips shown as clickable cards
- Relationship context: "Before this trip" / "After this trip"
- Visual timeline showing the overall journey

**UI Wireframe**:
```
┌─────────────────────────────────────────────┐
│ Paris Trip 1                                │
│ Jan 15-18, 2026                             │
├─────────────────────────────────────────────┤
│ [Items timeline...]                         │
├─────────────────────────────────────────────┤
│ Related Trips                               │
│ ┌─────────────────────────────────────────┐ │
│ │ → Brussels Trip (Jan 18-20)             │ │
│ │   Continues to Brussels via train       │ │
│ └─────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

---

### [UC-CAL-009] Merge two existing trips

**Actor**: User

**Preconditions**:
- Two trips exist that should be one
- Trips have same/similar location

**Steps**:
1. User views trip detail
2. User clicks "Merge with another trip"
3. System shows eligible trips (same location, nearby dates)
4. User selects target trip
5. System merges items from both trips
6. System adjusts dates to cover both
7. Source trip is deleted

**Expected Result**:
- Items from both trips combined
- Date range covers both original ranges
- Source trip deleted after merge
- Merge recorded in notes for audit

**CLI Test**:
```bash
# Setup - create two trips that should be merged
TRIP1=$(travel trips create --name "Paris Day 1-2" --purpose vacation \
  --start 2026-01-15 --end 2026-01-17 --location "Paris" --json | jq -r '.id')
TRIP2=$(travel trips create --name "Paris Day 3-4" --purpose vacation \
  --start 2026-01-17 --end 2026-01-19 --location "Paris" --json | jq -r '.id')

# Add items to each
travel items add $TRIP1 event --name "Eiffel Tower" --date 2026-01-16
travel items add $TRIP2 event --name "Louvre" --date 2026-01-18

# Action - merge trip2 into trip1
MERGED=$(travel trips merge $TRIP1 $TRIP2 --json)

# Verify
echo $MERGED | jq -e '.id == "'$TRIP1'"'
echo $MERGED | jq -e '.startDate == "2026-01-15"'
echo $MERGED | jq -e '.endDate == "2026-01-19"'
echo $MERGED | jq -e '.items | length == 2'

# Verify trip2 no longer exists
! travel trips get $TRIP2 2>/dev/null

# Cleanup
travel trips delete $TRIP1
```

---

### [UC-CAL-010] Filter virtual meetings from suggestions (COMPLETED)

**Actor**: System

**Preconditions**:
- Calendar contains virtual meeting events
- Events have URL locations (Zoom, Teams, Meet) or meeting room names

**Steps**:
1. System fetches calendar events
2. System filters out events with URL locations
3. System filters out events with meeting room resource names
4. Only physical travel events remain for suggestions

**Expected Result**:
- Events with `https://zoom.us/...` locations excluded
- Events with `https://meet.google.com/...` locations excluded
- Events with `https://teams.microsoft.com/...` locations excluded
- Events with meeting room names like "US-NYC-42W-3-A-Maple (8) [GVC]" excluded
- Only events with physical location addresses remain

**Status**: COMPLETED (Issue #28)

---

### [UC-CAL-011] Remember processed calendar events

**Actor**: System

**Preconditions**:
- User imports or dismisses trip suggestions
- Calendar events are associated with suggestions

**Steps**:
1. User imports a trip suggestion OR
2. User dismisses a trip suggestion
3. System records all source calendar event IDs as processed
4. Next time suggestions are fetched, processed events are excluded

**Expected Result**:
- Imported events don't reappear as new suggestions
- Dismissed events don't reappear
- User can "reset" processed events if needed
- Events processed as part of a merge are tracked

**Database Schema**:
```sql
CREATE TABLE processed_calendar_events (
    id TEXT PRIMARY KEY,
    calendar_event_id TEXT NOT NULL,
    calendar_id TEXT NOT NULL,
    action TEXT NOT NULL,  -- 'imported', 'dismissed', 'merged'
    trip_id TEXT,          -- which trip received this event (if imported/merged)
    item_id TEXT,          -- which item was created (if applicable)
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(calendar_id, calendar_event_id)
);
```

**API Additions**:
- `POST /api/calendar/suggestions/:id/dismiss` - Mark suggestion as dismissed
- Query param on suggestions: `?includeProcessed=true` to show all (for debugging)

**UI Additions**:
- "Dismiss" button on suggestion cards
- Visual indicator for "previously imported" if re-shown
- Settings option to "Reset processed events"

**Rationale**:
TripIt creates multiple calendar events for the same trip (summary + flight segments).
After importing the summary, the flight segments still appear as suggestions, creating noise.
Tracking processed events eliminates this redundancy.

---

## API Endpoints

| Method | Endpoint | Description | Use Cases |
|--------|----------|-------------|-----------|
| GET | `/api/calendar/suggestions` | Get trip suggestions with merge candidates | UC-CAL-005, UC-CAL-006 |
| POST | `/api/calendar/suggestions/:id/import` | Import suggestion as new trip with items | UC-CAL-001 |
| POST | `/api/calendar/suggestions/:id/merge/:tripId` | Merge suggestion into existing trip | UC-CAL-004 |
| POST | `/api/calendar/suggestions/:id/dismiss` | Mark suggestion as dismissed | UC-CAL-011 |
| DELETE | `/api/calendar/processed-events` | Reset all processed events | UC-CAL-011 |
| POST | `/api/trips/:id/merge/:sourceId` | Merge two existing trips | UC-CAL-009 |
| GET | `/api/trips/:id/related` | Get related trips | UC-CAL-008 |
| PUT | `/api/trips/:id/related` | Set related trips | UC-CAL-007 |

---

## MCP Tools

| Tool | Description | Use Cases |
|------|-------------|-----------|
| `suggest_trips_from_calendar` | Enhanced with merge candidates, item suggestions | UC-CAL-005, UC-CAL-006 |
| `import_trip_suggestion` | Import with items, optional merge target | UC-CAL-001, UC-CAL-004 |
| `merge_trips` | Merge two existing trips | UC-CAL-009 |
| `get_related_trips` | Get trips linked to a given trip | UC-CAL-008 |

---

## UI Components

| Component | Views | Use Cases |
|-----------|-------|-----------|
| `TripSuggestionCard` | Calendar suggestions page | UC-CAL-001, UC-CAL-005 |
| `MergeCandidateBadge` | Suggestion card | UC-CAL-005 |
| `ImportModal` | Suggestion import flow | UC-CAL-001, UC-CAL-004 |
| `RelatedTripsSection` | Trip detail page | UC-CAL-008 |
| `MergeTripModal` | Trip detail page | UC-CAL-009 |
| `ItemPreviewList` | Import modal | UC-CAL-001, UC-CAL-003 |

---

## Data Model Changes

### Trip Entity Additions

```go
type Trip struct {
    // ... existing fields ...

    // Related trips for complex itineraries
    RelatedTrips []RelatedTrip `json:"relatedTrips,omitempty"`

    // Calendar event IDs that were imported into this trip
    SourceCalendarEventIds []string `json:"sourceCalendarEventIds,omitempty"`
}

type RelatedTrip struct {
    TripID       string `json:"tripId"`
    Relationship string `json:"relationship"` // "before", "after", "during"
}
```

### TripSuggestion Entity Additions

```go
type TripSuggestion struct {
    // ... existing fields ...

    // Potential items to import
    SuggestedItems []SuggestedItem `json:"suggestedItems,omitempty"`

    // Existing trips that could be merged
    MergeCandidates []MergeCandidate `json:"mergeCandidates,omitempty"`

    // Source identifier for deduplication
    Source string `json:"source"` // "google", "tripit"
}

type SuggestedItem struct {
    Type         string `json:"type"` // "flight", "hotel", "train", "event"
    CalendarEvent CalendarEvent `json:"calendarEvent"`
    ParsedData   map[string]interface{} `json:"parsedData"`
}

type MergeCandidate struct {
    TripID      string `json:"tripId"`
    TripName    string `json:"tripName"`
    MatchReason string `json:"matchReason"`
}
```

---

## Acceptance Criteria

### Import with Items
- [ ] TripIt flight events parsed into flight items
- [ ] TripIt summary events parsed for location/dates
- [ ] Generic calendar events with location become event items
- [ ] Hotel events (containing "hotel", "checkin") become hotel items
- [ ] Train events (containing "train", "eurostar", "amtrak") become train items
- [ ] Imported items preserve calendar event ID for deduplication

### Merge Logic
- [ ] Suggestions include merge candidates when similar trips exist
- [ ] Fuzzy location matching (Paris == Paris, France)
- [ ] Date proximity check (overlap OR within 2 days)
- [ ] User confirmation required before merge
- [ ] Merge combines items and extends dates
- [ ] Source trip deleted after merge

### Nested Trips
- [ ] Multi-location itineraries create linked trip suggestions
- [ ] Related trips stored and queryable
- [ ] UI shows related trips section
- [ ] Navigation between related trips works

### Event Classification
- [ ] All-day events with location -> trip suggestions
- [ ] Timed events -> item suggestions
- [ ] TripIt-specific parsing for both formats
- [ ] Virtual meeting filtering (already implemented)

---

## Out of Scope

- Automatic merge without user confirmation
- Bi-directional sync (writing back to Google Calendar)
- Real-time calendar sync (webhook-based updates)
- Supporting calendar providers other than Google
- Supporting travel booking services other than TripIt
- GPS-based location resolution
- Automatic trip purpose detection
- Multi-user/shared trip merging

---

## MVP Scope

### Phase 2B MVP (COMPLETED)
- [UC-CAL-001] Import trip with travel items ✅
- [UC-CAL-002] Parse TripIt all-day summary event ✅
- [UC-CAL-003] Parse TripIt flight segment event ✅
- [UC-CAL-005] Detect merge candidates for suggestions ✅
- [UC-CAL-006] Distinguish all-day events vs timed events ✅
- [UC-CAL-010] Filter virtual meetings ✅

### Phase 2B.1: Smart Merging & Event Memory
- [UC-CAL-004] Merge imported trip with existing trip
- [UC-CAL-011] Remember processed calendar events (NEW)

### Deferred to Later (Phase 2C)
- [UC-CAL-007] Create nested/related trips
- [UC-CAL-008] Show related trips in UI
- [UC-CAL-009] Merge two existing trips

---

## Lessons Learned (Phase 2B MVP)

### What Worked Well
1. **TripIt parsing** - Successfully extracts flight details from TripIt calendar events
2. **Event classification** - Correctly distinguishes trips vs items, filters virtual meetings
3. **Merge candidate detection** - Identifies overlapping trips with fuzzy location matching
4. **Item preview UI** - Shows users what will be created before import

### Pain Points Discovered
1. **Redundant suggestions** - TripIt creates multiple events for the same trip (summary + segments). After importing the summary, flight segments still appear as new suggestions. This creates noise and confusion.

2. **No event memory** - The system doesn't remember which events have been imported or dismissed. Users see the same suggestions repeatedly.

3. **Merge is informational only** - Phase 2B shows "Similar trip exists" but doesn't let users merge. Users must import a duplicate and then manually manage it.

### Priority Adjustments
Based on real-world usage, **Phase 2B.1 priorities** are:
1. **UC-CAL-011 (Event memory)** - Highest impact for reducing noise
2. **UC-CAL-004 (Merge into existing)** - Enables proper workflow when similar trips exist

**Phase 2C (Related trips)** remains lower priority - useful for complex itineraries but not blocking daily use.

---

### MVP Rationale

The MVP focuses on **getting data into the system correctly**:
1. Parse TripIt events into structured items
2. Classify events as trips vs items
3. Show merge candidates (information only)

The deferred features involve **managing data relationships**:
1. Actually performing merges requires careful UX
2. Related trips need UI investment
3. These can be added once the import quality is proven

### MVP User Journey

```
Calendar Events               Trip Suggestions              Travel Calendar
─────────────────            ─────────────────            ─────────────────
TripIt: "michaelwinser       ┌─────────────────┐
is in Brussels..."      ───► │ Brussels Trip   │
                             │ Jan 23 - Feb 3  │
TripIt: "UA57 EWR→BRU"  ───► │                 │ ─Import─► Trip: Brussels
                             │ Items:          │          ├─ Flight: UA57
TripIt: "DL123 BRU→EWR" ───► │ - Flight UA57   │          ├─ Flight: DL123
                             │ - Flight DL123  │          └─ (location set)
                             │                 │
                             │ Similar: FOSDEM │          [User can manually
                             │ 2026 trip exists│           merge later]
                             └─────────────────┘
```

---

## Roadmap Proposal

### Current State (Phase 2A - DONE)
- Google Calendar OAuth complete
- Basic trip suggestions from calendar events
- Virtual meeting filtering implemented (#28)

### Proposed Phase 2B: Calendar Trip Intelligence MVP
| Feature | Status | Issues |
|---------|--------|--------|
| Import trips with travel items | Not Started | #25 |
| TripIt event parsing | Not Started | #27 |
| Event type classification | Not Started | #29 |
| Merge candidate detection | Not Started | #26 (partial) |

### Proposed Phase 2C: Trip Relationships
| Feature | Status | Issues |
|---------|--------|--------|
| Merge suggestion into existing trip | Not Started | #26 |
| Merge two existing trips | Not Started | #30 |
| Nested/related trips | Not Started | #30 |
| Related trips UI | Not Started | #30 |

### Dependencies
- Phase 2B depends on: Phase 2A (Google Calendar OAuth) - DONE
- Phase 2C depends on: Phase 2B (items import working correctly)

### Estimated Effort
- Phase 2B: Medium (parsing logic, API enhancement, basic UI)
- Phase 2C: Medium-Large (relationship model, merge logic, UI components)

---

## Technical Notes

### TripIt Event Detection

TripIt events can be identified by:
1. Event title pattern: `{username} is in {location} from {date} to {date}`
2. Event description containing TripIt link (`tripit.com`)
3. Flight segment pattern: `{carrier}{number} {origin} to {destination}`

```go
// TripIt all-day summary pattern
var tripItSummaryPattern = regexp.MustCompile(
    `(?i)^(\w+) is in (.+) from (\w+ \d+) to (\w+ \d+), (\d{4})$`)

// TripIt flight segment pattern
var tripItFlightPattern = regexp.MustCompile(
    `^([A-Z]{2})(\d+)\s+([A-Z]{3})\s+to\s+([A-Z]{3})$`)
```

### Location Fuzzy Matching

For merge candidate detection:
```go
func locationsMatch(loc1, loc2 string) bool {
    // Normalize: lowercase, remove punctuation
    // Extract city name (first part before comma)
    // Compare normalized city names
}

// Examples:
// "Paris" == "Paris, France" -> true
// "NYC" == "New York, NY, USA" -> true (alias map)
// "Brussels" == "London" -> false
```

### Calendar Event ID Tracking

Store imported event IDs to prevent re-import:
```sql
CREATE TABLE calendar_links (
    id TEXT PRIMARY KEY,
    trip_id TEXT REFERENCES trips(id),
    item_id TEXT REFERENCES items(id),
    calendar_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    source TEXT DEFAULT 'google',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(calendar_id, event_id)
);
```
