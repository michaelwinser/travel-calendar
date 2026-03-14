# Migration Plan: SQLite to Firestore

## Context

The Travel Calendar backend currently uses SQLite with raw SQL queries (no ORM). This plan switches to Firestore with Firebase Local Emulator for development. The current database layer is clean but tightly coupled — a concrete `*store.Store` struct is passed directly to the service layer with no interface abstraction. This plan introduces an interface, implements Firestore behind it, and updates infrastructure to use the Firebase Emulator.

Good news: The current query patterns are simple (no JOINs, no aggregations, one transaction) and the data model maps naturally to Firestore documents. The main trade-off is losing LIKE text search, which we'll handle pragmatically with in-memory filtering given the small dataset.

---

## Phase 1: Extract Store Interface (no behavior change)

### 1.1 Create `store/interface.go`

- Define `StoreInterface` with all 42 methods from the current `Store` struct
- Define `var ErrNotFound = errors.New("not found")` for store-agnostic not-found handling

### 1.2 Rename concrete implementation

In `packages/backend/internal/store/sqlite.go`:
- `Store` struct → `SQLiteStore`
- `New()` → `NewSQLite()`
- Add compile-time check: `var _ StoreInterface = (*SQLiteStore)(nil)`
- Have `GetTrip`, `GetItem`, etc. return `ErrNotFound` instead of `sql.ErrNoRows`

### 1.3 Update consumers

| File | Change |
|------|--------|
| `internal/service/service.go` | `store *store.Store` → `store store.StoreInterface` |
| `internal/service/calendar.go` | Same concrete→interface change |
| `cmd/server/main.go` | `store.New(...)` → `store.NewSQLite(...)` |
| All service code checking `sql.ErrNoRows` | Switch to `store.ErrNotFound` |
| Test files | `store.New(":memory:")` → `store.NewSQLite(":memory:")` |

### 1.4 Verify

`./tc test backend` — all tests pass, zero behavior change.

**Files touched:** `store/interface.go` (new), `store/sqlite.go`, `service/service.go`, `service/calendar.go`, `cmd/server/main.go`, test files

---

## Phase 2: Firestore Collection Design

Flat root collections (not subcollections) because items and other child entities are sometimes queried independently of their parent trip:

| Collection | Description |
|------------|-------------|
| `trips/{uuid}` | Trip documents |
| `items/{uuid}` | Items (`tripId` field for filtering) |
| `documents/{uuid}` | Documents (optional `tripId` field) |
| `tripLocations/{uuid}` | Per-day locations (`tripId` + `date` fields) |
| `config/{key}` | Key-value config (doc ID = key name) |
| `googleCredentials/{userId}` | OAuth credentials (doc ID = userId) |
| `userCalendars/{userId}_{calendarId}` | Composite ID for natural uniqueness |
| `calendarLinks/{tripId}_{calendarId}_{eventId}` | Composite ID for natural uniqueness |
| `processedEvents/{calendarId}_{eventId}` | Composite ID for natural uniqueness |

### Handling relational features in Firestore

| SQLite Feature | Firestore Approach |
|---------------|-------------------|
| `LIKE '%term%'` (SearchTrips) | Fetch all trips, filter in Go with `strings.Contains`. Fine for ~hundreds of trips. |
| `ON DELETE CASCADE` | `DeleteTrip()` deletes child items/locations/calendarLinks in a batch write |
| `ON DELETE SET NULL` (documents) | `DeleteTrip()` clears `tripId` on associated documents in same batch |
| Unique constraints | Composite document IDs (see above) or check-before-write |
| Transactions (MergeTrips) | Firestore `RunTransaction()` — dataset is small, well within 500-write limit |
| Date range queries | Firestore `Where("startDate", ">=", from)` + in-Go filtering for compound conditions |
| `ORDER BY COALESCE(...)` | Fetch + sort in Go |

**Feature to revisit:** Text search via `SearchTrips`. Current LIKE-based search is fine at small scale with in-memory filtering. If this grows, consider adding a keywords array field for Firestore `array-contains` queries, or an external search service.

---

## Phase 3: Implement Firestore Store

### 3.1 Add dependencies

```
cloud.google.com/go/firestore
firebase.google.com/go/v4
google.golang.org/api
```

### 3.2 Create `store/firestore.go`

Implement `StoreInterface` (~800-1000 lines). Key patterns:

```go
type FirestoreStore struct {
    client *firestore.Client
    ctx    context.Context  // stored at construction, used for all ops
}

func NewFirestore(ctx context.Context, projectID string) (*FirestoreStore, error) {
    // FIRESTORE_EMULATOR_HOST env var auto-detected by SDK
    client, err := firestore.NewClient(ctx, projectID)
    // ...
}
```

**Context approach:** Store `context.Background()` in the struct rather than adding `ctx` to every interface method. This avoids touching all 42 method signatures and every caller. Pragmatic for a single-user app.

### 3.3 Add `firestore:"..."` tags to entity structs

Add alongside existing `db:"..."` tags on all 8 entity files. The `db` tags get removed in Phase 6.

### 3.4 Handle UUID serialization

Firestore stores UUIDs as strings. Add helper methods to convert `uuid.UUID` ↔ `string` for document reads/writes.

**Files touched:** `store/firestore.go` (new), `go.mod`, all `entity/*.go` files

---

## Phase 4: Infrastructure (parallel with Phase 3)

### 4.1 Add Firebase Emulator to `docker-compose.yml`

```yaml
firebase-emulator:
  image: ghcr.io/nicholasgasior/firebase-emulators:latest
  ports:
    - "${PORT_2}:4000"   # Emulator UI
    - "${PORT_3}:8080"   # Firestore
  environment:
    - GOOGLE_CLOUD_PROJECT=travel-calendar-dev

backend:
  environment:
    - FIRESTORE_EMULATOR_HOST=firebase-emulator:8080
    - GOOGLE_CLOUD_PROJECT=travel-calendar-dev
    - STORE_TYPE=firestore
  depends_on:
    firebase-emulator:
      condition: service_healthy
```

### 4.2 Add store selection to `main.go`

Switch on `STORE_TYPE` env var (`"firestore"` or `"sqlite"`). Default to `firestore`.

### 4.3 Update `tc` script

- `db:reset` → clear Firestore emulator via REST: `DELETE /emulator/v1/projects/{id}/databases/(default)/documents`
- `db:shell` → open Emulator UI in browser
- Remove or adapt `db:backup`/`db:restore` (Firestore emulator has export/import)
- Add `emulator:ui` command

### 4.4 Allocate ports via portmanager

4 ports: backend, emulator UI, Firestore emulator, spare.

**Files touched:** `docker-compose.yml`, `tc`, `cmd/server/main.go`, `.env`

---

## Phase 5: Testing

### 5.1 Shared test suite

Create `store/store_test.go` with table-driven tests that run against both implementations:

```go
func runStoreTests(t *testing.T, newStore func(t *testing.T) StoreInterface) { ... }
func TestSQLiteStore(t *testing.T)    { runStoreTests(t, newSQLiteTest) }
func TestFirestoreStore(t *testing.T) { runStoreTests(t, newFirestoreTest) } // skips if no emulator
```

### 5.2 Update `tc test backend`

Add `--with-emulator` flag that starts the Firebase emulator before running tests with `FIRESTORE_EMULATOR_HOST` set.

### 5.3 E2E tests

Run existing E2E tests against the Firestore-backed app to verify end-to-end behavior.

---

## Phase 6: Migration & Cleanup

### 6.1 Data migration script

`scripts/migrate-sqlite-to-firestore.go` — reads SQLite, writes to Firestore. Simple sequential approach, fine for the small dataset.

### 6.2 Remove SQLite (after validation)

- Delete `sqlite.go`, `sqlite_test.go`
- Remove `github.com/mattn/go-sqlite3` (eliminates CGO dependency!)
- Remove `db:"..."` tags from entities
- Update `ARCHITECTURE.md`
- Default `STORE_TYPE` to `firestore`, remove SQLite option

**Bonus:** Removing CGO dependency simplifies Docker builds and enables static binary compilation.

---

## Implementation Order

```
Phase 1 (interface) ──→ Phase 3 (firestore impl) ──┐
                    ──→ Phase 4 (infrastructure)  ──┤
                                                    ├──→ Phase 5 (testing) ──→ Phase 6 (cleanup)
```

Phases 3 and 4 can run in parallel. Each phase is independently committable.

---

## Verification

After each phase:
- `./tc test backend` passes
- `./tc build && ./tc start && ./tc health` succeeds

After Phase 5:
- All CRUD operations work via API
- Trip deletion cascades to items, locations, calendar links
- Trip merge works correctly
- Search returns matching trips
- Google Calendar integration (credentials, calendars, links) works
- E2E tests pass
