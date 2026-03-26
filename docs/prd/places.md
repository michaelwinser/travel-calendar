# Places — PRD

## Vision

Evolve the travel calendar's location handling from freeform strings to structured places. A Place carries a name, geography, and timezone — enough for the app to understand that "SF" and "San Francisco" are the same city, that a flight connects two cities, and that Brussels is in CET.

This is a foundation feature. Every location-aware capability — smarter conflict detection, timezone display, map views, autocomplete — depends on having structured place data.

## Problem

Locations today are plain strings. This causes real problems:

1. **False conflicts**: "San Francisco" and "SF" are flagged as different locations on the same day.
2. **Missed conflicts**: No way to know that two differently-named locations are actually far apart vs. nearby.
3. **No timezone awareness**: Calendar import/export can't handle timezone conversion. "What time is it where I'll be?" is unanswerable.
4. **No autocomplete**: Users retype location strings with inconsistent spelling every time.
5. **Travel routes are fragile**: The parser produces `"London -> Paris"` as a single string. There's no structured representation of origin and destination.

## Design Principles

1. **Progressive enrichment** — A location string always works. Structured data is layered on, never required. Users who type "Home" should never be forced through a geocoding flow.
2. **User vocabulary first** — The user's name for a place is canonical. Geocoded data is metadata, not identity. "Home" is a valid place even though it isn't a city.
3. **Lightweight Places** — A Place is a small reusable entity, not a heavyweight address record. Think city-level, not street-level.
4. **Gradual migration** — Existing freeform strings continue to work. The system learns places over time from user input and offers to consolidate duplicates.

## Data Model

### Place Entity

```
Place
  id:        UUID
  userId:    string
  name:      string          # User's preferred name ("Home", "Brussels", "SFO")
  aliases:   []string        # Alternative names (["SF", "San Francisco", "SFO"])
  city:      string          # Normalized city name (optional)
  country:   string          # ISO 3166-1 alpha-2 (optional)
  latitude:  float64         # Optional, for distance calculations
  longitude: float64         # Optional, for distance calculations
  timezone:  string          # IANA timezone (optional, e.g. "Europe/Brussels")
  kind:      enum            # home | work | airport | city | venue | other
  createdAt: datetime
```

**Key decisions:**
- Places are per-user. "Home" means different things to different people.
- `name` is the display label. `aliases` enable fuzzy matching. `city`/`country`/`coordinates` enable geo features.
- All geo fields are optional. A place can start as just a name and get enriched later.
- `kind` helps the UI (airport icon, home icon) and conflict logic (airports aren't destinations).

### Activity Changes

The Activity entity gains an optional `placeId` field alongside the existing `location` string:

```
Activity
  location:  string          # Existing freeform field (kept for backwards compat)
  placeId:   string          # Optional FK to Place
```

When `placeId` is set, the Place's `name` is the canonical display value. The `location` string is still written (for export, CLI display, and backwards compatibility) but the Place is authoritative for conflict detection and geo features.

### Travel Activities: Origin and Destination

Travel-type activities represent transit between two places. Add optional fields:

```
Activity (when type == "travel")
  originPlaceId:      string   # Where you're coming from
  destinationPlaceId: string   # Where you're going
```

The existing `location` string (e.g. "EWR -> CDG") remains as display text. The structured place references enable route-aware conflict resolution.

## Features

### P0: Place Resolution and Autocomplete

**Goal:** When a user types a location, match it against known places and suggest reuse.

- As the user types in the location field (web) or provides `--loc` (CLI), check against existing Places for the user.
- Match on `name` and `aliases`, case-insensitive.
- If a match is found, link the activity to that Place.
- If no match, create a new Place from the string (name only, no geo data yet).
- Web UI: dropdown autocomplete showing matching places (see "Two moments of suggestion" below).
- CLI: exact match only (no interactive autocomplete). Print a note if a new place is created vs. an existing one matched.

**Parser integration:** When the quick-add parser extracts a location, run the same resolution logic. The `ParseResult` should include a `placeId` if a match was found, and `placeSuggestions` if there are near-matches.

**Two moments of suggestion:**

The system has two chances to link a location string to a Place — one eager, one lazy.

1. **Inline autocomplete (while typing):** As the user types in the location field, show a dropdown of matching places from their vocabulary. Matches on name and aliases, prefix and substring. Selecting a suggestion sets the `placeId` and fills the field with the Place's canonical name.

2. **Post-hoc suggestion chips (after save):** If the user ignores autocomplete or types a string that doesn't exactly match a Place, the activity is saved with the raw string and no `placeId`. The next time the user views or edits that activity, the UI shows suggestion chips — small clickable badges like `Nevada` — below the location field. These are candidate Place matches resolved from the raw string (prefix match, alias match, edit distance). Clicking a chip links the activity to that Place in one click.

This two-pass design means the system learns even when the user is moving fast. A user typing "Nev" and hitting enter doesn't get interrupted, but the app quietly resolves the best candidate and offers it non-intrusively on the activity detail view. Over time, unresolved locations shrink to zero as the user accepts chips during normal browsing.

**Resolution endpoint:** Both moments use the same backend:

```
POST /api/places/resolve
{ "text": "Nev" }

→ { "exact": null, "suggestions": [{ "id": "...", "name": "Nevada", "score": 0.9 }] }
```

When `exact` is non-null, the UI can auto-link silently (e.g., user typed "Nevada" and a Place named "Nevada" exists). When only `suggestions` are returned, the UI shows chips.

### P0: Conflict Detection with Places

**Goal:** Conflicts are determined by place identity, not string equality.

Current behavior: `len(uniqueLocationStrings) > 1` = conflict.

New behavior:
- Activities sharing the same Place (by ID) never conflict with each other, regardless of the `location` string value.
- Activities with different Places conflict only if the Places are in different cities (or coordinates are far apart).
- Activities with no Place fall back to string comparison (current behavior).

This immediately fixes the "SF" vs "San Francisco" false-conflict problem, without requiring geocoding.

### P1: Travel Activities Resolve Conflicts

**Goal:** A travel activity connecting location A to location B means the user is intentionally transiting. It should resolve the A/B conflict, not create one.

Rules:
- A day with activities in locations A and B is not a conflict if there is a `travel`-type activity on that day whose origin is A (or same city as A) and destination is B (or same city as B).
- A travel activity's own location (the route string) is not counted as a separate location for conflict purposes.
- A day with a travel activity AND a non-travel commitment in a location that doesn't match either origin or destination IS a conflict (you can't be at the dentist in Chicago while flying New York to London).

Edge cases:
- Multiple travel segments in one day (e.g., EWR->CDG, CDG->Brussels) — chain resolution. If the segments connect, no conflict.
- Travel activity without origin/destination place IDs — falls back to current behavior (counts as a location).

### P1: Place Enrichment

**Goal:** Optionally enrich places with geographic data.

Two enrichment paths:

1. **Geocoding service** (e.g., Google Places API, Nominatim): Given a place name, look up coordinates, timezone, and country. User confirms the result. This is a user-initiated action ("enrich this place"), not automatic.

2. **Manual entry**: User can edit a Place to add city, country, timezone, coordinates.

Enrichment enables:
- Distance-based conflict detection (P2)
- Timezone display (P2)
- Map views (future)

### P2: Geography-Aware Conflict Detection

**Goal:** Use coordinates to determine whether two locations are "nearby" (no conflict) vs. "far apart" (conflict).

- Configurable proximity threshold (default: 50km).
- Two places within threshold = same area, no conflict.
- Two places beyond threshold without a bridging travel activity = conflict.
- Requires coordinates on both places. If either lacks coordinates, fall back to place-identity or string comparison.

### P2: Timezone Awareness

**Goal:** Display timezone context for activities in locations with known timezones.

- Show local time in activity tooltips: "Brussels (CET, UTC+1)".
- On travel days, show timezone transition: "New York (EST) -> Brussels (CET)".
- Calendar export (iCal) uses correct timezone for events when known.
- "What time is it there?" indicator in the day view for away locations.

Requires Place enrichment with IANA timezone data.

### P2: Place Management

**Goal:** Users can view, edit, merge, and delete their places.

- List all places with usage count (number of activities referencing each).
- Edit place details (name, aliases, kind, geo data).
- Merge two places: reassign all activities from place B to place A, add B's name as an alias of A, delete B.
- Delete unused places.
- CLI commands: `travel places list`, `travel places show <name>`, `travel places merge <a> <b>`.
- Web UI: places management view (lower priority than CLI).

## API Endpoints

```
GET    /api/places              # List user's places
POST   /api/places              # Create a place
GET    /api/places/{id}         # Get place details
PUT    /api/places/{id}         # Update place
DELETE /api/places/{id}         # Delete place (fails if in use)
POST   /api/places/{id}/merge   # Merge another place into this one
POST   /api/places/resolve      # Given a string, find or create a matching place
```

## Migration

Existing data uses freeform location strings with no Place references.

**Phase 1 — Transparent coexistence:**
- All existing code continues to work with `location` strings.
- New activities created via the web UI or CLI get a Place auto-created if one doesn't exist.
- `placeId` is optional everywhere.

**Phase 2 — Backfill:**
- A one-time migration script scans all activities, groups by normalized location string, creates Places, and links them.
- Normalization: trim whitespace, collapse case, handle obvious duplicates ("Brussels" / "brussels").
- Ambiguous matches (e.g., "Home" — is it a generic label or a specific place?) are flagged for user review.

**Phase 3 — Place-first:**
- Web UI location input becomes place-first (autocomplete, create-on-miss).
- Conflict detection uses places as primary signal.
- `location` string is still written for CLI display but derived from Place name.

## CLI Changes

New commands:
```
travel places                    # List all places
travel places show <name>        # Show place details and linked activities
travel places merge <a> <b>      # Merge place B into A
```

Modified commands:
```
travel add ... --loc Brussels    # Resolves "Brussels" to a Place (creates if new)
travel list                      # No change to output format
travel check 2026-04-03          # Conflict detection uses place identity
```

## Non-Goals

- Street-level addresses or full postal addresses.
- Automatic geocoding without user confirmation (privacy concern).
- Real-time location tracking.
- Map view (future feature, enabled by this work but not in scope).
- Travel time estimates between places.

## Open Questions

1. **Geocoding provider**: Google Places API gives high quality but costs money and requires API key management. Nominatim (OpenStreetMap) is free but lower quality. Start with manual entry and defer the provider choice?

2. **Airport semantics**: Is "EWR" a place (Newark airport) or shorthand for the New York area? For conflict detection, you probably want to treat airports as belonging to their metro area. Should `kind: airport` imply a parent city relationship?

3. **"Home" handling**: Should there be a distinguished "home" place per user, or is it just a regular place with `kind: home`? A special home place would simplify the "default location when no activities" logic.

4. **Route locations**: When the parser produces "London -> Paris", should that create two places and set origin/destination? Or should the user be prompted to confirm the places separately?

## Related Issues

- #31 — Travel activities should resolve location conflicts, not create them
- #49 — Places: structured locations with geocoding
- #50 — Geography-aware conflict detection
- #51 — Timezone awareness
