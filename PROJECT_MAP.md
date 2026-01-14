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
│  A trip management system with three interfaces:                            │
│  • CLI → Command-line trip management                                       │
│  • MCP Server → LLM queries ("When is my next trip?")                       │
│  • Web UI → Visual calendar, trip editing                                   │
│                                                                              │
│  Core entity: TRIP (contains ITEMS like flights, hotels, events)            │
│                                                                              │
│  Architecture: OpenAPI-first (single source of truth for types)             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Components

### `api` - OpenAPI Specification

| Responsibility | Location |
|---------------|----------|
| API specification | `packages/api/openapi.yaml` |

**Key patterns**:
- Single source of truth for all types
- Generates Go server types (backend)
- Generates Go client types (CLI)
- Generates TypeScript types (frontend)

---

### `backend` - REST API (Go)

| Responsibility | Location |
|---------------|----------|
| HTTP server | `cmd/server/main.go` |
| Generated types | `internal/api/openapi.gen.go` |
| Trip entity | `internal/entity/trip.go` |
| Item entity | `internal/entity/item.go` |
| Document entity | `internal/entity/document.go` |
| Database layer | `internal/store/sqlite.go` |
| Business logic | `internal/service/service.go` |
| HTTP handlers | `internal/handler/handler.go` |

**Key patterns**:
- OpenAPI-first: handlers implement generated ServerInterface
- Layered: Handler → Service → Store → Entity
- Entities are internal, converted to/from API types
- All input validated at handler level

**Dependencies**: Chi router, go-sqlite3, google/uuid, oapi-codegen

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

### `mcp-server` - LLM Interface (Go)

| Responsibility | Location |
|---------------|----------|
| HTTP server | `cmd/mcp/main.go` |
| JSON-RPC handler | `internal/handler/mcp.go` |
| Tool registry | `internal/tools/registry.go` |
| Trip tools | `internal/tools/trips.go` |
| Document tools | `internal/tools/documents.go` |
| LLM formatters | `internal/formatter/markdown.go` |

**Key patterns**:
- HTTP Streamable transport (JSON-RPC 2.0 over HTTP)
- Facade over backend API (no direct DB)
- Responses formatted as markdown for LLM consumption
- Tools are verb-oriented (`get_trips`, not `trips`)

**Dependencies**: Standard library only

---

### `cli` - Command Line Interface (Go)

| Responsibility | Location |
|---------------|----------|
| Entry point | `cmd/travel/main.go` |
| Generated client | `internal/client/client.gen.go` |
| Root command | `internal/cmd/root.go` |
| Trips commands | `internal/cmd/trips.go` |
| Items commands | `internal/cmd/items.go` |
| Documents commands | `internal/cmd/documents.go` |
| Output formatting | `internal/output/output.go` |

**Key patterns**:
- OpenAPI-generated HTTP client
- Cobra command hierarchy
- `--json` flag for machine-readable output
- Environment variable configuration (TRAVEL_API_URL)

**Dependencies**: Cobra, Viper, oapi-codegen

---

### `shared` - TypeScript Types

| Responsibility | Location |
|---------------|----------|
| Generated types | `src/api.ts` |
| Type aliases | `src/index.ts` |

**Key patterns**:
- Types generated from OpenAPI spec
- No runtime code
- No dependencies (except openapi-typescript for generation)

---

### `infra` - Infrastructure & Containers

| Responsibility | Location |
|---------------|----------|
| Go development Dockerfile | `Dockerfile.go` |
| Service orchestration | `docker-compose.yml` |
| Build exclusions | `.dockerignore` |
| Data persistence | `data/` |
| Helper script | `./tc` |

**Key patterns**:
- All development is container-based
- Go services with CGO (for SQLite)
- Bind mounts for source code
- `./tc` script for all Docker operations

**Dependencies**: Docker, Docker Compose

---

## Data Flow

```
User/LLM Request
       │
       ▼
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│   CLI (Go)       │     │   MCP Server     │     │   Frontend       │
│   (cobra)        │     │   (Go tools)     │     │   (stores)       │
└────────┬─────────┘     └────────┬─────────┘     └────────┬─────────┘
         │                        │                        │
         │           HTTP/REST    │                        │
         └────────────────────────┼────────────────────────┘
                                  ▼
                       ┌──────────────────┐
                       │     Backend      │
                       │   (Go services)  │
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
| **Purpose** | Trip category: business, vacation, conference, family, other. | Type, kind, category |
| **Status** | Trip state: planning, confirmed, in_progress, completed, cancelled. | State, phase |
| **Store** | Svelte reactive store. Single source of truth for a resource type. | State, cache, model |
| **Tool** | MCP tool callable by LLM. Verb-oriented action. | Function, endpoint, command |
| **Resource** | MCP resource readable by LLM. Data endpoint. | Endpoint, API |
| **Component** | Svelte UI component OR project package (context-dependent). | Widget, module |
| **Service** | Go package containing business logic for an entity. | Controller, handler |
| **Entity** | Go struct with database tags. Internal representation. | Model, schema, table |
| **Handler** | Go HTTP handler implementing ServerInterface. | Controller, route |

---

## File Naming Conventions

| Pattern | Example | Used For |
|---------|---------|----------|
| `{resource}.go` | `trip.go` | Entity, service, or handler file |
| `{Resource}Card.svelte` | `TripCard.svelte` | List item view |
| `{Resource}Chip.svelte` | `TripChip.svelte` | Inline reference |
| `{Resource}Detail.svelte` | `TripDetail.svelte` | Full page view |
| `{Resource}Form.svelte` | `TripForm.svelte` | Create/edit form |
| `{resource}_test.go` | `trip_test.go` | Go unit tests |
| `uc-{number}-{description}.sh` | `uc-001-create-trip.sh` | E2E test script |

---

## API Endpoints

Defined in `packages/api/openapi.yaml`:

| Method | Endpoint | Returns |
|--------|----------|---------|
| `GET` | `/health` | `HealthResponse` |
| `GET` | `/api/trips` | `Trip[]` |
| `POST` | `/api/trips` | `Trip` |
| `GET` | `/api/trips/{tripId}` | `Trip` (with items) |
| `PATCH` | `/api/trips/{tripId}` | `Trip` |
| `DELETE` | `/api/trips/{tripId}` | `void` |
| `GET` | `/api/trips/search?q=` | `Trip[]` |
| `GET` | `/api/trips/{tripId}/items` | `Item[]` |
| `POST` | `/api/trips/{tripId}/items` | `Item` |
| `DELETE` | `/api/items/{itemId}` | `void` |
| `GET` | `/api/documents` | `Document[]` |

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
| `add_item` | Add item to trip |
| `delete_item` | Delete item |
| `get_documents` | List documents |

---

## CLI Commands

```
travel trips list [--upcoming] [--past] [--purpose X]
travel trips get <id>
travel trips create --name X --purpose X [--start X] [--end X]
travel trips update <id> [--name X] [--purpose X] [--status X]
travel trips delete <id>
travel trips search <query>
travel items list <trip-id>
travel items add <trip-id> <type> [--from X] [--to X] ...
travel items delete <id>
travel documents list [--trip X] [--unassociated]
travel completion [bash|zsh|fish|powershell]
```

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
