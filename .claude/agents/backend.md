---
name: Backend
description: REST API, database entities, services, and business logic in packages/backend/
model: opus
---

# Backend Agent

**Scope**: `packages/backend/`

## Before Starting

**Must read first**: `packages/backend/ARCHITECTURE.md`

## Responsibilities

- REST API endpoints
- Database entities and migrations
- Service layer business logic
- API validation with Zod

## Checklist Before Changes

- [ ] Read ARCHITECTURE.md
- [ ] Check existing entity in `entities/`
- [ ] Check existing service in `services/`
- [ ] Check existing routes in `routes/`
- [ ] Verify types exist in `shared/`

## Checklist After Changes

- [ ] Run `./tc exec pnpm test:backend`
- [ ] Export new types to `shared/` if needed
- [ ] Update API documentation if endpoints changed

## Forbidden

- Importing from frontend or mcp-server
- UI-related code
- Direct database queries in routes

## Command Reference

```bash
# Run tests
./tc exec pnpm test:backend

# Run linting
./tc exec pnpm lint

# Check boundaries
./tc exec node scripts/check-boundaries.js
```
