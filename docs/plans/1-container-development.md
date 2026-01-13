# Plan: Container-Based Development Environment

**Issue**: #1
**Components**: infra, backend, frontend, mcp-server, docs
**Status**: Implemented

---

## Summary

Add Docker-based development environment so all services run in containers. Developers only need Docker installed - no Node.js, pnpm, or other tools on their host machine.

## Goals

1. **Portability**: `docker compose up` gets a working dev environment
2. **Isolation**: No host pollution with tool versions
3. **Consistency**: Same environment locally and in CI
4. **Developer Experience**: Hot reload, fast rebuilds, easy debugging

## Non-Goals

- Production deployment configuration (separate concern)
- Kubernetes/orchestration (overkill for this project)
- Multi-architecture builds (can add later)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    docker-compose.yml                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   backend   │  │  frontend   │  │ mcp-server  │             │
│  │   :3000     │  │   :5173     │  │  (stdio)    │             │
│  │             │  │             │  │             │             │
│  │  Node 20    │  │  Node 20    │  │  Node 20    │             │
│  │  Hono       │  │  SvelteKit  │  │  MCP SDK    │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│         │                │                                      │
│         └────────────────┴──────────────────────┐               │
│                                                 │               │
│  ┌──────────────────────────────────────────────┴────────────┐ │
│  │                     volumes                                │ │
│  │  ./packages → /app/packages (bind mount, hot reload)      │ │
│  │  ./data → /app/data (SQLite persistence)                  │ │
│  │  node_modules → named volume (faster, isolated)           │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Infra Changes (NEW)

### Files to Create

| File | Purpose |
|------|---------|
| `Dockerfile` | Multi-stage build for all packages |
| `Dockerfile.dev` | Development image with hot reload |
| `docker-compose.yml` | Orchestrates all services |
| `docker-compose.override.yml` | Local overrides (gitignored) |
| `.dockerignore` | Exclude unnecessary files |

### Dockerfile.dev

```dockerfile
FROM node:20-alpine

# Install pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /app

# Copy package files for dependency installation
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml* ./
COPY packages/backend/package.json ./packages/backend/
COPY packages/frontend/package.json ./packages/frontend/
COPY packages/mcp-server/package.json ./packages/mcp-server/
COPY packages/shared/package.json ./packages/shared/

# Install dependencies
RUN pnpm install --frozen-lockfile

# Source code mounted as volume for hot reload
# CMD specified in docker-compose.yml per service
```

### docker-compose.yml

```yaml
services:
  backend:
    build:
      context: .
      dockerfile: Dockerfile.dev
    command: pnpm --filter @travel-calendar/backend dev
    ports:
      - "3000:3000"
    volumes:
      - ./packages:/app/packages
      - ./data:/app/data
      - backend_modules:/app/packages/backend/node_modules
    environment:
      - DATABASE_PATH=/app/data/travel.db

  frontend:
    build:
      context: .
      dockerfile: Dockerfile.dev
    command: pnpm --filter @travel-calendar/frontend dev --host
    ports:
      - "5173:5173"
    volumes:
      - ./packages:/app/packages
      - frontend_modules:/app/packages/frontend/node_modules
    environment:
      - API_URL=http://backend:3000
    depends_on:
      - backend

  mcp-server:
    build:
      context: .
      dockerfile: Dockerfile.dev
    command: pnpm --filter @travel-calendar/mcp-server dev
    volumes:
      - ./packages:/app/packages
      - mcp_modules:/app/packages/mcp-server/node_modules
    environment:
      - API_URL=http://backend:3000
    # No ports - MCP uses stdio
    profiles:
      - mcp  # Only start with --profile mcp

volumes:
  backend_modules:
  frontend_modules:
  mcp_modules:
```

---

## Backend Changes

- [ ] Add `dev` script that works in container (host binding)
- [ ] Ensure DATABASE_PATH is configurable via environment
- [ ] No code changes expected - just configuration

---

## Frontend Changes

- [ ] Add `--host` flag to dev server for container access
- [ ] Ensure API_URL is configurable via environment
- [ ] Update vite.config.ts for container networking

---

## MCP Server Changes

- [ ] Ensure API_URL is configurable via environment
- [ ] Document how to connect Claude Desktop to containerized server

---

## Shared Changes

- [ ] No changes needed (types only)

---

## Documentation Changes

Update `CONTRIBUTING.md` and `README.md`:

### New Development Workflow

```bash
# Start development environment
docker compose up

# Run in background
docker compose up -d

# View logs
docker compose logs -f backend

# Run tests
docker compose exec backend pnpm test

# Run CLI against containers
TRAVEL_API_URL=http://localhost:3000 ./cli/travel trips list

# Stop
docker compose down

# Rebuild after dependency changes
docker compose build --no-cache
```

### File Permissions

Add note about file permissions on Linux (container runs as root by default).

---

## Testing Strategy

1. **Manual verification**: `docker compose up` starts all services
2. **Hot reload test**: Edit a file, verify change appears
3. **E2E tests**: Run `tests/e2e/run-all.sh` against containerized backend
4. **CLI test**: Verify CLI works with `TRAVEL_API_URL=http://localhost:3000`

---

## Implementation Order

1. Create `.dockerignore`
2. Create `Dockerfile.dev`
3. Create `docker-compose.yml`
4. Update package.json scripts (if needed)
5. Update backend for environment configuration
6. Update frontend for environment configuration
7. Update MCP server for environment configuration
8. Update documentation
9. Test full workflow

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Slow rebuilds | Use volume mounts, named volumes for node_modules |
| Hot reload not working | Ensure proper volume mounts, polling if needed |
| SQLite file permissions | Mount data directory, set permissions |
| MCP stdio with containers | Document alternative: run MCP server on host |

---

## Approval

- [ ] Plan reviewed and approved
- [ ] User confirms approach aligns with expectations
