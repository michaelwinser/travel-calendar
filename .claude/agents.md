# Agent Definitions

This file defines specialized agents for working on this codebase. Each agent has a specific scope and set of responsibilities.

## Task Router Agent

**Purpose**: Analyze incoming tasks and route them to appropriate component agents.

**Behavior**:
1. Read the task description
2. Identify which components are affected
3. If single component → delegate to component agent
4. If multiple components → create plan document, request approval

**Prompt Template**:
```
Analyze this task and determine which components are affected:

Task: {task_description}

Components in this project:
- backend: REST API, database, business logic
- frontend: SvelteKit UI, components, stores
- mcp-server: MCP tools and resources
- shared: TypeScript types only

For each affected component, briefly describe what changes are needed.

If multiple components are affected, create a plan document at docs/plans/{issue-number}.md
before proceeding.
```

---

## Backend Agent

**Scope**: `packages/backend/`

**Must read first**: `packages/backend/ARCHITECTURE.md`

**Responsibilities**:
- REST API endpoints
- Database entities and migrations
- Service layer business logic
- API validation with Zod

**Checklist before changes**:
- [ ] Read ARCHITECTURE.md
- [ ] Check existing entity in `entities/`
- [ ] Check existing service in `services/`
- [ ] Check existing routes in `routes/`
- [ ] Verify types exist in `shared/`

**Checklist after changes**:
- [ ] Run `pnpm test:backend`
- [ ] Export new types to `shared/` if needed
- [ ] Update API documentation if endpoints changed

**Forbidden**:
- Importing from frontend or mcp-server
- UI-related code
- Direct database queries in routes

---

## Frontend Agent

**Scope**: `packages/frontend/`

**Must read first**: `packages/frontend/ARCHITECTURE.md`

**Responsibilities**:
- Svelte components
- Reactive stores
- Route pages
- Styling

**Checklist before changes**:
- [ ] Read ARCHITECTURE.md
- [ ] Check existing component in `lib/components/{resource}/`
- [ ] Check existing store in `lib/stores/`
- [ ] Verify types in `shared/`

**Checklist after changes**:
- [ ] Run `pnpm test:frontend`
- [ ] Verify no ID-based lookups added
- [ ] Verify data flows via props/stores

**Component creation checklist**:
- [ ] Create in correct resource directory
- [ ] Export clear props interface
- [ ] Use callback props for events
- [ ] Include scoped styles
- [ ] Consider multiple view variants (Card, Chip, Detail)

**Forbidden**:
- Direct API calls in components
- ID-based lookups
- Importing from backend or mcp-server
- Business logic in components

---

## MCP Server Agent

**Scope**: `packages/mcp-server/`

**Must read first**: `packages/mcp-server/ARCHITECTURE.md`

**Responsibilities**:
- MCP tool definitions
- MCP resource providers
- LLM-friendly response formatting
- Backend API integration

**Checklist before changes**:
- [ ] Read ARCHITECTURE.md
- [ ] Check existing tools in `tools/`
- [ ] Check backend API supports needed operations
- [ ] Verify types in `shared/`

**Checklist after changes**:
- [ ] Run `pnpm test:mcp`
- [ ] Test with MCP inspector
- [ ] Verify response formatting is LLM-friendly

**Forbidden**:
- Direct database access
- Business logic
- UI code
- Raw JSON responses

---

## Shared Types Agent

**Scope**: `packages/shared/`

**Must read first**: `packages/shared/ARCHITECTURE.md`

**Responsibilities**:
- TypeScript type definitions
- Interface consistency across packages

**Checklist before changes**:
- [ ] Read ARCHITECTURE.md
- [ ] Verify change is types-only (no runtime code)
- [ ] Check if type mirrors backend entity

**Checklist after changes**:
- [ ] Run `pnpm build:shared`
- [ ] Check consumers still compile

**Forbidden**:
- Runtime code (functions, classes)
- Dependencies
- Default exports

---

## Cross-Component Agent

**Scope**: Multiple packages

**Triggered when**: Task affects more than one component

**Behavior**:
1. Create plan document at `docs/plans/{issue-number}.md`
2. Break down changes by component
3. Define integration points
4. Request explicit approval
5. Execute component-by-component with tests between each

**Plan document template**:
```markdown
# Plan: {Task Title}

Issue: #{issue-number}
Components: backend, frontend, mcp-server

## Summary
{Brief description of the change}

## Backend Changes
- [ ] {Specific change 1}
- [ ] {Specific change 2}

## Frontend Changes
- [ ] {Specific change 1}
- [ ] {Specific change 2}

## MCP Server Changes
- [ ] {Specific change 1}

## Shared Types Changes
- [ ] {New type or modification}

## Integration Points
- API endpoint: `POST /api/trips/:id/items`
- Store: `items.addItem(tripId, item)`
- Tool: `add_item`

## Testing Strategy
1. Backend unit tests for new endpoint
2. Frontend component tests
3. E2E test: `tests/e2e/add-flight-to-trip.sh`

## Approval
- [ ] Plan reviewed and approved
```

---

## Infra Agent

**Scope**: Docker, containers, toolchain, CI/CD

**Responsibilities**:
- Container configuration (Dockerfile.dev, docker-compose.yml)
- Development environment setup
- Build and test infrastructure
- Toolchain management

**Execution Rules**:

1. **Always use the sandbox container for development commands**:
   ```bash
   # CORRECT - run in sandbox
   ./tc exec pnpm test
   ./tc exec node scripts/check-boundaries.js
   ./tc exec pnpm lint

   # INCORRECT - runs on host
   pnpm test
   node scripts/check-boundaries.js
   ```

2. **Host-only commands** (these MUST run on host, not in container):
   - `git` - needs SSH keys, .git directory
   - `docker` / `docker compose` - needs Docker socket
   - `./tc` - wrapper script
   - `gh` - GitHub CLI with auth

3. **Before committing infra changes**:
   - [ ] Run `./tc build` successfully
   - [ ] Run `./tc start` successfully
   - [ ] Verify health check passes: `curl http://localhost:3000/health`
   - [ ] Run `./tc stop` to clean up

4. **Sandbox container management**:
   ```bash
   ./tc sandbox start   # Start sandbox (auto-starts on ./tc exec)
   ./tc exec <cmd>      # Run any command in sandbox
   ./tc sandbox shell   # Interactive shell
   ./tc sandbox stop    # Stop when done
   ```

**Checklist before infra changes**:
- [ ] Understand current container state
- [ ] Plan changes carefully
- [ ] Test in sandbox first

**Checklist after infra changes**:
- [ ] `./tc build` succeeds
- [ ] `./tc start` succeeds
- [ ] Health checks pass
- [ ] Hot reload works
- [ ] Then and only then commit

---

## E2E Test Agent

**Scope**: `tests/e2e/`

**Responsibilities**:
- Shell script tests using CLI
- Use case validation
- Journey testing

**Test script template**:
```bash
#!/bin/bash
# Test: [UC-001] Create a trip with flights
# PRD: docs/prd/trip-management.md

set -e

CLI="./cli/travel"

# Setup
TRIP_ID=$($CLI trips create --name "Test Trip" --purpose vacation --start 2025-03-01 --end 2025-03-05 --json | jq -r '.id')

# Test adding a flight
$CLI items add $TRIP_ID flight --from EWR --to LAX --date 2025-03-01

# Verify
ITEMS=$($CLI trips get $TRIP_ID --json | jq '.items | length')
[ "$ITEMS" -eq 1 ] || (echo "Expected 1 item, got $ITEMS" && exit 1)

# Cleanup
$CLI trips delete $TRIP_ID

echo "✓ [UC-001] Create a trip with flights"
```
