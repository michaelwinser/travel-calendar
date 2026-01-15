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
| Trip items (flights, hotels) | Not Started | UC-TRP-003 | Nested entities within trips |
| Document upload | Not Started | - | File storage + association |
| JSON import/export | Not Started | - | Backup/restore capability |

**Next priority**: Trip items (flights, hotels, events) to complete UC-TRP-003

---

## Next Up

### Phase 2: Calendar Integration

**Goal**: Google Calendar sync for conflict detection and trip suggestions

| Feature | Status | Notes |
|---------|--------|-------|
| Google Calendar OAuth | Not Started | Secure token storage needed |
| Read calendar for conflicts | Not Started | Detect home events during trips |
| Suggest trips from calendar | Not Started | Events with locations → trip suggestions |
| Write trips to calendar | Not Started | Optional, dedicated "Travel" calendar |

**Depends on**: Phase 1 MVP completion

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

---

## Use Case Reference

All use cases are defined in [prd/trip-management.md](prd/trip-management.md):

| ID | Description | Status |
|----|-------------|--------|
| UC-TRP-001 | Create a basic trip | Done |
| UC-TRP-002 | List upcoming trips | Done |
| UC-TRP-003 | Get trip with all items | Partial (items not implemented) |
| UC-TRP-004 | Update trip details | Done |
| UC-TRP-005 | Delete trip and all items | Done |
| UC-TRP-006 | Search trips by name or location | Done |
| UC-TRP-007 | Ask LLM about next trip | Done (MCP server) |

---

## How to Use This Roadmap

1. **Pick work from "Current Focus"** - items here are highest priority
2. **Check the PRD** for detailed requirements before implementing
3. **Update status** when completing features
4. **Add new use cases** to relevant PRD files as needed
