# Project Map

> **Auto-validated**: Run `pnpm validate:map` to check this document against the codebase.
> **Last validated**: (not yet)

This is the authoritative reference for what exists in this project. Claude Code should consult this file to understand component responsibilities and locate code.

---

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         TRAVEL CALENDAR                                      │
│                                                                              │
│  A trip management system with two interfaces:                              │
│  • MCP Server → LLM queries ("When is my next trip?")                       │
│  • Web UI → Visual calendar, trip editing                                   │
│                                                                              │
│  Core entity: TRIP (contains ITEMS like flights, hotels, events)            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Components

### `backend` - REST API

| Responsibility | Location |
|---------------|----------|
| HTTP server | `src/index.ts` |
| Trip CRUD | `src/routes/trips.ts` |
| Item CRUD | `src/routes/items.ts` |
| Document CRUD | `src/routes/documents.ts` |
| Business logic | `src/services/*.ts` |
| Database schema | `src/entities/*.ts` |
| Validation | `src/lib/validation.ts` |

**Key patterns**:
- Routes are thin, services are thick
- All input validated with Zod
- Entities are database-only (no business logic)

**Dependencies**: Hono, Drizzle, better-sqlite3, Zod

---

### `frontend` - SvelteKit Web UI

| Responsibility | Location |
|---------------|----------|
| Route pages | `src/routes/` |
| Reactive stores | `src/lib/stores/*.ts` |
| Trip components | `src/lib/components/trip/` |
| Item components | `src/lib/components/item/` |
| Calendar views | `src/lib/components/calendar/` |
| API client | `src/lib/api/client.ts` |

**Key patterns**:
- Components receive data via props (no ID lookups)
- Stores are single source of truth
- Multiple view variants per resource (Card, Chip, Detail)

**Dependencies**: SvelteKit, Tailwind CSS

---

### `mcp-server` - LLM Interface

| Responsibility | Location |
|---------------|----------|
| MCP server setup | `src/index.ts` |
| Trip tools | `src/tools/trips.ts` |
| Item tools | `src/tools/items.ts` |
| Document tools | `src/tools/documents.ts` |
| Backend client | `src/lib/backend-client.ts` |
| LLM formatters | `src/lib/formatters.ts` |

**Key patterns**:
- Facade over backend API (no direct DB)
- Responses formatted for LLM consumption
- Tools are verb-oriented (`get_trips`, not `trips`)

**Dependencies**: @modelcontextprotocol/sdk

---

### `shared` - TypeScript Types

| Responsibility | Location |
|---------------|----------|
| Trip types | `src/trip.ts` |
| Item types | `src/item.ts` |
| Document types | `src/document.ts` |
| API types | `src/api.ts` |

**Key patterns**:
- Types only (no runtime code)
- No dependencies
- Mirrors backend entities

---

### `infra` - Infrastructure & Containers

| Responsibility | Location |
|---------------|----------|
| Development Dockerfile | `Dockerfile.dev` |
| Service orchestration | `docker-compose.yml` |
| Build exclusions | `.dockerignore` |
| Data persistence | `data/` |

**Key patterns**:
- All development is container-based
- Named volumes for node_modules (macOS performance)
- Bind mounts for source code (hot reload)
- Profiles for optional services (mcp, test)

**Dependencies**: Docker, Docker Compose

---

### `cli` - Command Line Interface

| Responsibility | Location |
|---------------|----------|
| Trip commands | `cli/travel` |
| Test automation | Used by `tests/e2e/*.sh` |

**Key patterns**:
- Bash script wrapping REST API
- `--json` flag for machine-readable output
- Used for E2E testing

---

## Data Flow

```
User/LLM Request
       │
       ▼
┌──────────────────┐     ┌──────────────────┐
│   MCP Server     │     │   Frontend       │
│   (tools)        │     │   (stores)       │
└────────┬─────────┘     └────────┬─────────┘
         │                        │
         │     HTTP/REST          │
         └──────────┬─────────────┘
                    ▼
         ┌──────────────────┐
         │     Backend      │
         │   (services)     │
         └────────┬─────────┘
                  │
                  ▼
         ┌──────────────────┐
         │     SQLite       │
         │   (entities)     │
         └──────────────────┘
```

---

## Lexicon

Precise terminology for this project. **Use these terms consistently.**

| Term | Definition | NOT |
|------|------------|-----|
| **Trip** | A travel event with dates, purpose, and items. The primary entity. | Journey, vacation, travel |
| **Item** | Something within a trip: flight, hotel, train, drive, event. Has a date. | Segment, leg, component |
| **Flight** | An Item of type "flight" with origin/destination airports. | Plane, air travel |
| **Hotel** | An Item of type "hotel" with check-in/check-out dates. | Lodging, accommodation, stay |
| **Document** | A file (PDF, image) attached to a trip or item. | Attachment, file, receipt |
| **Purpose** | Trip category: conference, work, vacation, family, personal. | Type, kind, category |
| **Status** | Trip state: planned, confirmed, completed, cancelled. | State, phase |
| **Store** | Svelte reactive store. Single source of truth for a resource type. | State, cache, model |
| **Tool** | MCP tool callable by LLM. Verb-oriented action. | Function, endpoint, command |
| **Resource** | MCP resource readable by LLM. Data endpoint. | Endpoint, API |
| **Component** | Svelte UI component OR project package (context-dependent). | Widget, module |
| **Service** | Backend class containing business logic for an entity. | Controller, handler |
| **Entity** | Database table definition (Drizzle schema). | Model, schema, table |

---

## File Naming Conventions

| Pattern | Example | Used For |
|---------|---------|----------|
| `{resource}.ts` | `trip.ts` | Entity, service, or type file |
| `{Resource}Card.svelte` | `TripCard.svelte` | List item view |
| `{Resource}Chip.svelte` | `TripChip.svelte` | Inline reference |
| `{Resource}Detail.svelte` | `TripDetail.svelte` | Full page view |
| `{Resource}Form.svelte` | `TripForm.svelte` | Create/edit form |
| `{resource}.test.ts` | `trip.test.ts` | Unit tests |
| `test-{feature}.sh` | `test-trip-crud.sh` | E2E test script |

---

## API Endpoints

| Method | Endpoint | Returns |
|--------|----------|---------|
| `GET` | `/api/trips` | `Trip[]` |
| `POST` | `/api/trips` | `Trip` |
| `GET` | `/api/trips/:id` | `TripWithItems` |
| `PATCH` | `/api/trips/:id` | `Trip` |
| `DELETE` | `/api/trips/:id` | `void` |
| `GET` | `/api/trips/:id/items` | `Item[]` |
| `POST` | `/api/trips/:id/items` | `Item` |
| `DELETE` | `/api/items/:id` | `void` |
| `GET` | `/api/documents` | `Document[]` |
| `POST` | `/api/documents` | `Document` |

---

## MCP Tools

| Tool | Purpose |
|------|---------|
| `get_trips` | List/filter trips |
| `get_trip` | Get trip with items |
| `create_trip` | Create new trip |
| `update_trip` | Update trip |
| `delete_trip` | Delete trip |
| `search_trips` | Full-text search |
| `get_documents` | List documents |
| `get_calendar_conflicts` | Find conflicting calendar events |

---

## Validation

To validate this map against the codebase:

```bash
pnpm validate:map
```

This checks:
- All listed files exist
- All endpoints are implemented
- All tools are registered
- Lexicon terms appear in code comments

**If validation fails**, either:
1. Update the code to match the map, or
2. Update the map to match the code
