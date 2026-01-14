# Tooling Improvements and Frontend Embedded in Go Server

**Date**: 2026-01-14

## Summary

- Created SvelteKit frontend with embedded static serving via go:embed
- Unified to single multi-stage Dockerfile (Node→Go→Alpine runtime)
- Added Commit Agent with exclusive commit authority
- Added Tools Agent with PreToolUse audit hook for session analysis
- Cleaned up settings.local.json conflicts
- Added `tc test-precommit` command and husky pre-commit hook
- Added `STATIC_CACHE` env var for production cache control

## Architecture Changes

**Before**: Separate frontend and backend containers with hot reload
**After**: Single Go binary with embedded static files (~17MB runtime)

```
┌─────────────────────────────────┐
│ Go Backend (port 3000)          │
│ ├─ REST API (/api/*, /health)   │
│ └─ Static Files (/* embedded)  │
└─────────────────────────────────┘
```

## New Workflow

```bash
./tc build    # Builds frontend + backend in one multi-stage Docker build
./tc start    # Starts the unified backend
./tc test-precommit  # Runs pre-commit checks
```

## Commits

- `ec7817c` feat(frontend): implement SvelteKit UI with embedded static serving
- `9324d9d` feat(backend): add STATIC_CACHE env var for cache control
- `bfb3b04` feat(infra): add pre-commit checks and Commit Agent
- `5e0af64` refactor(infra): unify Dockerfiles into single multi-stage build
- `b9ffc4a` feat(infra): add Tools Agent and PreToolUse audit hook
