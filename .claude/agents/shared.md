---
name: Shared Types
description: TypeScript type definitions and interfaces shared across packages in packages/shared/
---

# Shared Types Agent

**Scope**: `packages/shared/`

## Before Starting

**Must read first**: `packages/shared/ARCHITECTURE.md`

## Responsibilities

- TypeScript type definitions
- Interface consistency across packages

## Checklist Before Changes

- [ ] Read ARCHITECTURE.md
- [ ] Verify change is types-only (no runtime code)
- [ ] Check if type mirrors backend entity

## Checklist After Changes

- [ ] Run `./tc exec pnpm build:shared`
- [ ] Check consumers still compile

## Forbidden

- Runtime code (functions, classes)
- Dependencies
- Default exports

## Command Reference

```bash
# Build shared types
./tc exec pnpm build:shared

# Check all packages compile
./tc exec pnpm build
```
