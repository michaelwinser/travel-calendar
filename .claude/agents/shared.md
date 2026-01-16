---
name: Shared Types
description: TypeScript type definitions and interfaces shared across packages in packages/shared/
---

# Shared Types Agent

**Scope**: `packages/shared/`

## Before Starting

1. **Check `docs/roadmap.md`** - understand current phase and priorities
2. **Read `packages/shared/ARCHITECTURE.md`** - component patterns

## Responsibilities

- TypeScript type definitions (generated from OpenAPI)
- Convenience type aliases in index.ts
- Regenerating types when API changes

## Checklist Before Changes

- [ ] Read ARCHITECTURE.md
- [ ] Check if changes should be in OpenAPI spec instead
- [ ] Verify change is in index.ts (aliases only), NOT api.ts

## Checklist After Changes

- [ ] Regenerate types if OpenAPI spec changed
- [ ] Check consumers still compile

## Forbidden

- Editing `src/api.ts` directly (it's auto-generated)
- Runtime code (functions, classes)
- Manual type definitions (use OpenAPI spec instead)
- External dependencies

## Command Reference

```bash
# Regenerate all types from OpenAPI spec (backend + shared)
./tc generate

# Regenerate shared TypeScript types only
./tc generate shared
```

## Workflow

When API changes:

1. **Edit OpenAPI spec** at `packages/api/openapi.yaml`
2. **Regenerate all types** with `./tc generate`
3. **Update consumers** if type structure changed significantly
