# Feature: Trip Organization

## Overview

Operations for reorganizing trips and items after initial creation. Users often need to merge duplicate trips, convert a trip into an item on another trip, or move items between trips as plans evolve.

## User Value

Travel plans change. Users may:
- Import from calendar and get duplicate trips that should be merged
- Create a "trip" that's actually a day trip within a larger vacation
- Realize items belong on a different trip
- Need to split a trip into multiple trips

Without these operations, users must delete and recreate content manually, losing history and wasting time.

---

## Use Cases

### [UC-ORG-001] Merge two existing trips

**Actor**: User

**Preconditions**:
- Two trips exist

**Steps**:
1. User selects a source trip
2. User selects "Merge with..." action
3. User selects target trip
4. System shows preview (combined items, extended dates)
5. User confirms
6. System moves all items from source to target
7. System extends target dates if source dates are outside target range
8. System deletes source trip

**Expected Result**:
- Target trip contains all items from both trips
- Target trip dates cover the full range of both trips
- Source trip is deleted
- Item IDs preserved (no duplicate creation)

**Merge Rules**:
- Target trip keeps its name (user can rename after)
- Target trip keeps its purpose (user can change after)
- Target trip keeps its status unless source has "higher" status (confirmed > planning)
- Notes from both trips are concatenated
- Locations from source trip are merged (source locations for dates not in target)

**UI Flow**:
```
Trip Detail Page
    │
    └── Action Menu (...)
            │
            └── "Merge with..."
                    │
                    ▼
            ┌─────────────────────────────┐
            │ Select trip to merge into:  │
            │                             │
            │ ○ NYC Weekend (Mar 17-19)   │ ← sorted by date proximity
            │ ○ Boston Trip (Mar 25-27)   │
            │ ○ Paris Vacation (Apr 1-10) │
            │                             │
            │ [Cancel]  [Preview]         │
            └─────────────────────────────┘
                    │
                    ▼
            ┌─────────────────────────────┐
            │ Preview: Merged Trip        │
            │                             │
            │ Name: NYC Weekend           │
            │ Dates: Mar 15-19 (extended) │
            │ Items: 6 (4 + 2)            │
            │                             │
            │ ⚠ "NYC Business" will be    │
            │   deleted after merge       │
            │                             │
            │ [Cancel]  [Confirm Merge]   │
            └─────────────────────────────┘
```

---

### [UC-ORG-002] Convert trip to item on another trip

**Actor**: User

**Preconditions**:
- Source trip exists (the trip to convert)
- Target trip exists (where the item will be added)

**Steps**:
1. User selects a trip
2. User selects "Convert to item on..." action
3. User selects target trip
4. System creates an event item on target trip with:
   - Name: source trip name
   - Date: source trip start date (or date range for multi-day)
   - Location: source trip location (if set)
   - Notes: source trip notes
5. User chooses whether to delete source trip
6. System executes conversion

**Expected Result**:
- New event item created on target trip
- Item contains source trip's key information
- Source trip optionally deleted
- Source trip's items NOT transferred (user can move them separately)

**Use Case Example**:
- "Day trip to Napa" (1-day trip) → becomes an event on "California Vacation"

**UI Flow**:
```
Trip Detail Page
    │
    └── Action Menu (...)
            │
            └── "Convert to item on..."
                    │
                    ▼
            ┌─────────────────────────────┐
            │ Add as item on:             │
            │                             │
            │ ○ California Vacation       │ ← trips with overlapping/nearby dates first
            │   (Mar 1-10)                │
            │ ○ West Coast Tour           │
            │   (Feb 20-28)               │
            │                             │
            │ ☑ Delete "Day trip to Napa" │
            │   after conversion          │
            │                             │
            │ [Cancel]  [Convert]         │
            └─────────────────────────────┘
```

**Notes**:
- Source trip's items are NOT automatically moved
- User should be prompted: "This trip has 3 items. Move them to [target] as well?"
- If user says yes, execute UC-ORG-003 for each item

---

### [UC-ORG-003] Move item to another trip

**Actor**: User

**Preconditions**:
- Item exists on source trip
- Target trip exists (or user wants to create new trip)

**Steps**:
1. User selects an item
2. User selects "Move to..." action
3. User selects target trip OR "Create new trip"
4. If creating new trip:
   - System prompts for trip name (defaults to item name/location)
   - System sets dates based on item date(s)
5. System moves item from source to target trip
6. Item's dates remain unchanged

**Expected Result**:
- Item appears on target trip
- Item removed from source trip
- Item ID preserved (not delete + create)
- Item dates unchanged

**UI Flow**:
```
Item Card (on Trip Detail)
    │
    └── Action Menu (...)
            │
            └── "Move to..."
                    │
                    ▼
            ┌─────────────────────────────┐
            │ Move "EWR → SFO" to:        │
            │                             │
            │ ○ California Trip (Mar 1-10)│
            │ ○ NYC Weekend (Mar 17-19)   │
            │ ────────────────────────────│
            │ ○ + Create new trip         │
            │                             │
            │ [Cancel]  [Move]            │
            └─────────────────────────────┘
```

**"Create new trip" flow**:
```
            ┌─────────────────────────────┐
            │ Create trip from item       │
            │                             │
            │ Name: [SFO Trip          ]  │ ← default from destination
            │ Purpose: [vacation     ▼]   │
            │ Dates: Mar 5-5              │ ← from item date
            │                             │
            │ [Cancel]  [Create & Move]   │
            └─────────────────────────────┘
```

---

### [UC-ORG-004] Create trip from item(s)

**Actor**: User

**Preconditions**:
- One or more items exist on a trip

**Steps**:
1. User selects one or more items
2. User selects "Create trip from..." action
3. System prompts for new trip details:
   - Name (defaults based on item destination/location)
   - Purpose
   - Dates (auto-calculated from item dates)
4. System creates new trip
5. System moves selected items to new trip

**Expected Result**:
- New trip created with date range covering all selected items
- All selected items moved to new trip
- Items removed from source trip
- Item IDs preserved

**Use Case Example**:
- User has "West Coast Trip" with SF leg and LA leg
- User wants to split into "SF Trip" and "LA Trip"
- User selects SF-related items → creates "SF Trip"

**UI Flow** (single item):
```
Item Card
    │
    └── Action Menu → "Create trip from..."
```

**UI Flow** (multiple items - requires multi-select):
```
Trip Detail Page
    │
    └── [Select mode]
            │
            ├── ☑ Flight: EWR → SFO
            ├── ☑ Hotel: SF Marriott
            ├── ☐ Event: LA Meeting
            └── ☐ Flight: LAX → EWR

    └── [Create trip from selected]
            │
            ▼
        ┌─────────────────────────────┐
        │ Create trip from 2 items    │
        │                             │
        │ Name: [SF Trip           ]  │
        │ Purpose: [business     ▼]   │
        │ Dates: Mar 1-3 (auto)       │
        │                             │
        │ [Cancel]  [Create]          │
        └─────────────────────────────┘
```

---

### [UC-ORG-005] Bulk move items between trips

**Actor**: User

**Preconditions**:
- Source trip has multiple items
- Target trip exists

**Steps**:
1. User enters selection mode on trip detail
2. User selects multiple items
3. User selects "Move selected to..."
4. User selects target trip
5. System moves all selected items

**Expected Result**:
- All selected items moved to target trip
- Source trip retains unselected items
- Operation is atomic (all or nothing)

**UI Flow**:
```
Trip Detail Page (selection mode)
    │
    ├── ☑ Flight: EWR → SFO
    ├── ☑ Hotel: SF Marriott
    ├── ☐ Event: Team Dinner
    │
    └── [Move 2 items to... ▼]
            │
            ├── California Trip
            ├── NYC Weekend
            └── + Create new trip
```

**Notes**:
- Selection mode can be toggled via "Select" button or long-press on mobile
- Selection persists until user exits mode or completes action

---

## API Endpoints

| Method | Endpoint | Description | Use Cases |
|--------|----------|-------------|-----------|
| POST | `/api/trips/:id/merge/:targetId` | Merge trip into target | UC-ORG-001 |
| POST | `/api/trips/:id/convert-to-item` | Convert trip to item | UC-ORG-002 |
| POST | `/api/items/:id/move` | Move item to trip | UC-ORG-003, UC-ORG-004 |
| POST | `/api/items/bulk-move` | Move multiple items | UC-ORG-005 |

### Endpoint Details

#### POST `/api/trips/:sourceId/merge/:targetId`

Merges source trip into target trip.

**Request Body**:
```json
{
  "deleteSource": true,
  "mergeNotes": true
}
```

**Response**: Updated target trip with all items

#### POST `/api/trips/:id/convert-to-item`

Converts trip to an event item on another trip.

**Request Body**:
```json
{
  "targetTripId": "uuid",
  "deleteSourceTrip": true,
  "moveItems": false
}
```

**Response**: Created item

#### POST `/api/items/:id/move`

Moves an item to another trip or creates a new trip.

**Request Body**:
```json
{
  "targetTripId": "uuid"
}
```
OR (to create new trip):
```json
{
  "newTrip": {
    "name": "SF Trip",
    "purpose": "business"
  }
}
```

**Response**: Updated item (with new tripId)

#### POST `/api/items/bulk-move`

Moves multiple items at once.

**Request Body**:
```json
{
  "itemIds": ["uuid1", "uuid2"],
  "targetTripId": "uuid"
}
```
OR:
```json
{
  "itemIds": ["uuid1", "uuid2"],
  "newTrip": {
    "name": "New Trip",
    "purpose": "vacation"
  }
}
```

**Response**: Array of updated items

---

## MCP Tools

| Tool | Description | Use Cases |
|------|-------------|-----------|
| `merge_trips` | Merge two trips | UC-ORG-001 |
| `convert_trip_to_item` | Convert trip to item | UC-ORG-002 |
| `move_item` | Move item between trips | UC-ORG-003 |
| `create_trip_from_items` | Create trip from items | UC-ORG-004 |

### Tool Definitions

#### merge_trips

```yaml
x-mcp:
  tool: merge_trips
  description: Merge one trip into another, combining all items
  parameters:
    source_trip_id: Trip to merge from (will be deleted)
    target_trip_id: Trip to merge into (will be kept)
```

#### move_item

```yaml
x-mcp:
  tool: move_item
  description: Move an item to a different trip
  parameters:
    item_id: The item to move
    target_trip_id: Destination trip (optional if creating new)
    new_trip_name: Name for new trip (if not moving to existing)
    new_trip_purpose: Purpose for new trip (if creating new)
```

---

## UI Components

| Component | Location | Use Cases |
|-----------|----------|-----------|
| TripActionMenu | Trip detail page | UC-ORG-001, UC-ORG-002 |
| ItemActionMenu | Item cards | UC-ORG-003, UC-ORG-004 |
| TripPickerModal | Modal overlay | All |
| MergePreviewModal | Modal overlay | UC-ORG-001 |
| CreateTripFromItemModal | Modal overlay | UC-ORG-004 |
| ItemSelectionMode | Trip detail page | UC-ORG-005 |
| BulkActionBar | Trip detail page | UC-ORG-005 |

### Component Specifications

#### TripActionMenu

Extends existing trip action menu with:
- "Merge with..." → opens TripPickerModal
- "Convert to item on..." → opens TripPickerModal

#### ItemActionMenu

New action menu for items:
- "Edit" (existing)
- "Move to..." → opens TripPickerModal
- "Create trip from..." → opens CreateTripFromItemModal
- "Delete" (existing)

#### TripPickerModal

Reusable modal for selecting a trip:
- Shows trips sorted by date proximity to source
- Option to create new trip inline
- Preview of what will happen (context-dependent)

---

## MVP Scope

### Phase 1: Core Operations
- [UC-ORG-001] Merge existing trips
- [UC-ORG-003] Move single item to another trip

### Phase 2: Extended Operations
- [UC-ORG-002] Convert trip to item
- [UC-ORG-004] Create trip from single item

### Phase 3: Bulk Operations
- [UC-ORG-005] Bulk move items
- Multi-select UI
- Create trip from multiple items

---

## Acceptance Criteria

### UC-ORG-001: Merge trips
- [ ] All items from source trip appear on target trip
- [ ] Target trip dates extended if needed
- [ ] Source trip deleted after merge
- [ ] Item IDs preserved (no recreation)
- [ ] Undo not required (merge is intentional)

### UC-ORG-002: Convert trip to item
- [ ] Event item created with trip name, dates, location
- [ ] Source trip optionally deleted
- [ ] Source trip items not automatically moved (user choice)

### UC-ORG-003: Move item
- [ ] Item appears on target trip
- [ ] Item removed from source trip
- [ ] Item ID preserved
- [ ] Can create new trip inline

### UC-ORG-004: Create trip from item
- [ ] New trip created with appropriate defaults
- [ ] Item moved to new trip
- [ ] Trip dates match item dates

### UC-ORG-005: Bulk move
- [ ] All selected items moved atomically
- [ ] Selection mode UI works on mobile
- [ ] Clear visual feedback during selection

---

## Out of Scope

- Undo/redo for organization operations
- Merge suggestions (auto-detect duplicates)
- Drag-and-drop between trips
- Trip templates
- Copy item (vs move)
- Merge more than 2 trips at once
- Split trip (use create-from-items + move instead)

---

## Open Questions

1. **Merge conflict resolution**: If both trips have locations for the same date, which wins?
   - Proposal: Target trip's locations take precedence, source fills gaps

2. **Convert with items**: When converting trip to item, should items be moved automatically?
   - Proposal: Prompt user, don't auto-move

3. **Empty trip handling**: After moving all items out, should empty trip be deleted?
   - Proposal: No auto-delete, user decides

4. **Date adjustment**: When moving item to trip outside the trip's date range, extend trip dates?
   - Proposal: Yes, with confirmation

---

## Related Use Cases

- [UC-CAL-004] Merge imported trip suggestion with existing trip (calendar-trip-intelligence.md)
- [UC-TRP-001] Create a basic trip (trip-management.md)
- [UC-TRP-005] Delete trip and all items (trip-management.md)
