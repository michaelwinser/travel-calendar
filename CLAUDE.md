# Travel Calendar - Agent Instructions

This document is the constitution for AI agents working on this codebase. Read this file completely before making any changes.

---

## Part 1: Universal Principles

These principles apply to ALL agents. Every agent inherits and must follow these rules.

### Project Philosophy

This codebase is designed for **AI-assisted development** with strong component boundaries. Every component has:
1. Clear architectural principles
2. Automated tests enforcing those principles
3. Local documentation the agent must read before changes

### Component Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           COMPONENTS                                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐              │
│  │   backend    │    │   frontend   │    │  mcp-server  │              │
│  │   (Go)       │    │              │    │   (Go)       │              │
│  │  REST API    │◄──►│   SvelteKit  │    │  LLM Tools   │              │
│  │  Entities    │    │   MVC        │    │  JSON-RPC    │              │
│  │  Services    │    │   Components │    │              │              │
│  └──────────────┘    └──────────────┘    └──────────────┘              │
│         ▲                   │                   │                       │
│         │                   │                   │                       │
│  ┌──────┴───────┐          │                   │                       │
│  │    cli       │          │                   │                       │
│  │   (Go)       │          │                   │                       │
│  │   Cobra      │          │                   │                       │
│  └──────────────┘          │                   │                       │
│         │                   │                   │                       │
│         └───────────────────┴───────────────────┘                       │
│                             │                                           │
│                    ┌────────▼────────┐                                 │
│                    │     shared      │                                 │
│                    │                 │                                 │
│                    │  Types only     │                                 │
│                    │  (generated)    │                                 │
│                    └─────────────────┘                                 │
│                             ▲                                           │
│                    ┌────────┴────────┐                                 │
│                    │      api        │                                 │
│                    │  OpenAPI Spec   │                                 │
│                    │  (source of     │                                 │
│                    │   truth)        │                                 │
│                    └─────────────────┘                                 │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Component Boundaries (STRICTLY ENFORCED)

| Component | Owns | Never Contains |
|-----------|------|----------------|
| `api` | OpenAPI specification | Implementation code |
| `backend` | REST API, database, business logic (Go) | UI code, frontend imports |
| `frontend` | UI components, reactive stores, routing | Direct DB access, business logic |
| `mcp-server` | MCP tools, LLM-facing API (Go) | UI code, direct DB access |
| `cli` | Command-line interface (Go) | UI code, business logic |
| `shared` | TypeScript types (generated from OpenAPI) | Logic, runtime code, manual type definitions |

### Git Workflow

**CRITICAL: Only the Commit Agent may execute `git commit` or `git push` commands.**

All other agents must delegate commit/push operations to the Commit Agent. This ensures:
- Code Review Agent is always invoked before commits
- Pre-commit checks (`./tc test-precommit`) always run
- Consistent commit message formatting

1. **Every commit references an issue**: `feat(backend): add expense entity (#42)`
2. **Commit format**: `type(component): description (#issue)`
   - Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`
   - Component: `api`, `backend`, `frontend`, `mcp-server`, `cli`, `shared`, `infra`
3. **No direct commits to main** - all changes via PR
4. **PR title matches commit format**

### Code Changes

1. **Read before write**: Always read the relevant ARCHITECTURE.md before changes
2. **Tests first for bugs**: Bug fixes require a failing test first
3. **No cross-component imports** except from `shared`
4. **Run component tests** after changes

### Locality of Knowledge

When working on a file, check for:
1. **Component ARCHITECTURE.md** - overall patterns
2. **Directory README.md** - specific conventions for that directory
3. **Adjacent test files** - `*_test.go` or `*.test.ts` shows expected behavior
4. **Type definitions** - in `shared/` (generated from OpenAPI) for the entity you're touching
5. **OpenAPI spec** - `packages/api/openapi.yaml` for API contracts

### Pushback Protocol

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

### Automated Enforcement

The following are automatically checked:

| Check | Enforces | Runs |
|-------|----------|------|
| `lint:boundaries` | No cross-component imports | Pre-commit |
| `lint:types` | Shared types have no logic | Pre-commit |
| `test:unit` | Component contracts | Pre-commit |
| `test:e2e` | Use case journeys | CI |
| `lint:commits` | Commit message format | Pre-commit |

### Docker Operations

**All Docker operations MUST go through the `./tc` script.**

Raw `docker` and `docker-compose` commands are denied in `.claude/settings.json`. This ensures:
- All Docker operations are auditable through `tc`
- Consistent interface across all agents
- No sandbox/permission issues with Docker socket

```bash
# Use these:
./tc build                      # Build containers
./tc start                      # Start services
./tc stop                       # Stop services
./tc health                     # Check service health
./tc curl <service:port/path>   # HTTP requests inside container
./tc exec <command>             # Run command in backend container
./tc logs [service]             # View service logs

# Backend testing (Go):
./tc exec sh -c "cd packages/backend && go test ./..."

# MCP testing (Go):
./tc exec sh -c "cd packages/mcp-server && go test ./..."

# NOT these (will be denied):
docker compose up       # DENIED
docker build .          # DENIED
curl localhost:3000     # DENIED (use ./tc curl backend:3000)
```

### Protected Files

The following files have special ownership rules:

| File | Owner | Rule |
|------|-------|------|
| `tc` | Infra Agent ONLY | Must get user approval before ANY modifications |
| `Dockerfile.*` | Infra Agent | Standard infra review process |
| `docker-compose.yml` | Infra Agent | Standard infra review process |
| `.claude/settings.json` | Infra Agent | Standard infra review process |

**The `tc` script is critical infrastructure.** Only the Infra Agent may modify it, and ONLY after presenting proposed changes to the user and receiving explicit approval.

---

## Part 2: Agent Selection & Routing

### Task Decomposition Protocol

**BEFORE writing any code**, determine which components are affected:

1. **Single component** → Read the agent file, then proceed
2. **Multiple components** → Read `.claude/agents/cross-component.md`, create a plan document at `docs/plans/{issue-number}.md`, get approval before proceeding

### Agent Selection Matrix

| Task Type | Agent File | Scope |
|-----------|------------|-------|
| OpenAPI specification | `.claude/agents/cross-component.md` | `packages/api/` |
| Backend API, entities, services (Go) | `.claude/agents/backend.md` | `packages/backend/` |
| Frontend UI, stores, routes | `.claude/agents/frontend.md` | `packages/frontend/` |
| MCP tools, LLM resources (Go) | `.claude/agents/mcp-server.md` | `packages/mcp-server/` |
| CLI commands (Go) | `.claude/agents/cross-component.md` | `packages/cli/` |
| TypeScript types (generated) | `.claude/agents/shared.md` | `packages/shared/` |
| Multi-component changes | `.claude/agents/cross-component.md` | Multiple packages |
| Docker, containers, CI/CD | `.claude/agents/infra.md` | Infrastructure |
| End-to-end tests | `.claude/agents/e2e-test.md` | `tests/e2e/` |
| **Pre-commit review** | `.claude/agents/code-review.md` | All changes |
| **Git commits and pushes** | `.claude/agents/commit.md` | All git operations |
| **Tool usage & permissions** | `.claude/agents/tools.md` | Session analysis, tc/settings optimization |
| **Session summaries** | `.claude/agents/session-summary.md` | Progress tracking, blog entries |
| **PRDs & product roadmap** | `.claude/agents/product-management.md` | `docs/prd/*.md`, `docs/PRD.md`, feature definitions |

### Agent Quick Reference

Each agent has detailed checklists. Here are the key rules:

#### Backend Agent (Go)
- **Read first**: `packages/backend/ARCHITECTURE.md`
- **Test**: `./tc exec sh -c "cd packages/backend && go test ./..."`
- **Forbidden**: UI code, frontend imports, direct DB queries in handlers

#### Frontend Agent
- **Read first**: `packages/frontend/ARCHITECTURE.md`
- **Test**: `./tc exec pnpm test:frontend`
- **Forbidden**: Direct API calls in components, ID-based lookups, business logic

#### MCP Server Agent (Go)
- **Read first**: `packages/mcp-server/ARCHITECTURE.md`
- **Test**: `./tc exec sh -c "cd packages/mcp-server && go test ./..."`
- **Forbidden**: Direct database access, UI code, raw JSON responses

#### CLI Agent (Go)
- **Read first**: `packages/cli/ARCHITECTURE.md`
- **Build**: `cd packages/cli && go build -o travel ./cmd/travel`
- **Forbidden**: Direct API calls (use generated client), business logic

#### Shared Agent
- **Read first**: `packages/shared/ARCHITECTURE.md`
- **Regenerate**: `./tc exec pnpm --filter @travel-calendar/shared generate`
- **Forbidden**: Editing api.ts directly, runtime code, manual type definitions

#### Cross-Component Agent
- **Action**: Create plan at `docs/plans/{issue-number}.md`
- **Order**: api → shared → backend → cli → mcp-server → frontend → e2e tests
- **Requirement**: Get explicit user approval before implementing

#### Infra Agent
- **Owns**: `tc` script, Dockerfiles, docker-compose.yml, `.claude/settings.json`
- **Rule**: ONLY agent that can modify `tc`; must get user approval first
- **Test**: `./tc build && ./tc start && ./tc health`
- **Verify**: All health checks pass before committing

#### E2E Test Agent
- **Source**: PRDs in `docs/prd/` with `[UC-XXX]` identifiers
- **Output**: Shell scripts in `tests/e2e/uc-{number}-{description}.sh`
- **Test**: Run the script directly

#### Code Review Agent
- **When**: Before committing, after completing work, or on request
- **Does**: Identifies changed components, runs tests, checks CLAUDE.md compliance
- **Spawns**: Component agents to review their specific areas
- **Output**: Review report with APPROVED, NEEDS FIXES, or NEEDS DISCUSSION
- **On failure**: Prompts user for guidance on how to proceed

#### Commit Agent
- **Exclusive authority**: ONLY agent allowed to run `git commit` or `git push`
- **Mandatory steps**: Run `./tc test-precommit`, invoke Code Review Agent
- **Workflow**: Pre-commit checks → Code Review → Stage → Commit → Tools Review → (Push if requested)
- **Forbidden**: Committing without Code Review approval, force-push to main/master

#### Tools Agent
- **When**: After commits, on request, or when permission patterns emerge
- **Does**: Analyzes tool usage, identifies permission prompt patterns, proposes optimizations
- **Proposes**: Additions to `tc` script, changes to `.claude/settings.json` permissions
- **Goals**: User confidence, minimize prompts, identify workflow gaps
- **Works with**: Infra Agent (implements approved changes)

#### Session Summary Agent
- **When**: After commits (via Commit Agent), on request, or at session end
- **Does**: Maintains running summary in `blog/.current.md`, detects session boundaries
- **Output**: Dated blog entries in `blog/YYYY-MM-DD-slug.md` when sessions conclude
- **Invoked by**: Commit Agent (after Tools Agent step)
- **Files**: `blog/.current.md` (working file, gitignored), `blog/*.md` (finalized entries)

#### Product Management Agent
- **When**: New feature requests, scope clarification, feature completion review, roadmap planning
- **Does**: Creates/maintains PRDs, defines use cases with acceptance criteria, identifies MVP scope
- **Output**: PRDs in `docs/prd/*.md`, roadmap updates, UX/UI design guidance
- **Collaborates with**: Stakeholder (Claude end-user) for requirements, implementation agents for handoff
- **Evaluates**: Feature completion against acceptance criteria

---

## Reference Files

| File | Purpose | When to Read |
|------|---------|--------------|
| `CLAUDE.md` | This file - universal principles | Always (start here) |
| `PROJECT_MAP.md` | Component overview, lexicon | Understanding codebase |
| `.claude/agents/*.md` | Detailed agent checklists | Before component work |
| `.claude/agents/commit.md` | Commit workflow (exclusive authority) | Before any commit |
| `.claude/agents/code-review.md` | Pre-commit review process | Before committing |
| `.claude/agents/tools.md` | Tool usage analysis and optimization | After commits, on request |
| `.claude/agents/session-summary.md` | Progress tracking and blog entries | After commits, session end |
| `.claude/agents/product-management.md` | PRD ownership, MVP scoping, UX guidance | Feature planning, completion review |
| `.claude/reviewer.md` | Pushback/review criteria | Evaluating requests |
| `packages/*/ARCHITECTURE.md` | Component-specific patterns | Before component changes |
