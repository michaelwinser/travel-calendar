# Travel Calendar v2

A high-velocity planning tool for frequent travelers, built on [appbase](https://github.com/michaelwinser/appbase).

## Architecture

Single Go module. CLI and web server in one binary.

```
travel-calendar/
├── main.go          # CLI commands and server setup (uses appbase)
├── store.go         # Activity entity and store (uses appbase/store)
├── app.json         # Project identity (appbase convention)
├── go.mod
├── docs/
│   ├── prd/kinetic-ledger.md        # Product requirements
│   ├── kinetic-ledger-design.md     # Technical design & CLI spec
│   ├── kinetic-ledger-*.png         # UI mockups
│   └── appbase-feedback.md          # Issues/feedback for appbase module
├── blog/            # Development session logs
└── .claude/         # Agent configuration
```

### Key Dependency: appbase

This app is built on `github.com/michaelwinser/appbase`. Read `../appbase/CLAUDE.md` for module docs.

**Rules:**
- Do NOT modify appbase from this repo. If you need something new, log it in `docs/appbase-feedback.md`
- Domain entities and store are ours; appbase provides connections, auth, CLI base, server scaffolding
- Use `store.Collection[T]` for persistence — no raw SQL schema management
- Use `appbase.UserID(r)` for auth in HTTP handlers

### Data Model

**Activity** is the primary entity: a span of time with a purpose and location.

- Types: `travel`, `stay`, `conference`, `vacation`, `commitment`
- Location: plain string for now (formalized entity deferred)
- Conflicts: computed at query time, not stored

### Running

```bash
# Server
go run . serve

# CLI
go run . add "Trip Name" --from 2026-04-01 --to 2026-04-05 --loc Paris --type travel
go run . list [--month 2026-04]
go run . check 2026-04-03
go run . delete <id-prefix>

# Build
go build -o travel-calendar .

# Test
go test ./...
```

Environment: `STORE_TYPE=sqlite` (default), `SQLITE_DB_PATH=data/app.db` (default).

## Git Workflow

**Commit message format**: `type: description`
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

Use the Commit Agent for all commits when available.

## Design Documents

- **PRD**: `docs/prd/kinetic-ledger.md` — vision, use cases, features
- **Technical Design**: `docs/kinetic-ledger-design.md` — data model, CLI spec, acceptance criteria
- **Mockups**: `docs/kinetic-ledger-*.png` — UI wireframes

## Principles

1. **Model first** — Get the data model right before building UI. The CLI validates the model.
2. **appbase handles infrastructure** — Auth, sessions, database, deployment. We handle domain logic.
3. **Read before write** — Check adjacent test files and design docs before changes.
4. **Log appbase friction** — If appbase is missing something or has a rough edge, add it to `docs/appbase-feedback.md`.
