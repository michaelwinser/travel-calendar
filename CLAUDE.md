# Travel Calendar - Agent Instructions

This document is the constitution for AI agents working on this codebase. Read this file completely before making any changes.

## Quick Reference Files

| File | Purpose | When to Read |
|------|---------|--------------|
| `CLAUDE.md` | This file - constitution | Always (start here) |
| `PROJECT_MAP.md` | Component overview, lexicon | Understanding codebase |
| `.claude/agents.md` | Specialized agent definitions | Before component work |
| `.claude/reviewer.md` | Pushback/review criteria | Evaluating requests |
| `packages/*/ARCHITECTURE.md` | Component-specific rules | Before component changes |

## Project Philosophy

This codebase is designed for **AI-assisted development** with strong component boundaries. Every component has:
1. Clear architectural principles
2. Automated tests enforcing those principles
3. Local documentation the agent must read before changes

## Component Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           COMPONENTS                                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐              │
│  │   backend    │    │   frontend   │    │  mcp-server  │              │
│  │              │    │              │    │              │              │
│  │  REST API    │◄──►│   SvelteKit  │    │  LLM Tools   │              │
│  │  Entities    │    │   MVC        │    │              │              │
│  │  Services    │    │   Components │    │              │              │
│  └──────────────┘    └──────────────┘    └──────────────┘              │
│         │                   │                   │                       │
│         └───────────────────┴───────────────────┘                       │
│                             │                                           │
│                    ┌────────▼────────┐                                 │
│                    │     shared      │                                 │
│                    │                 │                                 │
│                    │  Types only     │                                 │
│                    │  No logic       │                                 │
│                    └─────────────────┘                                 │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Component Boundaries (STRICTLY ENFORCED)

| Component | Owns | Never Contains |
|-----------|------|----------------|
| `backend` | REST API, database, business logic | UI code, Svelte imports |
| `frontend` | UI components, reactive stores, routing | Direct DB access, business logic |
| `mcp-server` | MCP tools/resources, LLM-facing API | UI code, direct DB writes |
| `shared` | TypeScript types, constants | Logic, runtime code, dependencies |

## Task Decomposition Protocol

**BEFORE writing any code**, determine which components are affected:

1. **Single component** → Proceed directly, follow component's ARCHITECTURE.md
2. **Multiple components** → STOP. Create a plan document at `docs/plans/{issue-number}.md` that:
   - Lists each component affected
   - Describes changes to each component separately
   - Defines the integration points
   - Requires explicit user approval before proceeding

### Example Decomposition

```
Task: "Add expense tracking to trips"

Components affected:
- backend: New Expense entity, API endpoints
- frontend: ExpenseCard component, TripDetail updates
- mcp-server: New get_expenses tool
- shared: Expense types

→ This requires a plan document and component-by-component implementation
```

## Engineering Principles

### Git Workflow

1. **Every commit references an issue**: `feat(backend): add expense entity (#42)`
2. **Commit format**: `type(component): description (#issue)`
   - Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`
   - Component: `backend`, `frontend`, `mcp-server`, `shared`, `infra`
3. **No direct commits to main** - all changes via PR
4. **PR title matches commit format**

### Code Changes

1. **Read before write**: Always read the relevant ARCHITECTURE.md before changes
2. **Tests first for bugs**: Bug fixes require a failing test first
3. **No cross-component imports** except from `shared`
4. **Run component tests** after changes: `pnpm test:backend` or `pnpm test:frontend`

### PRD → Tests Pipeline

PRDs in `docs/prd/` contain use cases that become tests:
- Use cases have `[UC-XXX]` identifiers
- Each use case maps to a test in `tests/e2e/`
- Tests are shell scripts using the CLI client

## Component-Specific Instructions

Each component has an `ARCHITECTURE.md` file. **You must read it before making changes.**

- `packages/backend/ARCHITECTURE.md` - REST API patterns, entity design
- `packages/frontend/ARCHITECTURE.md` - MVC patterns, component rules
- `packages/mcp-server/ARCHITECTURE.md` - Tool design, resource patterns
- `packages/shared/ARCHITECTURE.md` - Type conventions

## Locality of Knowledge

When working on a file, check for:
1. **Component ARCHITECTURE.md** - overall patterns
2. **Directory README.md** - specific conventions for that directory
3. **Adjacent test files** - `*.test.ts` shows expected behavior
4. **Type definitions** - in `shared/` for the entity you're touching

## Pushback Protocol

Before executing requests, evaluate them against project principles. See `.claude/reviewer.md` for full criteria.

**Always pushback when**:
- Request would violate component boundaries
- Request bundles unrelated changes
- Request uses incorrect terminology (check PROJECT_MAP.md lexicon)
- Request adds complexity without clear benefit
- Request introduces new patterns without justification

**Pushback phrases**:
- "Before we proceed, let me check this against our principles..."
- "I notice this would touch multiple components. Should we create a plan first?"
- "This is doable, but I want to flag a potential concern..."
- "Our lexicon uses '{X}' instead of '{Y}'. Should I use the standard term?"

**Don't just execute**. Think critically. The user benefits from honest evaluation.

## Automated Enforcement

The following are automatically checked:

| Check | Enforces | Runs |
|-------|----------|------|
| `lint:boundaries` | No cross-component imports | Pre-commit |
| `lint:types` | Shared types have no logic | Pre-commit |
| `test:unit` | Component contracts | Pre-commit |
| `test:e2e` | Use case journeys | CI |
| `lint:commits` | Commit message format | Pre-commit |

## Quick Reference

**All development is container-based.** See `CONTRIBUTING.md` for full Docker workflow.

```bash
# Start development environment (Docker)
docker compose up               # Start all services
docker compose up -d            # Start in background
docker compose logs -f backend  # View logs

# Read component docs before changes
cat packages/{component}/ARCHITECTURE.md

# Run tests
docker compose exec backend pnpm test  # In container
docker compose --profile test up       # Full test suite

# Linting
docker compose exec backend pnpm lint

# CLI (against containerized backend)
export TRAVEL_API_URL=http://localhost:3000
./cli/travel trips list
./cli/travel trips get <id>
./cli/travel items add <trip> flight

# Stop
docker compose down
```

## Command Execution Rules

**All development commands MUST run through the sandbox container.** This ensures operations are scoped to the project directory.

### Run in Container (via `./tc exec`)

```bash
# Testing
./tc exec pnpm test
./tc exec pnpm test:backend
./tc exec pnpm test:frontend

# Linting
./tc exec pnpm lint
./tc exec node scripts/check-boundaries.js

# Node/pnpm operations
./tc exec pnpm install
./tc exec node scripts/validate-map.js
```

### Run on Host (directly)

These commands require host access:
```bash
# Git (needs SSH keys, .git directory)
git status
git add .
git commit -m "..."
git push

# Docker (needs socket)
docker compose up
./tc start
./tc build

# GitHub CLI (needs auth)
gh issue create
gh pr create
```

### Rule of Thumb

- **If it runs Node.js or accesses node_modules** → use `./tc exec`
- **If it accesses .git, Docker, or external services** → run on host

## Sandbox Configuration

This project is configured to minimize permission prompts for Claude Code.

### Sandbox Container (Primary)

```bash
# Start sandbox container
./tc sandbox start

# Run commands inside the sandbox
./tc exec pnpm test
./tc exec node scripts/check-boundaries.js
./tc exec ls -la

# Interactive shell
./tc sandbox shell

# Stop when done
./tc sandbox stop
```

The sandbox container:
- Mounts the project directory at `/app`
- Has all development tools (node, pnpm, git, etc.)
- Cannot access anything outside the project
- Auto-starts when you use `./tc exec`

### Permission Settings

Permissions are configured in `.claude/settings.json`:
- `./tc` commands are pre-allowed (including `./tc exec`)
- Git and Docker commands are pre-allowed
- Dangerous commands (`rm -rf /`, etc.) are blocked

**Restart Claude Code** after modifying settings for changes to take effect.
