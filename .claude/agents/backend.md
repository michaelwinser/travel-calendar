---
name: Backend
description: REST API, database entities, services, and business logic in packages/backend/
model: opus
---

# Backend Agent (Go)

**Scope**: `packages/backend/`

## Before Starting

1. **Check `docs/roadmap.md`** - understand current phase and priorities
2. **Read `packages/backend/ARCHITECTURE.md`** - component patterns

## Responsibilities

- REST API endpoints (Chi router)
- Database entities (Go structs with DB tags)
- Service layer business logic
- HTTP handlers implementing generated ServerInterface

## Checklist Before Changes

- [ ] Read ARCHITECTURE.md
- [ ] Check OpenAPI spec at `packages/api/openapi.yaml`
- [ ] Check existing entity in `internal/entity/`
- [ ] Check existing service in `internal/service/`
- [ ] Check existing handlers in `internal/handler/`

## Checklist After Changes

- [ ] Run Go tests: `./tc test backend`
- [ ] Verify health check: `./tc curl backend:3000/health`
- [ ] If API changed, regenerate types: `./tc generate`

## Forbidden

- Importing from frontend or mcp-server
- UI-related code
- Direct database queries in handlers (use service layer)
- Editing `internal/api/openapi.gen.go` directly

## Command Reference

```bash
# Run tests
./tc test backend

# Regenerate OpenAPI types (backend + shared)
./tc generate

# Regenerate backend Go types only
./tc generate backend

# Check API endpoint
./tc curl backend:3000/api/trips

# Health check
./tc curl backend:3000/health
```
