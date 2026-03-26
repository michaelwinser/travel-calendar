# Places and Geo: Design Document

## Overview

Replace freeform location strings with structured Place entities backed by an embedded city gazetteer. This enables autocomplete, duplicate detection ("SF" = "San Francisco"), travel route modeling, geography-aware conflict detection, and timezone display -- all without external API dependencies.

**PRD**: `docs/prd/places.md` covers product requirements, use cases, and UX flows. This document covers technical implementation.

**Issues**: #49 (Places), #31 (travel conflicts), #50 (geo conflicts), #51 (timezones)

## Principles

1. **Progressive enrichment.** A Place can be just a name. Gazetteer data is layered on when available, never required.
2. **Preserve user input.** `Activity.Location` always stores the user's original text. `Activity.PlaceID` is a separate optional link. The user's string is never overwritten or replaced.
3. **No external dependencies.** The gazetteer is embedded in the binary. Autocomplete and resolution work offline, in local CLI mode, and in the desktop app.
4. **Graceful degradation.** Every feature falls back through the same chain: coordinates -> place identity -> string comparison. No feature breaks if geo data is absent.

## Data Model

### Place entity

```go
type Place struct {
    ID        string   `json:"id"        store:"id,pk"`
    UserID    string   `json:"userId"    store:"user_id,index"`
    Name      string   `json:"name"      store:"name"`
    Aliases   string   `json:"aliases"   store:"aliases"`       // JSON-encoded []string
    City      string   `json:"city"      store:"city"`
    Country   string   `json:"country"   store:"country"`       // ISO 3166-1 alpha-2
    Latitude  float64  `json:"latitude"  store:"latitude"`
    Longitude float64  `json:"longitude" store:"longitude"`
    Timezone  string   `json:"timezone"  store:"timezone"`      // IANA, e.g. "Europe/Brussels"
    Kind      string   `json:"kind"      store:"kind"`          // home|work|airport|city|venue|other
    CreatedAt string   `json:"createdAt" store:"created_at"`
}
```

**Why `Aliases` is a JSON string, not a separate table:** appbase `store.Collection` maps to a single SQLite table with scalar columns. There is no relation/join support. Storing aliases as a JSON-encoded string array keeps the Place as a single row. Alias matching queries do a `LIKE` scan on the aliases column for prefix/substring search. At the scale of a single user's places (dozens, not thousands), this is fast enough. If it becomes a bottleneck, we can add a denormalized `place_aliases` collection later.

**Coordinates:** `Latitude` and `Longitude` use zero values (0,0) to mean "not set." This is technically a valid coordinate (Gulf of Guinea) but no user will have activities there. The API and store treat `latitude == 0 && longitude == 0` as "no coordinates." An alternative is pointer types, but appbase store tags do not support nullable floats. The zero-value convention is simpler and matches how the rest of the codebase handles optional strings (empty string = not set).

**Kind enum values:**
- `home` -- user's home base (at most one per user, used as default location)
- `work` -- regular workplace
- `airport` -- transit point, not a destination (affects conflict logic)
- `city` -- general city-level place
- `venue` -- specific venue (conference center, hotel, etc.)
- `other` -- catch-all

### Activity changes

```go
type Activity struct {
    // ...existing fields...
    PlaceID            string `json:"placeId"            store:"place_id"`
    OriginPlaceID      string `json:"originPlaceId"      store:"origin_place_id"`
    DestinationPlaceID string `json:"destinationPlaceId" store:"destination_place_id"`
}
```

`PlaceID` applies to all activity types. `OriginPlaceID` and `DestinationPlaceID` apply only when `Type == "travel"`. The server ignores origin/destination on non-travel activities.

The `Location` string field is kept. When a Place is linked, `Location` is still written (set to Place.Name if the user selected from autocomplete, or left as the user's raw input if they typed freely). This means CLI `list` output, exports, and shared views work without joining to the places table.

### OpenAPI schema changes

Add to `Activity` schema:
```yaml
placeId:
  type: string
  description: Optional reference to a Place entity
originPlaceId:
  type: string
  description: Origin place for travel activities
destinationPlaceId:
  type: string
  description: Destination place for travel activities
```

Add to `CreateActivityRequest` and `UpdateActivityRequest`: `placeId`, `originPlaceId`, `destinationPlaceId` (all optional strings).

Add `Place` schema and place endpoints (see API section below).

### Migration

appbase `store.Collection` auto-creates columns for new struct fields on startup. Adding `PlaceID`, `OriginPlaceID`, and `DestinationPlaceID` to the Activity struct is a zero-migration change -- the columns appear as empty strings on existing rows. No backfill script is needed for Phase 1. The system treats empty `PlaceID` as "no place linked" and falls back to string-based behavior.

## Gazetteer

### Data source

GeoNames `cities15000.txt` -- all cities with population greater than 15,000. Approximately 25,000 entries. Fields used:

| GeoNames column | Our field | Example |
|---|---|---|
| name | name | "Brussels" |
| alternatenames | alternateNames | "Bruxelles,Brussel,BRU" |
| country code | country | "BE" |
| latitude | lat | 50.85 |
| longitude | lng | 4.35 |
| timezone | timezone | "Europe/Brussels" |
| population | population | 1,019,022 |

Plus a supplementary airports dataset (~5,000 entries) mapping IATA codes to parent cities. This lets "BRU" resolve to Brussels and "EWR" resolve to the New York metro area.

### Embedded binary format

The raw GeoNames TSV is ~3MB. We preprocess it into a compact binary format at build time and embed it with `//go:embed`.

```
gazetteer/
    prepare.go          # Build-time: TSV -> binary converter (go generate)
    cities.bin          # Embedded binary data
    airports.bin        # Embedded airport data
    gazetteer.go        # Runtime: load, search, types
    gazetteer_test.go
```

Binary format (little-endian):

```
Header:
  uint32  entryCount
  uint32  nameIndexCount

Entry (fixed-size, packed):
  uint16  nameOffset      # into string table
  uint16  nameLength
  uint16  altNamesOffset  # into string table
  uint16  altNamesLength
  [2]byte country         # ISO alpha-2, ASCII
  float32 lat
  float32 lng
  uint32  population
  uint16  tzOffset        # into string table
  uint16  tzLength

NameIndex (sorted by lowercase name):
  uint16  nameOffset      # into string table (lowercased)
  uint16  nameLength
  uint16  entryIndex      # which Entry this name refers to

StringTable:
  []byte  concatenated UTF-8 strings
```

Each name and alternate name gets an entry in the NameIndex, all pointing back to the parent Entry. The NameIndex is sorted lexicographically, enabling binary search for prefix matching.

**Size estimate:** ~25k entries * ~40 bytes/entry + ~5k airports * ~20 bytes + name index (~150k entries * 6 bytes) + string table (~2MB) = roughly 3-4MB embedded in the binary. Acceptable for a Go binary that is already ~20MB.

**Why not just embed the CSV and parse at load time?** The binary format avoids allocating 150k+ Go strings on first access. The `//go:embed` directive causes the OS to memory-map the data section of the binary, so the gazetteer occupies virtual memory but not necessarily physical RAM until accessed. The NameIndex can be searched in-place via binary search on the byte slice without allocating.

### Lazy loading

```go
var (
    //go:embed cities.bin
    citiesData []byte

    //go:embed airports.bin
    airportsData []byte

    gazetOnce sync.Once
    gazet     *Gazetteer
)

func GetGazetteer() *Gazetteer {
    gazetOnce.Do(func() {
        gazet = loadGazetteer(citiesData, airportsData)
    })
    return gazet
}
```

The gazetteer is loaded on first call to `GetGazetteer()`, not at process startup. CLI commands that do not use location features pay zero cost. The `sync.Once` ensures thread safety for the web server.

### Search algorithm

```
PrefixSearch(query string, limit int) []GazetteerResult:
  1. Lowercase the query
  2. Binary search NameIndex for first entry >= query
  3. Scan forward while entries still have the query as prefix
  4. Collect matching Entry indices (deduplicate: multiple names can point to same entry)
  5. Sort by population descending
  6. Return top `limit` results
```

Complexity: O(log N) for the binary search + O(K) for scanning K prefix matches. For a 3-character prefix like "bru", K is typically < 50. No allocations beyond the result slice.

**Why not a trie?** A sorted array with binary search is simpler to implement, simpler to serialize as an embedded binary, and fast enough. The NameIndex has ~150k entries; binary search touches ~17 comparisons. If profiling shows this is slow (unlikely), a trie or radix tree can replace the search without changing the public API.

### Airport handling

Airports get entries in the NameIndex keyed by their IATA code (e.g., "ewr", "cdg", "bru"). Each airport entry links to its parent city's Entry. When "EWR" is searched, the result is the Newark/New York metro Entry, not a separate airport entity.

For Place creation, when a user selects an airport match, the Place gets `kind: airport` and the city-level coordinates/timezone from the parent city. This means "EWR" and "JFK" both resolve to the New York area for conflict detection purposes.

## API Endpoints

### Place CRUD

```
GET    /api/places                → list user's places
POST   /api/places                → create a place
GET    /api/places/{id}           → get place details
PUT    /api/places/{id}           → update place
DELETE /api/places/{id}           → delete (400 if activities reference it)
```

### Place resolution

```
POST   /api/places/resolve
  Request:  { "text": "Bru" }
  Response: {
    "exact": null | Place,
    "suggestions": [
      {
        "source": "user",        // from user's existing places
        "place": Place,
        "score": 1.0
      },
      {
        "source": "gazetteer",   // from embedded city data
        "name": "Brussels",
        "country": "BE",
        "latitude": 50.85,
        "longitude": 4.35,
        "timezone": "Europe/Brussels",
        "population": 1019022,
        "score": 0.95
      }
    ]
  }
```

Resolution logic:

1. Normalize input: trim whitespace, lowercase for matching.
2. Search user's existing Places: exact match on `name` or any alias (case-insensitive). If found, return as `exact`.
3. Search user's Places by prefix on `name` and aliases. Return as `suggestions` with `source: "user"`.
4. Search gazetteer by prefix. Return top 5 results as suggestions with `source: "gazetteer"`.
5. Merge and deduplicate: if a user place has the same city/country as a gazetteer result, prefer the user place.
6. Sort: exact match first, then user places, then gazetteer results (by population).

The `score` field is informational. For Phase 1, user exact matches get 1.0, user prefix matches get 0.9, gazetteer matches get 0.8 * (population rank). The score is not stored; it exists only to help the frontend sort the dropdown.

### Place merge

```
POST   /api/places/{id}/merge
  Request:  { "sourceId": "..." }
  Response: Place (the target, now with source's name as alias)
```

Merge moves all activity references from source to target, adds source's name to target's aliases, and deletes source. This is a single transaction.

## Server Implementation

### PlaceStore

```go
type PlaceStore struct {
    coll *store.Collection[Place]
}

func NewPlaceStore(d *db.DB) (*PlaceStore, error) {
    coll, err := store.NewCollection[Place](d, "places")
    if err != nil {
        return nil, err
    }
    return &PlaceStore{coll: coll}, nil
}
```

Methods follow the same pattern as `ActivityStore` and `TripStore`: `Create`, `Get`, `List(userID)`, `Update`, `Delete`.

Additional methods:

```go
// FindByName returns a user's place matching name or alias (case-insensitive).
func (s *PlaceStore) FindByName(userID, name string) (*Place, error)

// SearchByPrefix returns user's places where name or aliases start with prefix.
func (s *PlaceStore) SearchByPrefix(userID, prefix string, limit int) ([]Place, error)
```

`FindByName` does a case-insensitive query on `name`, then falls back to scanning the `aliases` JSON column with `LIKE`. `SearchByPrefix` uses `name LIKE 'prefix%'` plus alias scanning.

### Handler wiring

The `ActivityServer` struct gains a `places` field:

```go
type ActivityServer struct {
    store      *ActivityStore
    trips      *TripStore
    places     *PlaceStore
    gazetteer  *gazetteer.Gazetteer  // lazy-loaded, nil until first use
    // ...
}
```

New handler methods implement the `ServerInterface` additions from codegen:
- `ListPlaces`, `CreatePlace`, `GetPlace`, `UpdatePlace`, `DeletePlace`
- `ResolvePlaces` (the resolve endpoint)
- `MergePlaces`

### Activity creation with place resolution

When `CreateActivity` or `UpdateActivity` receives a `placeId`, the handler:
1. Verifies the Place exists and belongs to the user.
2. Stores the `placeId` on the Activity.
3. If `location` was not provided in the request, sets it to `Place.Name`.

When the request has a `location` string but no `placeId`, behavior is unchanged from today. The frontend/CLI is responsible for calling resolve and setting the `placeId`.

## Conflict Detection Changes

### Phase 1-2: Place identity

Replace the current conflict check in `CheckDate`:

```go
// Current
locations := map[string]bool{}
for _, a := range items {
    if a.Location != "" {
        locations[a.Location] = true
    }
}
hasConflict := len(locations) > 1

// New: extract to a function
hasConflict := detectConflict(items, placeStore, userID)
```

The `detectConflict` function:

```
detectConflict(activities, placeStore, userID):
  Group activities by effective location:
    - If activity has PlaceID → group key is PlaceID
    - Else → group key is lowercase(Location)

  If only one group → no conflict
  If multiple groups → conflict (for now)
```

This immediately fixes "SF" vs "San Francisco" when both activities link to the same Place, without requiring coordinates.

### Phase 3: Travel resolution (#31)

```
detectConflict(activities, placeStore, userID):
  Separate activities into travel and non-travel.

  Collect location set from non-travel activities (by place ID or string).

  For each travel activity with origin + destination:
    If origin matches one location in the set and destination matches another:
      Remove both from the conflict set (travel bridges them).

  For chained travel (A->B, B->C on same day):
    Build a graph of connected locations.
    If all non-travel locations are in the same connected component → no conflict.

  Remaining unconnected locations → conflict.
```

"Matches" means: same PlaceID, or same city on the Place, or (Phase 4) within distance threshold.

### Phase 4: Distance-based (#50)

```
detectConflict additions:
  When comparing two places that have coordinates:
    distance = haversine(place1, place2)
    if distance < threshold (default 50km):
      Treat as same location (no conflict).

  When one or both places lack coordinates:
    Fall back to place identity (same PlaceID) or string comparison.
```

The Haversine function is straightforward (~10 lines of Go) and lives in the gazetteer package since it operates on coordinates.

```go
// gazetteer/geo.go
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64
```

### Conflict detection fallback chain

The full chain, from most precise to least:

1. **Same PlaceID** -> no conflict (always checked)
2. **Coordinates within threshold** -> no conflict (Phase 4, requires both places to have coordinates)
3. **Travel bridges locations** -> no conflict (Phase 3, requires travel with origin/destination)
4. **Different PlaceIDs, no coordinates** -> conflict
5. **No PlaceID on one or both** -> fall back to string comparison (current behavior)
6. **Different location strings** -> conflict (current behavior, unchanged)

## CLI Changes

### Phase 1

```
travel places                    # List all places
travel places show <name-or-id>  # Show place details + linked activities
```

### Phase 2

```
travel add ... --loc Brussels    # Resolves "Brussels": exact match → link, no match → create
```

When `--loc` is provided, the CLI calls `POST /api/places/resolve` before creating the activity. If an exact match is returned, the activity is created with that `placeId`. If suggestions are returned but no exact match, the CLI creates a new Place from the string and prints a note:

```
Created new place "Brussels"
  Tip: travel places show Brussels (to see details)
```

If the gazetteer has a match, the CLI prints:

```
Created new place "Brussels" (Belgium, Europe/Brussels)
```

The CLI does not do interactive selection from suggestions. That is a web UI feature.

### Phase 3

```
travel places merge <target> <source>   # Merge source into target
travel places delete <name-or-id>       # Delete unused place
```

## Frontend Changes

### Autocomplete component

A new `PlaceAutocomplete.svelte` component wraps the location input field:

1. On input change (debounced 200ms), call `POST /api/places/resolve` with the current text.
2. Display a dropdown with results, grouped: "Your places" then "Cities".
3. Selecting a suggestion sets `placeId` on the form data and fills the text field with the place name.
4. Pressing Enter or clicking away without selecting closes the dropdown. The raw text is kept as-is.

The component replaces the plain text input on the activity create/edit form. TypeScript types are generated from the updated OpenAPI spec.

### Suggestion chips

On the activity detail view, if `placeId` is empty and `location` is non-empty, the frontend calls resolve and displays suggestion chips below the location. Clicking a chip calls `PUT /api/activities/{id}` with the selected `placeId`.

### Phase 5: Timezone display

When an activity's Place has a timezone, show it in the activity detail and day view. Use the browser's `Intl.DateTimeFormat` with the IANA timezone string. No new API calls needed -- the timezone comes from the Place entity which is already available.

## File Layout

New files:

```
internal/app/
    place_store.go          # PlaceStore (CRUD + search methods)
    place_handlers.go       # HTTP handlers for place endpoints

gazetteer/
    prepare.go              # go:generate script: TSV -> binary
    cities.bin              # Embedded city data
    airports.bin            # Embedded airport data
    gazetteer.go            # Gazetteer type, PrefixSearch, lazy loading
    geo.go                  # HaversineKm and distance utilities
    gazetteer_test.go       # Search correctness and benchmark tests
```

Modified files:

```
openapi.yaml                # Place schema, place endpoints, activity field additions
internal/app/store.go       # Activity struct gains PlaceID, OriginPlaceID, DestinationPlaceID
internal/app/server.go      # CheckDate conflict logic, activity create/update with placeId
main.go                     # CLI: places subcommand, --loc resolution in add command
```

## Phase Plan

### Phase 1: Place entity + gazetteer + resolution

**Goal:** Places exist as first-class entities. The gazetteer is embedded and searchable. The resolve endpoint works.

1. Add Place schema to `openapi.yaml`, run codegen.
2. Implement `PlaceStore` in `internal/app/place_store.go`.
3. Build gazetteer binary format and embed. Write `PrefixSearch`.
4. Implement resolve endpoint: search user places + gazetteer, return results.
5. Implement place CRUD handlers.
6. Add `placeId` to Activity schema (OpenAPI + struct). Run codegen.
7. Update activity create/update to accept and store `placeId`.
8. Tests: place CRUD, gazetteer search correctness (Brussels, EWR, edge cases), resolve endpoint.

**Done when:** `POST /api/places/resolve { "text": "Bru" }` returns Brussels as a gazetteer suggestion, and creating a Place from it works.

### Phase 2: Wire into activity flows

**Goal:** Location input across CLI and web UI uses place resolution.

1. CLI `add --loc` calls resolve before creating activity.
2. CLI `places` and `places show` commands.
3. Frontend `PlaceAutocomplete.svelte` component.
4. Wire autocomplete into activity create/edit forms.
5. Suggestion chips on activity detail view.
6. Update conflict detection to use place identity (`detectConflict` function).
7. Parser integration: `parseActivity` calls resolve on extracted location.
8. Tests: CLI place resolution, autocomplete E2E, conflict detection with places.

**Done when:** Two activities with "SF" and "San Francisco" linked to the same Place do not conflict.

### Phase 3: Travel conflict resolution (#31)

**Goal:** Travel activities bridge locations instead of creating conflicts.

1. Add `originPlaceId` and `destinationPlaceId` to Activity schema.
2. Update resolve to handle route strings ("London -> Paris" -> two resolve calls).
3. Implement travel-aware conflict detection (chain resolution).
4. CLI: travel-type activities prompt for origin/destination places.
5. Frontend: origin/destination fields on travel activity form.
6. Tests: travel bridges, chained segments, mixed travel + non-travel days.

**Done when:** A day with "Flight EWR->BRU" + "FOSDEM in Brussels" shows no conflict.

### Phase 4: Geography-aware conflicts (#50)

**Goal:** Distance-based conflict detection using gazetteer coordinates.

1. Implement `HaversineKm` in gazetteer package.
2. Add distance threshold to conflict detection logic.
3. Add user preference for threshold (default 50km).
4. Update DateCheck response to include distance info when relevant.
5. Tests: nearby places (no conflict), far places (conflict), missing coordinates (fallback).

**Done when:** Activities in "Brooklyn" and "Manhattan" (both with NYC-area coordinates) do not conflict.

### Phase 5: Timezone display (#51)

**Goal:** Show timezone context using gazetteer timezone data.

1. Frontend: display timezone in activity detail when Place has timezone.
2. Frontend: timezone transition display on travel days.
3. iCal export: set VTIMEZONE from Place timezone when available.
4. Tests: timezone display correctness, iCal export with timezones.

**Done when:** An activity in Brussels shows "(CET, UTC+1)" in the detail view.

## Open Questions Resolved

**Geocoding provider:** Answered -- use the embedded GeoNames gazetteer. No external API. City-level resolution covers the primary use cases (autocomplete, conflict detection, timezones). Street-level geocoding is out of scope.

**Airport semantics:** Airports are linked to their parent city in the gazetteer. `kind: airport` on a Place means it is a transit point. For conflict detection, airports are treated as their parent city's location. "EWR" and "JFK" both resolve to the New York area.

**"Home" handling:** `kind: home` is a regular place with a distinguished kind. The system does not enforce one-per-user at the store level, but the CLI `add` command defaults unresolved empty locations to the user's home place if one exists.

**Route locations:** When the parser extracts "London -> Paris", the resolve endpoint is called twice (once for "London", once for "Paris"). The resulting place IDs populate `originPlaceId` and `destinationPlaceId`. The `location` string stores the original "London -> Paris" for display.

## Testing Strategy

**Gazetteer unit tests:**
- Prefix search for "bru" returns Brussels before Bruges (population ranking).
- "EWR" returns New York area.
- Empty query returns nothing.
- Exact match for "Tokyo" returns Tokyo.
- Case insensitivity: "BRUSSELS" = "brussels" = "Brussels".

**Place store tests:**
- CRUD operations.
- `FindByName` matches aliases.
- `SearchByPrefix` returns correct results.

**Resolve endpoint tests:**
- Exact match against existing user place.
- Prefix match against user places + gazetteer.
- Gazetteer-only results when user has no places.
- Deduplication when user place matches gazetteer entry.

**Conflict detection tests:**
- Same place ID, different location strings -> no conflict.
- Different place IDs, no coordinates -> conflict.
- Travel bridges two locations -> no conflict.
- Chained travel segments -> no conflict.
- Nearby coordinates -> no conflict (Phase 4).
- Missing coordinates -> fallback to identity/string.

**E2E tests:**
- `travel add "Trip" --loc Brussels && travel add "Meeting" --loc Brussels` -> same place, no conflict on check.
- `travel places` lists created places.
- `travel places show Brussels` shows linked activities.
