---
name: Shared Types
description: TypeScript type definitions and interfaces shared across packages in packages/shared/
---

# Shared Types Agent

**Scope**: `packages/shared/`

## Before Starting

**Must read first**: `packages/shared/ARCHITECTURE.md`

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
# Regenerate types from OpenAPI spec
./tc exec pnpm --filter @travel-calendar/shared generate

# Or manually
npx openapi-typescript packages/api/openapi.yaml -o packages/shared/src/api.ts

# Build shared types
./tc exec pnpm --filter @travel-calendar/shared build
```

## Workflow

When API changes:

1. **Edit OpenAPI spec** at `packages/api/openapi.yaml`
2. **Regenerate Go types** in backend and CLI
3. **Regenerate TypeScript types** with `pnpm generate` in shared
4. **Update consumers** if type structure changed significantly
