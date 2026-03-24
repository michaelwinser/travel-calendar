# Trips: Design Document

## Overview

A **Trip** is a lightweight grouping concept that connects related activities into a single journey. It reflects how users think: "my FOSDEM trip" encompasses flights, hotels, the conference itself, and free days — but the user plans and reviews at the trip level.

## Principles

1. **Trips are metadata, not containers.** Activities are always visible in all views. Trips add visual context on top — they never hide anything.
2. **Activities are the atoms.** A trip's date range, locations, and conflicts are all derived from its activities. No duplicate data on the trip entity.
3. **Trips are optional.** Standalone activities (no trip) remain first-class. Not everything needs a trip. A dentist appointment doesn't need a container.
4. **One trip per activity.** An activity belongs to at most one trip. Trips are journeys, not tags. (If an activity spans two trips, it's probably two activities.)

## Data Model

### Trip entity

```
Trip:
  id          string       (UUID)
  userId      string       (owner)
  name        string       (e.g., "FOSDEM 2027", "Tokyo client visit")
  createdAt   datetime
```

No stored dates, locations, or other derived fields. Everything is computed from the trip's activities.

### Activity changes

Add one nullable field:

```
Activity:
  ...existing fields...
  tripId      string?      (FK to Trip, nullable)
```

### Derived properties (computed, not stored)

```
Trip (computed):
  startDate   = min(activities.startDate)
  endDate     = max(activities.endDate)
  locations   = unique(activities.location)
  types       = unique(activities.type)
  duration    = endDate - startDate + 1 day
```

## API

### Trip CRUD

```
POST   /api/trips                    → create trip (name)
GET    /api/trips                    → list user's trips (with computed fields)
GET    /api/trips/{id}               → get trip with its activities
PUT    /api/trips/{id}               → update trip (name)
DELETE /api/trips/{id}               → delete trip (unlinks activities, doesn't delete them)
```

### Activity assignment

```
PUT    /api/activities/{id}          → existing endpoint, add tripId field
```

Assigning an activity to a trip is just an update: `PUT /api/activities/{id} { tripId: "..." }`. Removing from a trip: `PUT /api/activities/{id} { tripId: null }`.

### List trips response

```json
{
  "id": "abc123",
  "name": "FOSDEM 2027",
  "startDate": "2027-01-30",
  "endDate": "2027-02-04",
  "locations": ["Brussels", "Paris CDG"],
  "activityCount": 4,
  "createdAt": "2026-03-24T..."
}
```

The `startDate` and `endDate` are computed server-side from the trip's activities. The list endpoint returns these so the frontend doesn't have to compute them.

## Visual Treatment

### Month view

- A **subtle background band** spans the trip's date range, rendered behind (below) the activity bars.
- The band uses a neutral color (light gray or a very faint tint of the dominant activity type color).
- The trip name appears as small text at the start of the band, similar to how month labels work.
- When multiple trips overlap, their bands stack or use slightly different tints.

```
┌───────────────────────────────────────────────┐
│  FOSDEM 2027 (subtle background band)         │
│  ████ Flight EWR→BRU  ████████ Conference     │
│       ████████ Hotel Brussels                 │
│                       ████ Flight BRU→EWR     │
└───────────────────────────────────────────────┘
```

### Year view

- Same concept: a light background region spanning the trip's date range behind the activity bars.
- At year scale, the trip might be more prominent than individual activities — showing "FOSDEM 2027" as a label is more useful than truncated activity names.

### Day view

- Days that belong to a trip show the trip name as a subtle label (e.g., in the date column or as a small badge).
- Could also show a left-edge color bar for the trip.

### Agenda view

- Activities grouped under trip headers.
- Trip header shows: name, date range, locations.
- Activities within a trip are indented or visually nested.
- Standalone activities (no trip) render at the top level as before.

```
FOSDEM 2027 — Jan 30 → Feb 4 — Brussels, Paris
  ├ Flight EWR→BRU       Jan 30         travel
  ├ Hotel Brussels        Jan 30 - Feb 3 stay
  ├ FOSDEM Conference     Feb 1 - Feb 3  conference
  └ Flight BRU→EWR       Feb 4          travel

Dentist                   Feb 10         commitment
```

## Conflict Detection

### Current behavior (string-based, per-day)

Multiple activities with different locations on the same day = conflict. This produces false positives within trips (flight and hotel in different cities on the same day).

### With trips

- Activities **within the same trip** do not conflict with each other. The trip itself implies intentional movement between locations.
- Activities in **different trips** on the same day = conflict (worth flagging).
- A **standalone activity** conflicting with a **trip activity** = conflict (you have a commitment while traveling).
- Trips **overlapping each other** = conflict at the trip level ("you have two trips that overlap").

This subsumes #31 (travel activities resolving conflicts) for the common case. Travel-type activities within a trip are implicitly connected. Standalone travel activities still resolve conflicts per #31.

## Interaction Design

### Creating a trip

**From quick add:** Typing "FOSDEM Jan 30 - Feb 4 in Brussels" could create a trip + first activity. The parser could detect multi-day events with a name that sounds like a trip. This is a future enhancement — for v1, trips are created explicitly.

**From the UI:**
1. A "New trip" button or modal that asks for a name.
2. Then add activities to it (click existing → "Add to trip", or create new within the trip).
3. Or: select multiple activities in agenda view → "Group into trip".

**From calendar import:**
1. Import candidate events → select several → "Create trip from selection".
2. Or the import flow suggests trip groupings based on date proximity and location.

### Editing a trip

- Rename via the trip header in agenda view or a trip detail modal.
- Drag an activity out of a trip (unassign) or into a trip (assign).
- Delete a trip: activities become standalone (not deleted).

### Quick assignment

- In the activity modal (create/edit), add a "Trip" dropdown showing existing trips + "New trip..." option.
- This is the primary way to assign activities to trips.

## Implementation Phases

### Phase 1: Data model + API (backend)

1. Add `Trip` entity and store (using `store.Collection[Trip]`)
2. Add `tripId` to Activity entity
3. Add Trip CRUD endpoints to OpenAPI spec
4. Update activity endpoints to accept/return tripId
5. Update codegen

### Phase 2: Basic UI (frontend)

1. Trip dropdown in the activity modal
2. Agenda view grouping (trip headers with nested activities)
3. Trip management (create, rename, delete) — could be a simple list in a settings/sidebar panel

### Phase 3: Visual treatment (frontend)

1. Month view background bands for trips
2. Year view background bands
3. Day view trip labels

### Phase 4: Conflict evolution

1. Suppress intra-trip conflicts
2. Flag inter-trip overlap
3. Flag standalone-vs-trip conflicts

## Relationship to Other Features

- **#31 Travel resolves conflicts**: Trips partially subsume this. Within a trip, all location changes are intentional. For standalone activities, #31 still applies.
- **#52 Calendar import**: Trips provide the grouping target for imported events. "Import these 5 events as a trip" is the key flow.
- **#49 Places**: Trips with structured locations enable "which cities does this trip visit?" queries.
- **#54 Public sharing**: The "Where is Michael" dashboard could show trips as the primary unit: "In Brussels for FOSDEM" rather than listing individual activities.

## Open Questions

1. **Should trips have their own color?** Or derive color from the dominant activity type? A trip color would help distinguish overlapping trips visually. But it's another thing to configure.

2. **Should we show a trip summary in the tooltip?** When hovering over an activity that belongs to a trip, the tooltip could show "Part of FOSDEM 2027 (4 activities, Jan 30 - Feb 4)".

3. **Multi-location trip labels**: A trip visiting Brussels, Amsterdam, and Paris — what location do we show in compact views? "Brussels +2" or "Europe" or just the first location?

4. **Trip templates**: "I go to this conference every year" — could a trip be duplicated with shifted dates? Deferred but the data model should support it (trips are just named groups, easy to clone).
