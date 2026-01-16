# Product Roadmap

This roadmap tracks implementation progress for the Travel Calendar application. It references [PRD.md](PRD.md) for overall vision and [prd/*.md](prd/) for detailed use cases.

---

## Current Focus

### Phase 1: Core MVP

**Goal**: Basic trip CRUD with web UI and MCP access

| Feature | Status | Use Cases | Notes |
|---------|--------|-----------|-------|
| Data model & SQLite storage | Done | - | Go backend with entities |
| REST API (trips CRUD) | Done | UC-TRP-001 to UC-TRP-005 | OpenAPI-generated types |
| Web UI: Trip list | Done | UC-TRP-002 | SvelteKit with calendar bars |
| Web UI: Trip create/edit | Done | UC-TRP-001, UC-TRP-004 | Form with validation |
| Web UI: Trip delete | Done | UC-TRP-005 | With confirmation |
| MCP server: Trip tools | Done | UC-TRP-007 | Go implementation |
| Trip search | Done | UC-TRP-006 | Full-text search on name/notes |
| Trip items (flights, hotels) | Done | UC-TRP-003 | Flights, hotels, trains, drives, events |
| Document upload | Not Started | - | File storage + association (deferred) |
| JSON import/export | Not Started | - | Backup/restore capability |

**Next priority**: Phase 2C Trip Organization (document upload deferred)

---

## Current Focus

### Phase 2: Calendar Integration

**Goal**: Google Calendar sync for conflict detection and trip suggestions

#### Phase 2A: OAuth Foundation (DONE)

| Feature | Status | Notes |
|---------|--------|-------|
| Google Calendar OAuth | Done | OAuth flow, settings UI, calendar selection |

#### Phase 2B: Calendar Trip Intelligence (DONE)

**Goal**: Import trips with items, TripIt support, smart suggestions

| Feature | Status | Use Cases | Notes |
|---------|--------|-----------|-------|
| Suggest trips from calendar | Done | - | Basic suggestions with filtering |
| Filter virtual meetings | Done | UC-CAL-010 | URLs and meeting rooms excluded (#28) |
| Import trips with items | Done | UC-CAL-001 | Calendar events → trip items with preview |
| TripIt event parsing | Done | UC-CAL-002, UC-CAL-003 | Parse TripIt calendar format |
| Event type classification | Done | UC-CAL-006 | All-day → trips, timed → items |
| Merge candidate detection | Done | UC-CAL-005 | Show similar existing trips |
| Merge suggestion into trip | Done | UC-CAL-004 | Merge dropdown in suggestions UI |
| Remember processed events | Done | UC-CAL-011 | Dismiss button + reset in settings |

**PRD**: [prd/calendar-trip-intelligence.md](prd/calendar-trip-intelligence.md)

#### Phase 2C: Trip Organization (LATER)

**Goal**: Reorganize trips and items after creation

| Feature | Status | Use Cases | Notes |
|---------|--------|-----------|-------|
| Merge existing trips | Done | UC-ORG-001 | Combine two trips into one |
| Convert trip to item | Not Started | UC-ORG-002 | Day trip → event on larger trip |
| Move item between trips | Done | UC-ORG-003 | Reassign items |
| Create trip from item(s) | Not Started | UC-ORG-004 | Split trip workflow |
| Bulk move items | Not Started | UC-ORG-005 | Multi-select operations |

**PRD**: [prd/trip-organization.md](prd/trip-organization.md)

#### Phase 2D: Additional Calendar Features (LATER)

| Feature | Status | Notes |
|---------|--------|-------|
| Read calendar for conflicts | Not Started | Detect home events during trips |
| Write trips to calendar | Not Started | Optional, dedicated "Travel" calendar |

---

## Later

### Phase 3: Document Intelligence

**Goal**: Automated document capture and parsing

| Feature | Status | Notes |
|---------|--------|-------|
| Email forwarding capture | Not Started | Dedicated address for forwarding |
| PDF parsing | Not Started | Extract confirmation numbers, amounts |
| Auto-association | Not Started | Match documents to trips by date/location |
| Receipt/expense extraction | Not Started | OCR or structured parsing |

### Phase 4: Advanced Features

**Goal**: Power features for frequent travelers

| Feature | Status | Notes |
|---------|--------|-------|
| Expense tracking & reporting | Not Started | Budget categories, totals |
| Multi-device sync | Not Started | Cloud backend option |
| Sharing/collaboration | Not Started | Family trips, shared docs |
| Mobile app or PWA | Not Started | Responsive UI first |

---

## Completed Milestones

### Infrastructure Foundation (Jan 2026)
- Go monorepo with backend, CLI, MCP server
- OpenAPI 3.1 spec as source of truth
- TypeScript types generated from OpenAPI
- Docker development environment via `tc` script
- SvelteKit frontend embedded in Go binary
- Commit workflow with code review
- Pre-commit checks and CI pipeline

### Trip Management MVP (Jan 2026)
- Trip CRUD operations (create, read, update, delete)
- Trip list with calendar visualization
- Trip editing UI with form validation
- Backend unit tests for store, service, and handler layers

### Trip Items Feature (Jan 2026)
- Items CRUD (flights, hotels, trains, drives, events)
- Add Item UI with type-specific forms
- Item timeline grouped by date
- MCP tools for item management
- E2E and browser tests for UC-TRP-003

### Google Calendar OAuth Foundation (Jan 2026)
- OAuth 2.0 flow with Google Calendar API
- Settings page with connect/disconnect UI
- Calendar selection for monitoring
- Database schema for credentials and calendar links
- Multi-user ready architecture

---

## Use Case Reference

### Trip Management ([prd/trip-management.md](prd/trip-management.md))

| ID | Description | Status |
|----|-------------|--------|
| UC-TRP-001 | Create a basic trip | Done |
| UC-TRP-002 | List upcoming trips | Done |
| UC-TRP-003 | Get trip with all items | Done |
| UC-TRP-004 | Update trip details | Done |
| UC-TRP-005 | Delete trip and all items | Done |
| UC-TRP-006 | Search trips by name or location | Done |
| UC-TRP-007 | Ask LLM about next trip | Done (MCP server) |

### Calendar Trip Intelligence ([prd/calendar-trip-intelligence.md](prd/calendar-trip-intelligence.md))

| ID | Description | Status |
|----|-------------|--------|
| UC-CAL-001 | Import trip with travel items | Done |
| UC-CAL-002 | Parse TripIt all-day summary event | Done |
| UC-CAL-003 | Parse TripIt flight segment event | Done |
| UC-CAL-004 | Merge imported trip with existing trip | Done |
| UC-CAL-005 | Detect merge candidates for trip suggestion | Done |
| UC-CAL-006 | Distinguish all-day events vs timed events | Done |
| UC-CAL-007 | Create nested/related trips | Not Started |
| UC-CAL-008 | Show related trips in UI | Not Started |
| UC-CAL-009 | Merge two existing trips | Done (via UC-ORG-001) |
| UC-CAL-010 | Filter virtual meetings from suggestions | Done |
| UC-CAL-011 | Remember processed calendar events | Done |

### Trip Organization ([prd/trip-organization.md](prd/trip-organization.md))

| ID | Description | Status |
|----|-------------|--------|
| UC-ORG-001 | Merge two existing trips | Done |
| UC-ORG-002 | Convert trip to item on another trip | Not Started |
| UC-ORG-003 | Move item to another trip | Done |
| UC-ORG-004 | Create trip from item(s) | Not Started |
| UC-ORG-005 | Bulk move items between trips | Not Started |

---

## How to Use This Roadmap

1. **Pick work from "Current Focus"** - items here are highest priority
2. **Check the PRD** for detailed requirements before implementing
3. **Update status** when completing features
4. **Add new use cases** to relevant PRD files as needed
