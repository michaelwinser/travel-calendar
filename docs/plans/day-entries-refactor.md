# Plan: Day Entries as the Primitive

## Vision

Day entries are the atomic unit. A day entry is: **date + location + optional description**. Trips are optional groupings of days. Users can mark "dentist in Fairfield" on a Tuesday without creating a trip, or create "Milan Apr 5-9" which fills 5 day entries and wraps them in a trip.

## Current State

- **Trip** has a `Location` field (primary location label, metadata only)
- **TripLocation** stores per-day locations within a trip (requires tripID)
- **Base locations** (home/work) fill in gaps when no trip exists
- Day-level locations cannot exist without a trip
- Calendar views show trip bars only — no day-level location data visible

## Target State

- **DayEntry** is a standalone entity: date + location + description + optional tripID
- Day entries can exist with or without a trip
- Trips are groupings — they have a name, date range, purpose/status, but no primary location
- Trip creation auto-generates day entries from the trip's date range + location
- Calendar views show both trip bars AND day-level location/description text
- Click-to-add on any day in both calendar views

## Data Model Changes

### New: DayEntry entity (replaces TripLocation)

```
DayEntry {
  ID          uuid.UUID   - Primary key
  UserID      string      - User ownership
  Date        time.Time   - The date (one entry per date+location+user)
  Location    string      - Where (e.g., "Milan", "Fairfield")
  Description *string     - What (e.g., "dentist", "EF Security Offsite")
  TripID      *uuid.UUID  - Optional FK to Trip (nil = standalone day entry)
  CreatedAt   time.Time
}
```

**Key differences from TripLocation:**
- `UserID` field (multi-tenant)
- `Description` field (optional context)
- `TripID` is nullable (standalone entries work without a trip)
- One-to-one mapping: each row = one location on one day (no arrays)

### Modified: Trip entity

Remove `Location` field. A trip's location is derived from its day entries.

```
Trip {
  ID        uuid.UUID
  UserID    string
  Name      string
  Purpose   string
  StartDate *time.Time
  EndDate   *time.Time
  Status    string
  Notes     *string
  CreatedAt time.Time
  UpdatedAt time.Time
}
```

### Keep unchanged
- Item, Document, CalendarLink, ProcessedCalendarEvent, Session, GoogleCredentials, UserCalendar

---

## API Changes

### New endpoints

**`GET /api/days`** — List day entries for a date range
- Query params: `from` (date, required), `to` (date, required)
- Returns: `DayEntry[]`
- Includes both standalone and trip-associated entries

**`POST /api/days`** — Create a day entry
- Body: `{ date, location, description?, tripId? }`
- Returns: `DayEntry`

**`PUT /api/days/{id}`** — Update a day entry
- Body: `{ location?, description?, tripId? }`
- Returns: `DayEntry`

**`DELETE /api/days/{id}`** — Delete a day entry

**`POST /api/days/parse`** — Parse free text into a day entry (optional, could be client-only)
- Body: `{ text, date }`
- Returns: `{ location, description }`

### Modified endpoints

**`POST /api/trips`** — CreateTripRequest
- Remove `location` field
- Add optional `location` that, if provided, auto-creates day entries for all dates in range

**`GET /api/trips/{tripId}`** — GetTrip response
- Replace `locations: TripDayLocation[]` with `dayEntries: DayEntry[]`

**`PUT /api/trips/{tripId}/locations`** → **Deprecated/removed**
- Day entries are managed via `/api/days` instead

**`GET /api/location/on/{date}`** — GetLocationOnDate
- Now queries DayEntry table instead of TripLocation
- Falls back to base location if no entries

**`GET /api/location/range`** — GetLocationRange
- Same logic, reads from DayEntry table

### Removed endpoints
- `GET /api/trips/{tripId}/locations` — replaced by filtering `/api/days?from=X&to=Y`
- `PUT /api/trips/{tripId}/locations` — replaced by individual `/api/days` CRUD

---

## Store Changes

### New methods
```go
ListDayEntries(userID string, from, to time.Time) ([]entity.DayEntry, error)
GetDayEntry(userID string, id uuid.UUID) (*entity.DayEntry, error)
CreateDayEntry(entry *entity.DayEntry) error
UpdateDayEntry(userID string, entry *entity.DayEntry) error
DeleteDayEntry(userID string, id uuid.UUID) error
GetDayEntriesForTrip(userID string, tripID uuid.UUID) ([]entity.DayEntry, error)
DeleteDayEntriesByTrip(tripID uuid.UUID) error
```

### Modified methods
- `GetTripsForDateRange` — unchanged (still queries trips table)
- Remove: `GetTripLocations`, `SetTripLocations`, `GetTripLocationsForDateRange`

### Migration
- SQLite: Create `day_entries` table, migrate data from `trip_locations` (add UserID from parent trip, description = NULL)
- Firestore: Create `dayEntries` collection, migrate from `tripLocations` documents
- Drop `trip_locations` table/collection after migration
- Remove `location` column from `trips` table

---

## Service Changes

### New: DayEntryService (or methods on existing Service)
- `ListDayEntries(userID, from, to)` — returns day entries
- `CreateDayEntry(userID, date, location, description, tripID)` — creates entry
- `UpdateDayEntry(userID, id, ...)` — updates entry
- `DeleteDayEntry(userID, id)` — deletes entry
- `CreateDayEntriesForTrip(userID, tripID, location, startDate, endDate)` — bulk create for new trip

### Modified
- `CreateTrip` — if location provided, call `CreateDayEntriesForTrip`
- `DeleteTrip` — cascade delete associated day entries
- `GetLocationOnDate` — query DayEntry instead of TripLocation
- `GetLocationRange` — query DayEntry instead of TripLocation
- `MergeTrips` — reassign day entries from source to target trip

### Remove
- `GetTripLocations`, `SetTripLocations` (replaced by DayEntry operations)

---

## Frontend Changes

### Calendar views (MonthRow + MonthGrid)

**Show day-level data on calendar:**
- Below each day number (or as a subtle label), show the location text
- In MonthGrid: location text inside the day cell, below the number
- In MonthRow: tooltip on hover showing day's location (space is tight)

**Click-to-add on days:**
- MonthGrid: click empty day → popup with quick-entry-style input
- MonthRow: click day number → same popup
- Popup: text input "dentist in Fairfield" → parser extracts location + description
- Date is implicit from the clicked day
- Save creates a DayEntry via API

### New components

**DayEntryPopup.svelte** — small popup for adding/editing a day entry
- Appears anchored to the clicked day cell
- Quick-entry text input with parser
- Shows existing entries for that day (editable, deletable)
- Close on Escape or click-outside

### Modified components

**TripForm.svelte**
- Remove `location` field (trips don't have primary locations)
- Remove the notes workaround for location
- Keep trip name, dates, purpose, status, notes

**QuickEntry.svelte**
- When creating a trip, auto-create day entries for the date range
- The `location` from parsing becomes the default day entry location

**LocationEditor.svelte** — may be removed or simplified
- Currently edits TripLocation per day within a trip
- Could be replaced by the DayEntryPopup + inline editing on calendar

### New store

**dayEntries store**
- `load(from, to)` — fetch day entries for visible range
- `create(entry)` — add entry
- `update(id, changes)` — edit entry
- `delete(id)` — remove entry
- Reactive — calendar views subscribe to this

### API client additions
```typescript
api.days.list(from, to): Promise<DayEntry[]>
api.days.create(input): Promise<DayEntry>
api.days.update(id, input): Promise<DayEntry>
api.days.delete(id): Promise<void>
```

---

## Implementation Order

```
Phase A: Backend data model
  1. Create DayEntry entity
  2. Add store methods (SQLite + Firestore)
  3. Add API endpoints (OpenAPI spec → codegen → handlers)
  4. Migrate trip creation to auto-generate day entries
  5. Update location query service to use DayEntry

Phase B: Frontend day entries
  1. Add dayEntries store + API client methods
  2. Create DayEntryPopup component
  3. Add click-to-add to MonthGrid (day cells)
  4. Add click-to-add to MonthRow (day numbers)
  5. Show location text on calendar days

Phase C: Cleanup
  1. Remove Trip.Location field
  2. Remove TripLocation entity and store methods
  3. Remove /api/trips/{id}/locations endpoints
  4. Update TripForm (remove location field)
  5. Update QuickEntry to create day entries
  6. Simplify or remove LocationEditor
  7. Regenerate shared types
```

Phase A and B can partially overlap. Phase C should come last after everything works with the new model.

---

## Risks & Considerations

- **Existing trip_locations data**: Needs migration to day_entries. Since we said "start fresh," this is low risk.
- **Performance**: Day entries could be numerous. Need indexes on (user_id, date) and pagination for large ranges.
- **Multiple entries per day**: A day can have multiple entries (e.g., travel day: "London" morning, "Brussels" afternoon). The model supports this naturally since each is a separate row.
- **Trip date changes**: If a trip's dates are extended, should day entries auto-extend? Probably not — the user should add locations for new days manually.
- **Orphan day entries**: If a trip is deleted, its day entries could become standalone (preserving location data) or be cascade-deleted. Cascade delete is simpler and matches user expectation.
- **Calendar view performance**: Loading day entries for the visible date range on every scroll could be chatty. Consider loading in chunks (e.g., 3 months at a time) and caching.
