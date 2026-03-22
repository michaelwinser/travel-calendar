# Travel Calendar v2

A high-velocity planning tool for frequent travelers, built on [appbase](https://github.com/michaelwinser/appbase).

## Architecture

API-first, single Go module. OpenAPI spec is the source of truth. CLI talks to the server via HTTP.

```
travel-calendar/
├── openapi.yaml                # API contract (source of truth)
├── oapi-codegen.yaml           # Server codegen config
├── oapi-codegen-client.yaml    # Client codegen config
├── api/
│   ├── server.gen.go           # Generated: ServerInterface + types (DO NOT EDIT)
│   └── client.gen.go           # Generated: HTTP client (DO NOT EDIT)
├── main.go                     # CLI commands (uses generated client) + server wiring
├── server.go                   # Implements api.ServerInterface
├── store.go                    # Activity entity + ActivityStore (uses appbase/store)
├── app.yaml                    # App config (appbase convention)
├── app.json                    # Deploy config (read by shell scripts)
├── go.mod
├── docs/
│   ├── prd/kinetic-ledger.md        # Product requirements
│   ├── kinetic-ledger-design.md     # Technical design & CLI spec
│   ├── kinetic-ledger-*.png         # UI mockups
│   └── appbase-feedback.md          # Issues/feedback for appbase module
├── blog/            # Development session logs
└── .claude/         # Agent configuration
```

### API-First Workflow

1. Define endpoints in `openapi.yaml`
2. Regenerate: `/Users/michaelw/go/bin/oapi-codegen -config oapi-codegen.yaml openapi.yaml && /Users/michaelw/go/bin/oapi-codegen -config oapi-codegen-client.yaml openapi.yaml`
3. Implement new methods in `server.go` (compiler will tell you what's missing)
4. Add CLI commands in `main.go` using the generated client
5. Never hand-write route registrations or request/response types

### Key Dependency: appbase

This app is built on `github.com/michaelwinser/appbase`. Read `../appbase/CLAUDE.md` for module docs.

**Rules:**
- Do NOT modify appbase from this repo. If you need something new, log it in `docs/appbase-feedback.md`
- Domain entities and store are ours; appbase provides connections, auth, CLI base, server scaffolding
- Use `store.Collection[T]` for persistence — no raw SQL schema management
- Use `appbase.UserID(r)` for auth in HTTP handlers
- CLI commands MUST go through the API, never access the store directly

### Data Model

**Activity** is the primary entity: a span of time with a purpose and location.

- Types: `travel`, `stay`, `conference`, `vacation`, `commitment`
- Location: plain string for now (formalized entity deferred)
- Conflicts: computed at query time, not stored

### Running

```bash
# Server
go run . serve

# CLI (requires a running server + login)
go run . login --server http://localhost:3000
go run . add "Trip Name" --from 2026-04-01 --to 2026-04-05 --loc Paris --type travel
go run . list [--month 2026-04]
go run . check 2026-04-03
go run . delete <id-prefix>

# Build
go build -o travel-calendar .

# Test
go test ./...
```

Config: `app.yaml` (loaded by appbase). Env var overrides still work (`STORE_TYPE`, `PORT`, etc).

## Git Workflow

**Commit message format**: `type: description`
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

Use the Commit Agent for all commits when available.

## Design Documents

- **PRD**: `docs/prd/kinetic-ledger.md` — vision, use cases, features
- **Technical Design**: `docs/kinetic-ledger-design.md` — data model, CLI spec, acceptance criteria
- **Mockups**: `docs/kinetic-ledger-*.png` — UI wireframes

## Principles

1. **API first** — OpenAPI spec is the contract. Generated code enforces it. CLI tests the API.
2. **Model first** — Get the data model right before building UI.
3. **appbase handles infrastructure** — Auth, sessions, database, deployment. We handle domain logic.
4. **Read before write** — Check adjacent test files and design docs before changes.
5. **Log appbase friction** — If appbase is missing something or has a rough edge, add it to `docs/appbase-feedback.md`.
