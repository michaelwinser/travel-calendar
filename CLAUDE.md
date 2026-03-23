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
├── tc                              # Project command script (build, test, serve, codegen, deploy)
├── blog/            # Development session logs
└── .claude/         # Agent configuration
```

### API-First Workflow

1. Define endpoints in `openapi.yaml`
2. Regenerate: `./tc codegen`
3. Implement new methods in `server.go` (compiler will tell you what's missing)
4. Add CLI commands in `main.go` using the generated client
5. Never hand-write route registrations or request/response types
6. Verify: `./tc lint-api`

### Key Dependency: appbase

This app is built on `github.com/michaelwinser/appbase`. Read `../appbase/CLAUDE.md` for module docs.

**The `appbase` CLI must be installed** for codegen, deploy, and secrets commands:
```bash
go install github.com/michaelwinser/appbase/cmd/appbase@latest
```
The binary is at `$(go env GOPATH)/bin/appbase`. If not on PATH, use the full path or add `$(go env GOPATH)/bin` to PATH. Do NOT use `go run ../appbase` — the module is a dependency, not a peer source checkout.

**Rules:**
- Do NOT modify appbase from this repo. If you need something new, log it in `docs/appbase-feedback.md`
- Domain entities and store are ours; appbase provides connections, auth, CLI base, server scaffolding
- Use `store.Collection[T]` for persistence — no raw SQL schema management
- Use `appbase.UserID(r)` for auth in HTTP handlers
- CLI commands MUST go through the API, never access the store directly

### appbase API Patterns (current)

- **Config:** `appbase.Config{LocalMode: appcli.IsLocalMode}` for CLI, `LocalMode: true` for desktop
- **CLI commands:** Use `appcli.ClientForCommand(cmd, "travel-calendar", app.Handler())`
- **DB path:** `cfg.DB.SQLitePath = appcli.LocalDataPath + "/app.db"` in setup()
- **Direct store access:** Use `appcli.LocalUserID()` not `"cli-user"`
- **SQLite driver:** `modernc.org/sqlite` (pure Go, driver name is `sqlite` not `sqlite3`)
- **Desktop:** Use `app.LocalHandler()` with Wails, not `app.Handler()`

See `../appbase/docs/migration-local-mode.md` for the full migration guide.

### Data Model

**Activity** is the primary entity: a span of time with a purpose and location.

- Types: `travel`, `stay`, `conference`, `vacation`, `commitment`
- Location: plain string for now (formalized entity deferred)
- Conflicts: computed at query time, not stored

### Running

```bash
# Project commands (use ./dev for everything)
./dev serve              # Start the web server (loads secrets)
./dev build              # Build the travel binary
./dev test               # Run Go tests
./dev e2e                # Run E2E smoke tests
./dev codegen            # Regenerate API code from openapi.yaml
./dev lint               # Run go vet
./dev lint-api           # Verify codegen is up to date
./dev ci                 # Full CI pipeline (lint + build + test + e2e)
./dev secret import ...  # Import Google OAuth credentials
./dev provision email    # Full GCP setup
./dev deploy             # Deploy to Cloud Run

# CLI — just works, no serve or login needed for local use
./travel add "Trip Name" --from 2026-04-01 --to 2026-04-05 --loc Paris --type travel
./travel list [--month 2026-04]
./travel check 2026-04-03
./travel quick "FOSDEM Jan 22 - Feb 3 in Brussels" --yes
./travel update <id-prefix> --title "New Title"
./travel delete <id-prefix>

# Remote server (requires login)
./travel login --server https://travel-calendar.run.app
./travel list --server https://travel-calendar.run.app
```

Three runtime modes:
- **Local CLI**: `./travel add ...` — in-process transport, single-user, no TCP
- **Web server**: `./dev serve` — full OAuth, persistent server
- **Remote CLI**: `./travel list --server https://...` — keychain auth

Config: `app.yaml` (loaded by appbase). Env var overrides still work (`STORE_TYPE`, `PORT`, etc).

## Git Workflow

**Commit message format**: `type: description`
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

Use the Commit Agent for all commits when available.

## Design Documents

- **PRD**: `docs/prd/kinetic-ledger.md` — vision, use cases, features
- **Technical Design**: `docs/kinetic-ledger-design.md` — data model, CLI spec, acceptance criteria
- **Mockups**: `docs/kinetic-ledger-*.png` — UI wireframes

## Development Tooling

**Do not install Node.js, npm, pnpm, or frontend build tools on the host.**
All frontend tooling runs inside the project's devcontainer.

### Frontend work

Never run `npm`, `npx`, `pnpm`, or `yarn` directly. Use the devcontainer:

```bash
# Start container, run command, stop container
docker compose -f .devcontainer/docker-compose.yml up -d frontend
docker compose -f .devcontainer/docker-compose.yml exec frontend sh -c "cd /app && <command>"
docker compose -f .devcontainer/docker-compose.yml down
```

If the project doesn't have `.devcontainer/Dockerfile.frontend` yet, create one following the pattern in `../appbase/.devcontainer/Dockerfile.frontend`.

To add a new frontend tool, add it to `.devcontainer/Dockerfile.frontend` — do not install on the host.

### Frontend types

TypeScript types are generated from `openapi.yaml`, not hand-written:

```bash
# Inside the devcontainer:
npx openapi-typescript openapi.yaml -o frontend/src/lib/api-types.ts
```

Import generated types in `api.ts`: `import type { components } from './api-types'`

Do not hand-write TypeScript interfaces for API request/response types.

## Principles

1. **API first** — OpenAPI spec is the contract. Generated code enforces it. CLI tests the API.
2. **Model first** — Get the data model right before building UI.
3. **appbase handles infrastructure** — Auth, sessions, database, deployment. We handle domain logic.
4. **Read before write** — Check adjacent test files and design docs before changes.
5. **Log appbase friction** — If appbase is missing something or has a rough edge, add it to `docs/appbase-feedback.md`.
