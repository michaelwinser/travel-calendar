# Product Roadmap

Travel Calendar helps you see **where you'll be** across time. A trip is fundamentally a location + dates. Calendar events, confirmations, and other details are context that surfaces alongside trips — not the trips themselves.

---

## Current Focus

### Phase 5: Location-Centric Trips

**Goal**: Refocus the app on location awareness. A trip = where + when.

#### 5A: UX Quick Wins

| Feature | Status | Notes |
|---------|--------|-------|
| Month calendar view | In Progress | 7-column week grid (Sun–Sat), trip bars spanning days, endless scroll of months |
| "New Trip" button on every page | Done | Moved to global layout header |
| Trip creation returns to calendar | Done | Redirects to /calendar instead of trip detail |
| Location as primary trip field | Done | Added to OpenAPI, backend, both stores, and trip form UI |
| Fix trip purpose selector feedback | Done | Explicit Tailwind class mappings instead of dynamic interpolation |
| Auto-detect OAuth redirect URL | Not Started | Infer from request Host/protocol instead of requiring GOOGLE_REDIRECT_URL env var |

#### 5B: Quick Entry

| Feature | Status | Notes |
|---------|--------|-------|
| Free-text trip creation | Done | QuickEntry component on calendar page; parses "Milan Jan 23-27", "London next week business" |
| Natural language dates | Done | Date fields accept "Jan 23", "next Tuesday", "tomorrow"; converts on blur |

#### 5C: Related Items Panel

| Feature | Status | Notes |
|---------|--------|-------|
| Calendar events panel on trip | Not Started | Show Google Calendar events that fall within trip dates, as context |
| Location conflict detection | Not Started | Flag home-area appointments during travel (e.g. dentist while in Milan) |
| Promote event to trip item | Not Started | Optional — user can add a related event as a trip item if desired |

---

## Next

### Phase 6: Account & Data Management

| Feature | Status | Notes |
|---------|--------|-------|
| User-controlled account deletion | Not Started | "Delete my data" in Settings |
| `./tc admin:delete-user <email>` | Not Started | Admin command to remove all user data |
| `./tc admin:cleanup-orphans` | Not Started | Purge pre-multi-tenancy data with empty user_id |

### Phase 7: Sharing & Collaboration

| Feature | Status | Notes |
|---------|--------|-------|
| Share trip visibility | Not Started | Let contacts see where you'll be (ties into whereish concepts) |
| Write trips to calendar | Not Started | Optional sync to a dedicated "Travel" calendar |

---

## Parked

Features built or planned that are deprioritized pending the location-centric refocus.

| Feature | Status | Reason Parked |
|---------|--------|---------------|
| Trip items (flights, hotels, trains) | Built | Overbuilt for primary use case; may resurface via related items panel |
| Trip import from calendar (as items) | Built | Created too much detail; replaced by related items approach |
| TripIt event parsing | Built | Specialized; not core to location awareness |
| Merge candidate detection | Built | Complexity for calendar import flow; less relevant with simpler trips |
| Trip organization (merge, move, bulk) | Partially built | Premature; revisit if item management returns |
| Document upload | Not started | No clear use case yet |
| Document intelligence (email, PDF) | Not started | Premature |
| Expense tracking | Not started | Not core to "where will I be" |
| JSON import/export | Not started | Low priority |

---

## Completed

### Infrastructure & Deployment (Jan–Mar 2026)
- Go monorepo: backend, CLI, MCP server
- OpenAPI 3.1 spec as source of truth
- TypeScript types generated from OpenAPI
- Docker dev environment via `./tc` script
- SvelteKit frontend embedded in Go binary
- Cloud Run deployment (`./tc deploy`, `./tc provision`)
- Firestore + SQLite dual-store pattern

### Trip Management MVP (Jan 2026)
- Trip CRUD with REST API
- Year calendar visualization
- Trip items (flights, hotels, trains, drives, events)
- Trip search, MCP tools
- Pre-commit checks and test pipeline

### Google Calendar Integration (Jan–Mar 2026)
- OAuth 2.0 with calendar permissions
- Calendar selection for monitoring
- Trip suggestions from calendar events
- TripIt parsing, merge candidates, event filtering

### Authentication & Multi-Tenancy (Mar 2026)
- Google OAuth login (single consent for auth + calendar)
- Session-based auth with cookie middleware
- User-scoped data isolation (trips, credentials, processed events)
- ALLOWED_USERS env var for access control

---

## Design Principles

1. **A trip is a location + dates** — not an itinerary
2. **Calendar events are context** — they surface alongside trips, not as trips
3. **Quick entry over forms** — typing "Milan Jan 23" should just work
4. **Location conflicts are the key insight** — "you have a dentist appointment while you're in Milan"
5. **Start simple, layer detail** — users can optionally add items; the app doesn't force it
