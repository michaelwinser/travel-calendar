# Import Staging: Design Document

## Overview

A staged import system that connects external calendar sources (starting with public iCal feeds) and flows events through a staging area where the user curates them before they become activities. Sources are persistent connections that re-sync, not one-shot imports.

## Principles

1. **Sources are connections, not uploads.** A source persists and re-syncs. The system remembers what it has seen before via source event IDs.
2. **Staging is curation, not approval.** The staging area is where the user selects what matters. Events they ignore stay out of the way but are recoverable.
3. **Filters reduce noise automatically.** Most calendar events are virtual meetings. Built-in heuristics and user-defined keywords keep the staging area focused on travel-relevant items.
4. **Imported events stay linked.** When a staged event becomes an activity, the link is preserved so re-syncs can update titles/dates without creating duplicates.

---

## Data Model

### ImportSource entity

```
ImportSource:
  id            string       (UUID, PK)
  userId        string       (index)
  name          string       (user-chosen label, e.g. "Work Calendar")
  url           string       (iCal feed URL)
  sourceType    string       (ical | json — extensible)
  filterConfig  string       (JSON-encoded FilterConfig)
  lastSyncAt    string       (RFC3339, empty if never synced)
  status        string       (active | paused)
  createdAt     string       (RFC3339)
```

Key for dedup: `userId + url` (a user cannot add the same URL twice).

### StagedEvent entity

```
StagedEvent:
  id              string     (UUID, PK)
  userId          string     (index)
  sourceId        string     (FK to ImportSource, index)
  sourceEventId   string     (iCal UID or equivalent — for dedup/sync)
  title           string
  type            string     (activity type: travel, stay, conference, vacation, commitment)
  startDate       string     (YYYY-MM-DD)
  endDate         string     (YYYY-MM-DD)
  location        string
  notes           string
  state           string     (new | imported | hidden)
  activityId      string     (FK to Activity, set when state = imported)
  createdAt       string     (RFC3339)
  updatedAt       string     (RFC3339)
```

Key for dedup: `sourceId + sourceEventId`. On re-sync, existing staged events are updated in place (title, dates, location) rather than duplicated. State and activityId are preserved across syncs.

### FilterConfig (JSON, stored on ImportSource)

```json
{
  "excludeKeywords": ["standup", "1:1", "sync", "retro"],
  "includeKeywords": ["flight", "hotel", "airbnb", "conference"],
  "disableBuiltinExcludes": false,
  "disableBuiltinIncludes": false
}
```

Empty or null `filterConfig` means "use defaults only."

---

## Store Layer

Follows existing patterns in `internal/app/store.go` using `store.Collection[T]`.

### ImportSourceStore

```go
type ImportSourceStore struct {
    coll *store.Collection[ImportSource]
}
```

Methods:
- `Create(userID, name, url, sourceType, filterConfig string) (*ImportSource, error)`
- `Get(id string) (*ImportSource, error)`
- `List(userID string) ([]ImportSource, error)`
- `Update(s *ImportSource) error`
- `Delete(id string) error`
- `FindByURL(userID, url string) (*ImportSource, error)` -- prevents duplicate sources

### StagedEventStore

```go
type StagedEventStore struct {
    coll *store.Collection[StagedEvent]
}
```

Methods:
- `Create(e *StagedEvent) error`
- `Get(id string) (*StagedEvent, error)`
- `ListBySource(sourceID string) ([]StagedEvent, error)`
- `ListByUser(userID string, stateFilter string) ([]StagedEvent, error)` -- empty stateFilter = all
- `FindBySourceEventID(sourceID, sourceEventID string) (*StagedEvent, error)` -- for dedup
- `Update(e *StagedEvent) error`
- `Delete(id string) error`
- `DeleteBySource(sourceID string) error` -- cascade on source removal

---

## Filter System

Filters run server-side during sync. They decide which parsed iCal events get staged and which are silently dropped.

### Evaluation Order

For each parsed calendar event:

1. **Built-in excludes** (unless disabled): if event matches an exclude pattern, mark excluded.
2. **User exclude keywords**: if title or location contains any keyword, mark excluded.
3. **Built-in includes** (unless disabled): if event matches an include pattern, mark included.
4. **User include keywords**: if title or location contains any keyword, mark included.
5. **Default rule**: if the event has a non-empty location string (after stripping video URLs), include it. Otherwise, exclude it.

Include overrides exclude. This means a "flight" keyword match keeps the event even if it would otherwise be excluded by a built-in pattern.

### Built-in Exclude Patterns

Applied by default (toggle with `disableBuiltinExcludes`):

| Pattern | Matches |
|---------|---------|
| Video conference URLs in location | `zoom.us`, `meet.google.com`, `teams.microsoft.com`, `webex.com` |
| Meeting room names in location | "Meeting Room", "Conf Room", "Phone Booth", "Room" + digit |
| Zero-duration events | Events where start == end and not all-day |

### Built-in Include Patterns

Applied by default (toggle with `disableBuiltinIncludes`):

| Pattern | Effect |
|---------|--------|
| All-day events | Always stage (regardless of other filters) |
| Multi-day events (end > start + 1 day) | Always stage |
| Events with physical location (non-URL, non-room) | Stage |

### Type Inference

When staging events, the system infers an activity type from the event content:

| Signal | Inferred Type |
|--------|---------------|
| Title contains "flight", "fly" | `travel` |
| Title contains "hotel", "airbnb", "stay" | `stay` |
| Title contains "conference", "summit", "expo" | `conference` |
| All-day, multi-day, location present | `vacation` (default for multi-day) |
| Single day with location | `commitment` (default) |

The user can change the type when reviewing staged events or after import.

---

## API Endpoints

### Sources

| Method | Endpoint | Operation | Description |
|--------|----------|-----------|-------------|
| GET | `/api/sources` | `listSources` | List user's import sources |
| POST | `/api/sources` | `createSource` | Add a new import source |
| GET | `/api/sources/{id}` | `getSource` | Get source details |
| PUT | `/api/sources/{id}` | `updateSource` | Update name, filter config, status |
| DELETE | `/api/sources/{id}` | `deleteSource` | Remove source and its staged events |
| POST | `/api/sources/{id}/sync` | `syncSource` | Trigger a sync (fetch + filter + stage) |

### Staged Events

| Method | Endpoint | Operation | Description |
|--------|----------|-----------|-------------|
| GET | `/api/staged` | `listStagedEvents` | List staged events (query: sourceId, state) |
| POST | `/api/staged/import` | `importStagedEvents` | Import selected staged events as activities |
| POST | `/api/staged/hide` | `hideStagedEvents` | Hide (dismiss) selected staged events |
| PUT | `/api/staged/{id}` | `updateStagedEvent` | Edit a staged event before importing |

### Request/Response Schemas

**CreateSourceRequest:**
```yaml
required: [name, url]
properties:
  name: { type: string, minLength: 1 }
  url: { type: string, format: uri }
  sourceType: { type: string, enum: [ical], default: ical }
  filterConfig:
    $ref: '#/components/schemas/FilterConfig'
```

**FilterConfig:**
```yaml
properties:
  excludeKeywords: { type: array, items: { type: string } }
  includeKeywords: { type: array, items: { type: string } }
  disableBuiltinExcludes: { type: boolean }
  disableBuiltinIncludes: { type: boolean }
```

**ImportStagedEventsRequest:**
```yaml
required: [ids]
properties:
  ids:
    type: array
    items: { type: string }
    description: Staged event IDs to import as activities
```

**HideStagedEventsRequest:**
```yaml
required: [ids]
properties:
  ids:
    type: array
    items: { type: string }
```

---

## CLI Commands

### Source Management

```
travel source add <name> <url> [--type ical]
```
Creates an import source and triggers the first sync.

```
travel source list
```
Lists sources with name, URL, status, last sync time, and staged event counts.

```
travel source sync [name]
```
Re-fetches from the named source (or all active sources if no name given). Prints count of new/updated/unchanged staged events.

```
travel source remove <name>
```
Removes the source and all its staged events. Imported activities are not affected.

```
travel source pause <name>
travel source resume <name>
```
Toggles source status between active and paused.

### Staging

```
travel staged [--source name] [--state new|imported|hidden]
```
Lists staged events. Default filter: `--state new` (show unprocessed events).

```
travel staged import <id-prefix...>
```
Imports one or more staged events as real activities. Uses ID prefix matching (same pattern as `travel delete`).

```
travel staged import --all [--source name]
```
Imports all "new" staged events (optionally filtered by source).

```
travel staged hide <id-prefix...>
```
Marks staged events as hidden (dismissed). They remain in the database for dedup but are hidden from the default listing.

```
travel staged unhide <id-prefix...>
```
Restores hidden events back to "new" state.

---

## Sync Behavior

### First Sync

1. Fetch the iCal feed from the source URL.
2. Parse all VEVENT entries.
3. Apply filter rules to each event.
4. For events that pass filters, create StagedEvent records with `state: new`.
5. Update `lastSyncAt` on the source.

### Re-Sync

1. Fetch the iCal feed.
2. Parse all VEVENT entries.
3. For each event:
   - Look up existing StagedEvent by `sourceId + sourceEventId`.
   - If found: update title, dates, location, notes, updatedAt. Preserve state and activityId.
   - If not found: apply filters, create new StagedEvent if it passes.
4. Events that were previously staged but no longer appear in the feed are left as-is (not deleted). The calendar may have a limited time window.
5. Update `lastSyncAt`.

### Imported Event Updates

When a re-sync updates a staged event that has `state: imported`:
- The staged event fields are updated.
- The linked activity is NOT automatically updated (the user chose to import it and may have edited it).
- Future enhancement: show a "source changed" indicator in the UI so the user can manually re-sync individual imported events.

---

## UI

### Source Management View

Accessible from the main navigation (new "Sources" tab or within settings).

```
+--------------------------------------------------+
| Import Sources                          [+ Add]  |
+--------------------------------------------------+
| Work Calendar          active   synced 2h ago    |
|   https://cal.google.com/...ical                 |
|   12 new | 8 imported | 3 hidden                |
|   [Sync Now]  [Edit Filters]  [Pause]  [Remove]  |
+--------------------------------------------------+
| Conference Schedule    paused                     |
|   https://fosdem.org/2027/schedule/ical           |
|   0 new | 15 imported                             |
|   [Sync Now]  [Edit Filters]  [Resume] [Remove]  |
+--------------------------------------------------+
```

**Add Source modal/form:**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| Name | text | Yes | 1-100 chars |
| URL | url | Yes | Valid URL |
| Type | select | No | ical (default) |

### Staging View

Per-source or combined view of staged events.

```
+------------------------------------------------------------------+
| Staged Events: Work Calendar            [Import Selected] [Hide] |
+------------------------------------------------------------------+
| [ ] NEW  Flight to London         Apr 12-12   London, UK         |
| [ ] NEW  FOSDEM 2027              Feb 1-2     Brussels           |
| [ ] NEW  Team Offsite             Mar 5-7     Austin, TX         |
| [x] IMPORTED  Client Visit Tokyo  Jan 15-20   Tokyo              |
|     HIDDEN    Dentist              Apr 3       123 Main St        |
+------------------------------------------------------------------+
| Showing: new (3) | imported (1) | hidden (1)                     |
| [Select All New]                                                  |
+------------------------------------------------------------------+
```

Visual treatment:
- **New**: bold text, checkbox enabled
- **Imported**: muted text, no checkbox, linked activity ID shown
- **Hidden**: strikethrough, expandable to unhide

### Filter Configuration

Inline or modal editor per source:

```
+--------------------------------------------------+
| Filter Rules: Work Calendar                      |
+--------------------------------------------------+
| Built-in excludes: [x] enabled                   |
|   Video conference URLs, meeting rooms            |
|                                                   |
| Built-in includes: [x] enabled                   |
|   All-day events, multi-day, physical locations   |
|                                                   |
| Include keywords (comma-separated):               |
| [flight, hotel, conference, dentist, doctor     ] |
|                                                   |
| Exclude keywords (comma-separated):               |
| [standup, 1:1, sync, retro, sprint planning     ] |
|                                                   |
|                        [Save]  [Cancel]           |
+--------------------------------------------------+
```

---

## iCal Parsing

Use a Go iCal library (e.g., `github.com/emersion/go-ical`) to parse VCALENDAR/VEVENT data.

### Field Mapping

| iCal Field | StagedEvent Field | Notes |
|------------|-------------------|-------|
| UID | sourceEventId | Primary dedup key |
| SUMMARY | title | |
| DTSTART | startDate | Convert to YYYY-MM-DD |
| DTEND | endDate | Convert to YYYY-MM-DD, subtract 1 day for all-day events (iCal uses exclusive end) |
| LOCATION | location | Strip video conference URLs before storing |
| DESCRIPTION | notes | Truncate to reasonable length |

### All-Day Event Detection

An iCal event is all-day if DTSTART is a DATE (not DATE-TIME). For all-day events, DTEND is exclusive, so the stored endDate = DTEND - 1 day.

---

## Implementation Sequence

1. **Entities and store** -- ImportSource + StagedEvent structs, store methods, OpenAPI schemas, codegen.
2. **Source management CLI** -- add/list/sync/remove commands using the generated client.
3. **Staging CLI** -- list/import/hide commands.
4. **Auto-filtering** -- built-in exclude/include patterns, filter evaluation logic.
5. **Source management UI** -- frontend source list, add/edit/remove.
6. **Staging UI** -- frontend staging review, checkboxes, bulk import/hide.
7. **Keyword filtering** -- user-defined include/exclude keywords per source.

---

## Out of Scope

- **Authenticated calendar sources** (Google Calendar OAuth, CalDAV auth) -- deferred to a future iteration. This design covers public iCal URLs only.
- **Automatic periodic sync** -- the user (or a future cron) triggers syncs manually. No background scheduler in v1.
- **Bidirectional sync** -- changes to imported activities do not flow back to the source calendar.
- **Push notifications** -- no alerts when new events appear in staging.
- **Bulk edit of staged events** -- users can edit individual staged events before import but not bulk-edit fields across multiple events.
