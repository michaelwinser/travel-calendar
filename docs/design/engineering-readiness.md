# Engineering Readiness — Design Document

**Date**: 2026-03-29
**Status**: Draft
**Milestone**: Engineering Readiness

## Problem Statement

The project has reached feature maturity across multiple milestones (Sharing, Geo, Import/Export, UX Polish) but the engineering infrastructure hasn't kept pace. Concrete symptoms:

- **Build failures reach the user.** Vite only transpiles TypeScript — type errors pass silently. A recent bug shipped `tripName` to an API expecting `tripId` because nothing checked the types. (#128)
- **Schema changes break at runtime.** Adding a column to a store entity produces a 500 error because SQLite `CREATE TABLE IF NOT EXISTS` doesn't add columns. This has happened repeatedly (ShareLink, Place, Trip). (#92)
- **No structured CI.** `./dev ci` runs Go lint + build + test + e2e, but frontend type checking, Svelte warnings, and lint are absent. (#128, #130, #131)
- **Agent definitions are aspirational.** The existing agents (Commit, Code Review, Product Management, Session Summary) reference components and workflows that don't match the actual project structure (e.g., `packages/backend`, `./tc test backend`, Playwright tests). They were written for a different architecture.
- **No release process.** No versioning, no changelog, no deploy gates. (#125)

## Goals

### 1. Nothing Ships Broken

Every build artifact should be validated before the user sees it. Specifically:

- `./dev ci` catches **all** classes of errors: Go compilation, Go vet, Go tests, frontend TypeScript type errors, Svelte warnings, OpenAPI/codegen drift
- Schema changes are detected and applied automatically on startup (or fail explicitly with instructions)
- E2E smoke tests cover the critical path through CLI and API

### 2. Structured Development Workflow

Commits, reviews, and releases follow a predictable process that doesn't depend on memory:

- Pre-commit validation is automated and blocking
- Commit messages follow a consistent format
- Releases are tagged, changelogged, and deployable
- PRs (when used) have a standard template

### 3. Agents Work for Real

Agent definitions match the actual project, run efficiently, and are designed for a future where separate Claude Code sessions communicate through GitHub Issues:

- Each agent has a clear, minimal scope
- Agents read project state from files and issues, not conversation context
- The forge (GitHub Issues + PRs) is the coordination layer, not in-conversation delegation

---

## Plan

### Phase 1: Validation Pipeline (#128, #130, #131, #92)

Make `./dev ci` the single command that validates everything. If it passes, the code is shippable.

#### 1a. Frontend Type Checking (#128)

Add `svelte-check` to the devcontainer and wire it into the build:

```
./dev ci
  ├── go vet ./...
  ├── go build
  ├── go test ./...
  ├── frontend: svelte-check          ← NEW
  ├── frontend: vite build
  ├── codegen drift check (./dev lint-api)
  └── e2e smoke test
```

**Expected impact**: First run of `svelte-check` will surface existing type errors. These should be fixed as part of this work, not deferred.

#### 1b. Fix Svelte/Vite Warnings (#130)

Audit and fix all build warnings. Many are `a11y` ignores that should either be fixed or explicitly silenced. A clean build log means new warnings are immediately visible.

#### 1c. Linters (#131)

- **Go**: `go vet` is already in CI. Consider adding `staticcheck` or `golangci-lint` if warranted.
- **Frontend**: ESLint for TypeScript/Svelte. Start with the default Svelte config — don't over-customize.
- **OpenAPI**: `spectral` or equivalent to lint the API spec.

#### 1d. Database Schema Migration (#92)

The pragmatic fix for a single-user SQLite app:

**On startup**, compare the struct's `store` tags to the existing table's columns (via `PRAGMA table_info`). For any missing column, run `ALTER TABLE ADD COLUMN` with a sensible default. Log what was added.

This should be an appbase enhancement (`store.Collection` auto-migrates). If appbase can't be changed quickly, implement it in this app's `setup()` function as a temporary measure. File an appbase issue either way.

**What this does NOT cover**: column renames, type changes, column removal. Those still require manual migration. That's fine for now.

### Phase 1e: Predictable Runtime Modes (#82, #140)

Tests, agents, and scripts need to invoke the CLI and API predictably. Today the mode (local/remote), identity, and database path are all implicit.

#### Explicit mode flag (#82)

Add `--local` flag to force in-process single-user mode. Document auto-detection behavior. Resolve the three-database problem (CLI, server, and project root all use different paths).

#### Test identity (#140)

An `TRAVEL_TEST_USER` env var that provides a known identity for testing:
- CLI uses it instead of the default local user
- Server accepts a `X-Test-User` header when `TRAVEL_TEST_MODE=true`
- Never available in production

This unblocks API-level e2e tests and predictable agent invocations.

### Phase 1f: Sandboxing with nono (#141)

All development, testing, and agent sessions should run inside a nono sandbox with a project-specific capability profile.

#### Project profile

A `.nono/profile.json` checked into the repo that declares exactly what the project needs: project directory read/write, Go toolchain read, data directory write, port 3001 bind. No more ad-hoc `--allow` flags.

#### Agent profiles

Each agent type gets a constrained profile matching its role:
- **Validation**: read-only project + write test output
- **Commit**: read-write project + git network
- **Review**: read-only everything

This is a natural fit for forge-based coordination (#136) — each agent session launched with `nono run --profile .nono/agent-review.json -- claude -p "..."` gets exactly the capabilities it needs and nothing more.

#### Integration with ./dev

When nono is available, `./dev` wraps commands in the sandbox automatically. Tests verify they work inside the sandbox.

### Phase 2: E2E Testing (#124)

The current smoke test (`tests/e2e/smoke-test.sh`) covers basic CLI operations. Expand to:

#### 2a. CLI Scenario Tests

Shell scripts that exercise complete workflows:
- Trip lifecycle (create with dates/status → add activities → list → update → delete)
- Import workflow (add source → sync → list staged → import → verify)
- Share workflow (create link → verify access → revoke)
- Quick-add parser (exercise all patterns)

Each script is self-contained: creates its own data, verifies output, cleans up.

#### 2b. API Tests

For endpoints that the CLI doesn't fully exercise (overlay, public dashboard, display endpoint), add targeted HTTP tests. Could be shell scripts using `curl` against a running server, or Go integration tests.

#### 2c. Playwright (Deferred)

Browser tests are valuable but high-cost to set up and maintain. Defer until the UI is more stable. The CLI and API tests provide sufficient coverage for now.

### Phase 3: Release Process (#125)

#### Versioning

Semantic versioning: `v0.x.y` during development, `v1.0.0` when the PRD core use cases are complete.

#### Changelog

Auto-generated from conventional commits using a tool like `git-cliff` or a simple script that parses `feat:`, `fix:`, etc.

#### Release Workflow

```
1. ./dev ci passes
2. Tag: git tag v0.x.y
3. Changelog generated
4. Deploy: ./dev deploy (Cloud Run)
5. GitHub Release created with changelog
```

#### Deploy Gates

`./dev deploy` should refuse to deploy if:
- `./dev ci` hasn't passed on the current commit
- There are uncommitted changes
- The working tree is dirty

### Phase 4: Agent Redesign (#134 — new issue)

The existing agent definitions were written for a monorepo with `packages/` directories, `./tc` commands, and component-specific agents (Frontend Agent, Backend Agent, etc.) that don't exist. They need a ground-up rewrite.

#### Current Agents → Proposed Changes

| Agent | Current State | Proposed |
|-------|--------------|----------|
| **Commit** | 10-step workflow referencing Tools Agent, roadmap.md, ./tc test-precommit | Simplify: run `./dev ci`, review diff, commit. Drop references to nonexistent agents and files. |
| **Code Review** | References packages/, component agents, Playwright | Rewrite for actual project structure. Focus on: types match OpenAPI, no direct store access from CLI, test coverage. |
| **Product Management** | Comprehensive but references nonexistent structure | Keep scope, update file paths and component names. |
| **Session Summary** | Solid design, works as-is | Minor updates to match actual blog/ structure. |

#### New Agent: Validation

A dedicated agent whose only job is to run the full validation pipeline and report results. Other agents invoke it rather than each implementing their own checks.

```
Validation Agent
  Input: "validate current state"
  Output: pass/fail with details
  Runs: ./dev ci (captures output)
  Reports: structured result
```

#### Design for Forge-Based Coordination

The long-term vision: separate Claude Code runs, each with a specific agent, communicating through GitHub Issues. This requires:

1. **Issues as work items.** Each agent reads its assigned issues, performs work, and comments with results. The issue is the coordination artifact, not conversation context.

2. **Structured issue templates.** Agents need parseable inputs:
   ```
   ## Agent: validation
   ## Trigger: pre-commit
   ## Ref: commit abc123
   ## Files: server.go, store.go
   ```

3. **Agent-to-agent communication via issues.** Instead of in-conversation spawning:
   - Commit Agent creates a "review request" issue → Code Review Agent picks it up
   - Code Review Agent comments with findings → Commit Agent reads and proceeds

4. **Idempotent agents.** Each agent must be able to start fresh, read the issue, understand context from the repo state + issue description, and produce a result. No dependency on prior conversation.

5. **State in the repo, not in memory.** Agent progress, decisions, and artifacts live in files (`.claude/state/`, issue comments, PR descriptions) not in conversation context.

This is beyond the scope of this project but the agent rewrites should be designed with this in mind — agents that work from explicit inputs and produce explicit outputs are already forge-ready.

### Phase 5: Documentation (#126)

- **README.md**: Project overview, screenshots, setup instructions, CLI reference
- **CONTRIBUTING.md**: Development setup, PR process, agent workflow
- **Architecture decision records**: Key decisions documented (why SQLite, why appbase, why API-first)

---

## Issues

### Existing

| # | Title | Phase |
|---|-------|-------|
| 92 | Database migration strategy for schema changes | 1d |
| 124 | Ensure complete coverage of PRD use cases and e2e tests | 2 |
| 125 | Formal release processes and infrastructure | 3 |
| 126 | Product documentation | 5 |
| 128 | Add TypeScript type checking to frontend build | 1a |
| 130 | Fix warnings in vite / Svelte build | 1b |
| 131 | Add linters everywhere | 1c |

### New

| # | Title | Phase |
|---|-------|-------|
| 134 | Rewrite agent definitions for actual project structure | 4 |
| 135 | Add validation agent | 4 |
| 136 | Design agent communication via GitHub Issues | 4 |
| 137 | Deploy gates in ./dev deploy | 3 |
| 138 | Auto-migrate SQLite schema on startup | 1d |
| 139 | Fix existing TypeScript type errors from svelte-check | 1a |
| 140 | Test identity: ENV-based user for testing and CI | 1e |
| 82 | Polish and clarity on single-user mode | 1e |
| 141 | Implement nono sandboxing for agent and development sessions | 1f |

---

## Implementation Order

```
Phase 1d: Schema migration (unblocks everything — no more 500s)
Phase 1e: Runtime modes + test identity (unblocks testing and agent work)
Phase 1f: Sandboxing with nono (capability-based security for all sessions)
Phase 1a: svelte-check + fix type errors (biggest safety gap)
Phase 1b: Fix Svelte warnings (clean build log)
Phase 1c: Linters (incremental)
Phase 2a: CLI scenario tests (expand e2e coverage)
Phase 3:  Release process (versioning, changelog, deploy gates)
Phase 4:  Agent redesign (rewrite for reality, design for forge)
Phase 5:  Documentation (README, contributing guide)
Phase 2b: API tests (fill gaps)
```

Schema migration is first because it's the most frequent source of user-visible failures. Runtime mode clarity is second because it unblocks both testing and predictable agent invocations — two of the three core goals.

---

## Appendix: Relationship Between API Spec, Database Schema, and Codegen

### Current Architecture

```
openapi.yaml  ──codegen──→  api/server.gen.go (Go types + interface)
                             api/client.gen.go (Go HTTP client)
                             frontend/src/lib/api-types.ts (TS types)

Go structs    ──store tags──→  SQLite schema (CREATE TABLE)
(store.go)                     migration (ALTER TABLE ADD COLUMN)
```

These are **two separate sources of truth** kept in sync manually. Adding `status` to Trip requires editing both `openapi.yaml` and `store.go`, then running codegen. Forgetting one breaks things silently or at runtime.

### Why They're Separate (Correctly)

The store entity and the API type are not the same thing:

- `TripSummary` has computed fields (`activityCount`, `locations`) that don't exist in storage
- `SharedActivity` strips fields (`notes`, `userId`) for privacy
- `Activity` uses `openapi_types.Date` in the API but `string` in the store
- `ParseResult` is entirely ephemeral — no storage at all

The store is the persistence model. The API is the contract. They *should* be separate.

### Established Patterns

| Pattern | Example | Single Source? | Trade-off |
|---------|---------|---------------|-----------|
| **Model-first** | Rails, Django | Model drives schema + serializers | Tight coupling, less API control |
| **Schema-first** | Prisma | `.prisma` generates DB client + types | Good for CRUD, less flexible for computed fields |
| **DB-first** | PostGraphile, Hasura | DB schema generates API | Minimal code, but API is constrained by schema |
| **Proto-first** | gRPC + Protobuf | `.proto` generates client/server | Storage still usually separate |
| **API-first** | This project | OpenAPI drives contract, store is separate | Most flexible, most manual |

### What This Means for Engineering Readiness

The two-source design is correct. The problems we've hit are not architectural — they're operational:

1. **Schema drift** (store struct gains fields, table doesn't) → solved by auto-migration (#138)
2. **Type drift** (API type has field, store doesn't) → caught at compile time in Go (server handler won't compile if it can't read the field)
3. **Frontend type drift** (API changes, TS types stale) → solved by codegen + svelte-check (#128)

The one remaining gap: there's no automated check that a new API field has a corresponding store field when persistence is required. This could be a lint rule in `./dev lint-api` — verify that `UpdateTripRequest` fields map to store struct fields — but the effort/value ratio is low given that Go compilation already catches most mismatches in the handler code.

### Decision

Keep the current API-first + separate store design. Close the operational gaps through auto-migration, type checking, and codegen drift detection rather than collapsing to a single source of truth.
