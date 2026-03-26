# Calendar Export & Synchronization: Design Document

## Overview

Push travel-calendar data to external calendars and keep them in sync. The app already imports from public iCal feeds (see [import-staging.md](import-staging.md)). This document covers the export direction: serving iCal feeds from the app, writing events to Google Calendar, and maintaining sync state to keep both sides consistent.

Three export models, in order of complexity:

1. **iCal feed** -- the app serves `.ics` endpoints that external calendars subscribe to (pull-based, read-only, no OAuth)
2. **Google Calendar push** -- the app creates/updates/deletes events on the user's Google Calendar via API (write, requires OAuth)
3. **Staged export** -- user selects which activities to push, same curation model as import staging

---

## Principles

1. **Never export what was imported.** Activities with `source != "manual"` are never export candidates. This prevents circular sync where an event imported from Google Calendar gets pushed back to Google Calendar.
2. **Dedicated calendar, not default.** App-created events go to a dedicated "Travel" calendar on Google Calendar. Easy to toggle visibility, easy to bulk delete.
3. **Idempotent sync.** Running push twice produces the same result. Deterministic event IDs from activity keys ensure no duplicates.
4. **User controls what leaves the app.** No automatic export without explicit user action (connect + push or feed subscription).
5. **Clean disconnection.** The user must be able to delete all app-created events from the target calendar in one operation.

---

## Data Model

### SyncTarget entity

Represents a configured export destination for a user. Analogous to ImportSource on the import side.

```
SyncTarget:
  id              string     (UUID, PK)
  userId          string     (index)
  name            string     (user-chosen label, e.g. "My Google Calendar")
  targetType      string     (google_calendar | ical_feed)
  calendarId      string     (Google Calendar ID of the dedicated calendar, empty for ical_feed)
  config          string     (JSON-encoded SyncTargetConfig)
  status          string     (active | paused | error)
  lastSyncAt      string     (RFC3339, empty if never synced)
  lastError       string     (last error message, cleared on success)
  createdAt       string     (RFC3339)
```

Key for dedup: `userId + targetType` (one Google Calendar target per user for now).

### SyncTargetConfig (JSON, stored on SyncTarget)

```json
{
  "windowMonthsBefore": 1,
  "windowMonthsAfter": 6,
  "syncTrips": true,
  "syncActivities": true,
  "activityTypes": ["travel", "stay", "conference", "vacation", "commitment"],
  "tripIds": []
}
```

All fields optional with sensible defaults. Empty `activityTypes` means all types. Empty `tripIds` means all trips.

### SyncRecord entity

Tracks the mapping between a local entity (activity or trip) and its corresponding event on the target calendar.

```
SyncRecord:
  id                string     (UUID, PK)
  userId            string     (index)
  syncTargetId      string     (FK to SyncTarget, index)
  entityType        string     (activity | trip)
  entityId          string     (activity ID or trip ID)
  entityKey         string     (activity key or trip key -- stable across ID changes)
  calendarEventId   string     (Google Calendar event ID)
  syncHash          string     (SHA-256 of synced fields -- detect local changes)
  lastSyncedAt      string     (RFC3339)
  status            string     (synced | pending_delete | deleted | error)
```

Key for dedup: `syncTargetId + entityType + entityId`.

The `syncHash` is computed from the fields that get pushed to the calendar (title, dates, location, type, trip name). When the hash changes, the record needs a re-push. When the hash matches, skip the update.

### Why a separate entity instead of fields on Activity?

- Activities should not know about sync concerns. The Activity struct is already large.
- Sync records need to exist for trips too, not just activities.
- A user could have multiple sync targets in the future.
- Cleanup requires listing all synced events for a target -- an index on `syncTargetId` is cleaner than scanning all activities.

---

## Store Layer

Follows existing patterns in `internal/app/store.go` using `store.Collection[T]`.

### SyncTargetStore

```go
type SyncTargetStore struct {
    coll *store.Collection[SyncTarget]
}
```

Methods:
- `Create(userID, name, targetType, calendarId, config string) (*SyncTarget, error)`
- `Get(id string) (*SyncTarget, error)`
- `List(userID string) ([]SyncTarget, error)`
- `FindByType(userID, targetType string) (*SyncTarget, error)` -- for single-target-per-type lookup
- `Update(t *SyncTarget) error`
- `Delete(id string) error`

### SyncRecordStore

```go
type SyncRecordStore struct {
    coll *store.Collection[SyncRecord]
}
```

Methods:
- `Create(r *SyncRecord) error`
- `Get(id string) (*SyncRecord, error)`
- `FindByEntity(syncTargetId, entityType, entityId string) (*SyncRecord, error)` -- dedup lookup
- `ListByTarget(syncTargetId string) ([]SyncRecord, error)` -- for cleanup
- `ListPendingDelete(syncTargetId string) ([]SyncRecord, error)` -- status=pending_delete
- `Update(r *SyncRecord) error`
- `Delete(id string) error`
- `DeleteByTarget(syncTargetId string) error` -- cascade on target removal

---

## Circular Sync Prevention

The `source` field on Activity (`manual`, `google_calendar`, `system`) is the gate. The export engine applies this rule:

```
exportable(activity) = activity.Source == "manual"
```

Activities imported via the staging system (see [import-staging.md](import-staging.md)) have `source = "google_calendar"` or another non-manual value. They are never included in export queries.

This is enforced at the query level in the sync engine, not at the API level. The engine fetches activities with `source = "manual"` and applies the time window and type filters from SyncTargetConfig.

For trips: all trips are exportable since trips are always user-created. The trip's calendar event uses the trip's computed date range and dominant location from its activities.

---

## Export Model 1: iCal Feed

### How it works

The app serves `.ics` files at URLs that external calendars can subscribe to. Google Calendar, Apple Calendar, and Outlook all support subscribing to iCal URLs. The external calendar polls periodically (typically every few hours).

### Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/shared/{token}/feed.ics` | Token in URL | iCal feed for a share link |
| GET | `/public/{handle}/feed.ics` | None | iCal feed for a public profile |

These build on the existing `/shared/{token}.json` and `/public/{handle}.json` infrastructure. The `.ics` routes return `text/calendar` instead of JSON.

### iCal Generation

Each activity becomes a VEVENT:

```
BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Travel Calendar//EN
CALSCALE:GREGORIAN
METHOD:PUBLISH
X-WR-CALNAME:Travel Calendar - {label}
BEGIN:VEVENT
UID:{activity.key}@travel-calendar
DTSTART;VALUE=DATE:{startDate as YYYYMMDD}
DTEND;VALUE=DATE:{endDate + 1 day as YYYYMMDD}
SUMMARY:{title or "Busy" if showTitle=false}
LOCATION:{location}
TRANSP:TRANSPARENT
CATEGORIES:{type}
X-TRAVEL-CALENDAR-TYPE:{type}
END:VEVENT
...
END:VCALENDAR
```

Notes:
- iCal DTEND for all-day events is exclusive (day after the last day), matching the iCal spec.
- `TRANSP:TRANSPARENT` marks events as free/informational.
- Privacy filtering follows the share link's `showTitle` setting. When false, title is replaced with the activity type label ("Travel", "Conference", etc.).
- UID uses the activity key for stability -- if the activity is updated, the UID stays the same and calendar apps update in place.
- The feed respects the share link's `fromDate`, `toDate`, and `tripIds` filters.

### Trips in the feed

Trips are emitted as separate VEVENTs with:
- UID: `{trip.key}@travel-calendar`
- SUMMARY: trip name (or "Trip" if showTitle=false)
- DTSTART/DTEND: computed from the trip's activity date range
- LOCATION: dominant location from trip activities
- CATEGORIES: `trip`

### Implementation

A new `icalFeed` function in `internal/app/server.go` that:
1. Resolves the share link / public profile (reusing existing lookup logic)
2. Fetches privacy-filtered activities (reusing existing `filteredActivities` logic)
3. Renders activities as VCALENDAR text
4. Sets `Content-Type: text/calendar; charset=utf-8` and `Content-Disposition: attachment; filename="travel.ics"`

Use `github.com/emersion/go-ical` (already a candidate dependency from import-staging) for generation, or generate the text directly since the output format is simple.

---

## Export Model 2: Google Calendar Push

### OAuth Setup

The app already uses Google OAuth for login (via appbase). Calendar export requires additional scopes:

| Scope | Purpose |
|-------|---------|
| `https://www.googleapis.com/auth/calendar` | Create/read/update/delete events and calendars |
| `https://www.googleapis.com/auth/calendar.settings.readonly` | Read Working Location settings (phase 5) |

OAuth token storage (access_token, refresh_token) is available via appbase #27. The sync engine retrieves the user's stored tokens and creates a Google Calendar API client.

### Connect Flow

```
travel sync connect google
```

1. Initiate OAuth flow requesting the `calendar` scope (incremental consent if user already has a session).
2. Store the refresh token via appbase's token storage.
3. Create a secondary calendar named "Travel Calendar" via the Google Calendar API.
4. Store the new calendar's ID in the SyncTarget's `calendarId` field.
5. Create the SyncTarget record with `targetType = "google_calendar"`, `status = "active"`.

The dedicated calendar is created with:
- Summary: "Travel Calendar"
- Description: "Managed by Travel Calendar app. Do not edit events directly."
- Color: a distinguishable color (e.g., Google Calendar color ID 9, blueberry)
- TimeZone: user's primary calendar timezone

### Event Mapping

#### Activity to Google Calendar Event

| Activity Field | Google Calendar Event Field | Notes |
|---|---|---|
| title | summary | |
| startDate | start.date | All-day event |
| endDate + 1 day | end.date | Google uses exclusive end for all-day |
| location | location | Plain string |
| type | extendedProperties.private.travelCalendarType | |
| key | extendedProperties.private.travelCalendarKey | For lookup/dedup |
| -- | transparency | "transparent" (free) |
| -- | visibility | "public" |

#### Trip to Google Calendar Event

| Trip Field | Google Calendar Event Field | Notes |
|---|---|---|
| name | summary | Prefixed with trip emoji or "[Trip]" |
| computed startDate | start.date | Earliest activity in trip |
| computed endDate + 1 day | end.date | Latest activity in trip |
| dominant location | location | Most common location across activities |
| key | extendedProperties.private.travelCalendarKey | |
| -- | transparency | "transparent" |
| -- | colorId | Mapped from trip hex color to nearest Google Calendar color |

#### Trip Status Mapping (future, #107)

| Trip Status | Google Calendar Event Status |
|---|---|
| confirmed | confirmed |
| tentative | tentative |
| planned | tentative |

### Event Identification

Events are identified on Google Calendar by the custom extended property `travelCalendarKey`. This is used to:
- Find existing events during sync (avoid duplicates)
- Identify app-created events during cleanup

Google Calendar also assigns its own event ID, which is stored in SyncRecord.calendarEventId for direct API calls (update, delete).

### Sync Hash

The sync hash determines whether an event needs updating:

```go
func syncHash(title, startDate, endDate, location, actType, tripName string) string {
    h := sha256.New()
    fmt.Fprintf(h, "%s|%s|%s|%s|%s|%s", title, startDate, endDate, location, actType, tripName)
    return hex.EncodeToString(h.Sum(nil))[:16]
}
```

A 16-character hex prefix is sufficient for change detection (not cryptographic security).

### Push Algorithm

```
push(user, syncTarget):
    config = parse(syncTarget.config)
    windowStart = today - config.windowMonthsBefore months
    windowEnd = today + config.windowMonthsAfter months

    // 1. Collect exportable entities
    activities = listActivities(user, from=windowStart, to=windowEnd, source="manual")
    activities = filterByConfig(activities, config)  // type filter, trip filter
    trips = computeTrips(activities)  // unique trips from the activity set

    // 2. Build desired state map: entityKey -> {entity, hash}
    desired = {}
    for each activity: desired[activity.key] = {entity: activity, hash: syncHash(activity)}
    for each trip: desired[trip.key] = {entity: trip, hash: syncHash(trip)}

    // 3. Load current sync records
    records = listSyncRecords(syncTarget.id)
    current = map records by entityKey

    // 4. Create/Update/Delete
    for key, want in desired:
        if key not in current:
            event = createCalendarEvent(want.entity)
            createSyncRecord(syncTarget, key, event.id, want.hash)
        else if current[key].syncHash != want.hash:
            updateCalendarEvent(current[key].calendarEventId, want.entity)
            updateSyncRecord(current[key], want.hash)
        // else: no change, skip

    for key, have in current:
        if key not in desired and have.status == "synced":
            deleteCalendarEvent(have.calendarEventId)
            markSyncRecord(have, status="deleted")

    // 5. Update sync target
    syncTarget.lastSyncAt = now()
    syncTarget.lastError = ""
    updateSyncTarget(syncTarget)
```

The algorithm is idempotent. Interrupted syncs leave some records stale but the next run corrects them.

### Rate Limiting

Google Calendar API has a quota of ~1,000,000 queries/day per project and 500 queries/100 seconds per user. For a typical user with < 100 activities, a full sync is well within limits. No batching or throttling needed initially.

For safety, the push engine should:
- Use batch requests where possible (Google Calendar supports batching up to 50 requests)
- Log API quota usage
- Back off on 429 responses with exponential retry

---

## Export Model 3: Staged Export

For users who want fine-grained control over what gets pushed, the staged export model mirrors import staging.

### How it works

1. Activities appear as export candidates in a staging view (filtered by `source = "manual"` and time window).
2. User selects which activities to include in the next push.
3. Push only sends selected activities.
4. Unselected activities are never pushed.

### Implementation

The SyncRecord entity already tracks what has been synced. The staged export model adds a `pending` state:

- When the user selects activities for export, SyncRecords are created with `status = "pending"`.
- The push command processes records with `status = "pending"` (creates events) and `status = "synced"` where hash changed (updates events).
- Activities with no SyncRecord are shown as "not exported" in the staging view.

This is an enhancement to the base push model, not a replacement. The default behavior (push everything exportable in the time window) works without staging. Staging is opt-in via a config flag on the SyncTarget.

---

## API Endpoints

### Sync Targets

| Method | Endpoint | Operation | Description |
|--------|----------|-----------|-------------|
| GET | `/api/sync/targets` | `listSyncTargets` | List user's export targets |
| POST | `/api/sync/targets` | `createSyncTarget` | Add an export target |
| GET | `/api/sync/targets/{id}` | `getSyncTarget` | Get target details |
| PUT | `/api/sync/targets/{id}` | `updateSyncTarget` | Update config, status |
| DELETE | `/api/sync/targets/{id}` | `deleteSyncTarget` | Remove target and sync records |

### Sync Operations

| Method | Endpoint | Operation | Description |
|--------|----------|-----------|-------------|
| POST | `/api/sync/targets/{id}/push` | `pushSync` | Push changes to target calendar |
| POST | `/api/sync/targets/{id}/cleanup` | `cleanupSync` | Delete all app-created events from target |
| GET | `/api/sync/targets/{id}/status` | `getSyncStatus` | Detailed sync status with record counts |
| GET | `/api/sync/targets/{id}/records` | `listSyncRecords` | List sync records for a target |

### Sync Connect (Google Calendar specific)

| Method | Endpoint | Operation | Description |
|--------|----------|-----------|-------------|
| GET | `/api/sync/connect/google` | `connectGoogle` | Initiate OAuth flow for Calendar scope |
| GET | `/api/sync/connect/google/callback` | `connectGoogleCallback` | OAuth callback, creates target |

### iCal Feed

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/shared/{token}/feed.ics` | Token in URL | iCal feed for share link |
| GET | `/public/{handle}/feed.ics` | None | iCal feed for public profile |

These are not part of the OpenAPI spec (no auth, no JSON) -- they are plain HTTP handlers registered alongside the existing `/shared/{token}.json` and `/public/{handle}.json` routes.

### Request/Response Schemas

**CreateSyncTargetRequest:**
```yaml
required: [name, targetType]
properties:
  name: { type: string, minLength: 1 }
  targetType: { type: string, enum: [google_calendar] }
  config:
    $ref: '#/components/schemas/SyncTargetConfig'
```

**SyncTargetConfig:**
```yaml
properties:
  windowMonthsBefore: { type: integer, default: 1 }
  windowMonthsAfter: { type: integer, default: 6 }
  syncTrips: { type: boolean, default: true }
  syncActivities: { type: boolean, default: true }
  activityTypes:
    type: array
    items: { type: string, enum: [travel, stay, conference, vacation, commitment] }
  tripIds:
    type: array
    items: { type: string }
```

**SyncStatus:**
```yaml
properties:
  targetId: { type: string }
  targetName: { type: string }
  targetType: { type: string }
  status: { type: string }
  lastSyncAt: { type: string }
  lastError: { type: string }
  counts:
    type: object
    properties:
      synced: { type: integer }
      pending: { type: integer }
      pendingDelete: { type: integer }
      error: { type: integer }
```

**PushResult:**
```yaml
properties:
  created: { type: integer }
  updated: { type: integer }
  deleted: { type: integer }
  skipped: { type: integer }
  errors:
    type: array
    items:
      type: object
      properties:
        entityKey: { type: string }
        message: { type: string }
```

---

## CLI Commands

### Sync Management

```
travel sync status
```
Shows all sync targets with their status, last sync time, and record counts. Example output:
```
Sync Targets:
  Google Calendar    active   last sync: 2h ago
    Calendar: Travel Calendar (abc123@group.calendar.google.com)
    Synced: 24 events | Pending: 2 | Errors: 0

  iCal Feed          active   (pull-based, no push needed)
    /shared/abc.../feed.ics
    /public/michael/feed.ics
```

```
travel sync connect google
```
Initiates the Google Calendar OAuth flow. Opens a browser for consent. Creates the dedicated "Travel Calendar" on the user's Google account.

```
travel sync disconnect google
```
Prompts: "Delete all app-created events from Google Calendar? [y/N]". If yes, runs cleanup first. Then removes the SyncTarget and all SyncRecords. Does not delete the dedicated calendar itself (the user can do that manually).

```
travel sync push [--dry-run] [--target google]
```
Pushes changes to connected calendars. With `--dry-run`, prints what would be created/updated/deleted without making API calls. Example:
```
Dry run — no changes will be made.

  CREATE  travel/2026-04-01/london-trip     → "London Trip" Apr 1-5
  UPDATE  trip/fosdem-2026                  → dates changed Feb 1-3
  DELETE  travel/2025-12-01/old-conference  → outside time window
  SKIP    stay/2026-04-01/hotel-marriott   → unchanged

Summary: 1 create, 1 update, 1 delete, 1 skip
```

```
travel sync cleanup [--target google]
```
Deletes all app-created events from the target calendar. Finds events by querying SyncRecords. Also scans the dedicated calendar for events with the `travelCalendarKey` extended property as a safety net.

```
travel sync push --force
```
Ignores sync hashes and re-pushes all events. Useful after a bug fix or schema change.

---

## Working Location (Phase 5)

Google Calendar supports a "Working Location" feature that shows where a person is working from. This is separate from calendar events.

### Scope

Requires `https://www.googleapis.com/auth/calendar.events.owned` scope (already included in the base `calendar` scope).

### Mapping

For each day within the sync window where the user has a `stay` or `conference` activity:
- Set Working Location to the activity's location
- If a Place with timezone is linked, include the office building / custom location details

Working Location events are special all-day events with `eventType: "workingLocation"`. They use the same create/update/delete API as regular events.

### Deferred details

The Working Location API has specific requirements around overlapping working locations and office building IDs. Full design deferred to when this phase is implemented.

---

## iCal Feed Privacy

The iCal feed inherits the privacy model from share links and public profiles:

| Setting | Behavior in Feed |
|---------|-----------------|
| `showTitle = true` | SUMMARY = activity title |
| `showTitle = false` | SUMMARY = activity type label (e.g., "Travel", "Conference") |
| Share link `fromDate`/`toDate` | Only events in range are included |
| Share link `tripIds` | Only events for specified trips |
| Share link `expiresAt` | Feed returns 410 Gone after expiry |

The feed URL itself is the authentication. Anyone with the URL can read the feed. This matches how iCal subscriptions work universally.

---

## Time Window

The sync time window limits what gets pushed, preventing unbounded growth of calendar events.

- Default: 1 month in the past, 6 months in the future (relative to today)
- Configurable per SyncTarget via `windowMonthsBefore` and `windowMonthsAfter`
- Activities outside the window are candidates for deletion from the target calendar
- The window shifts as time passes, so old events are automatically cleaned up on the next push

For the iCal feed, the time window applies to the rendered output. Subscribers see a rolling window of events.

---

## Error Handling

### Recoverable Errors

| Error | Handling |
|-------|----------|
| Google API 429 (rate limit) | Exponential backoff, retry up to 3 times |
| Google API 503 (service unavailable) | Retry with backoff |
| Network timeout | Retry once, then mark target status as "error" |
| Single event create/update fails | Log error on SyncRecord, continue with remaining events |

### Non-Recoverable Errors

| Error | Handling |
|-------|----------|
| OAuth token revoked | Mark target status as "error", set lastError, prompt re-auth on next CLI command |
| Calendar deleted externally | Mark target status as "error", prompt user to reconnect |
| Insufficient scope | Prompt user to reconnect with required scopes |

### SyncRecord Error Tracking

When a single event fails to sync, the SyncRecord's status is set to "error" and the error is logged. The push command reports per-event errors in its output. The next push retries errored records.

---

## Implementation Sequence

### Phase 1: iCal Feed

**No OAuth, no new entities, no external API calls.**

1. Add `GET /shared/{token}/feed.ics` handler -- render activities as VCALENDAR text.
2. Add `GET /public/{handle}/feed.ics` handler -- same logic, different lookup.
3. Reuse existing `filteredActivities` logic for privacy and date filtering.
4. Add trip VEVENTs to the feed.
5. CLI: `travel share` output includes the `.ics` URL for subscription.

Dependencies: none. Can ship independently.

### Phase 2: Sync Entities and Store

1. Add SyncTarget and SyncRecord entities to `internal/app/store.go`.
2. Add SyncTargetStore and SyncRecordStore with methods listed above.
3. Add OpenAPI schemas and endpoints for sync target management.
4. Run codegen.
5. Implement server handlers for CRUD on sync targets.

Dependencies: none beyond current codebase.

### Phase 3: Google Calendar Connect

1. Implement `travel sync connect google` -- OAuth flow with calendar scope.
2. Use appbase #27 token storage for refresh tokens.
3. Create dedicated "Travel Calendar" via Google Calendar API.
4. Store calendar ID on SyncTarget.
5. Implement `travel sync disconnect google`.

Dependencies: Phase 2, appbase #27 (OAuth token storage).

### Phase 4: Push Trips and Activities

1. Implement the push algorithm (create/update/delete events).
2. Implement sync hash computation and comparison.
3. Implement `travel sync push` with `--dry-run` support.
4. Implement `travel sync cleanup`.
5. Implement `travel sync status`.
6. Add Google Calendar API client wrapper (`google.golang.org/api/calendar/v3`).

Dependencies: Phase 3.

### Phase 5: Working Location

1. Add Working Location event creation to the push engine.
2. Map `stay` and `conference` activities to Working Location events.
3. Handle overlapping working locations.

Dependencies: Phase 4, further design needed.

### Phase 6: Bidirectional Sync (future)

1. Detect changes on Google Calendar side (polling or push notifications).
2. Conflict resolution strategy (last-write-wins or prompt user).
3. Flow external changes back through import staging.

Dependencies: Phase 4, import staging ([import-staging.md](import-staging.md)). Separate design document needed.

---

## Out of Scope

- **Outlook/Microsoft 365 push** -- iCal feed covers Outlook subscription. Direct API push deferred.
- **Apple Calendar push** -- iCal feed covers Apple Calendar subscription. No direct API.
- **Automatic periodic push** -- user triggers push manually or via cron. No background scheduler in v1.
- **Bidirectional conflict resolution** -- deferred to Phase 6. This document covers one-way push only.
- **Per-event export selection UI** -- the staged export model (Export Model 3) is described but deferred to after the basic push works.
- **Trip status (#107)** -- trip status mapping to tentative/confirmed events is noted but tracked in its own issue.
- **CalDAV protocol** -- serving a CalDAV endpoint is significantly more complex than iCal feeds. Deferred indefinitely.

---

## Dependencies

| Dependency | Purpose | Phase |
|---|---|---|
| `github.com/emersion/go-ical` | iCal generation (and already needed for import parsing) | 1 |
| `google.golang.org/api/calendar/v3` | Google Calendar API client | 3 |
| `golang.org/x/oauth2/google` | OAuth2 token management | 3 |
| appbase #27 (OAuth token storage) | Store refresh tokens securely | 3 |

---

## Testing Strategy

### Phase 1 (iCal Feed)

- Unit test: render activities as VCALENDAR, verify field mapping, privacy filtering.
- E2E test: create share link, fetch `.ics` URL, parse with go-ical, verify events.
- Manual test: subscribe from Google Calendar and Apple Calendar, verify events appear.

### Phase 2-4 (Google Calendar Push)

- Unit test: sync hash computation, push algorithm logic (mock Google API).
- Integration test: use Google Calendar API test fixture to verify event creation/update/delete.
- E2E test: `travel sync connect google` + `travel sync push --dry-run` end-to-end.
- Manual test: verify events appear on Google Calendar, run cleanup, verify deletion.

### Circular Sync Prevention

- E2E test: import an event from Google Calendar, verify it does not appear in export candidates.
- Unit test: `exportable()` returns false for `source != "manual"`.
