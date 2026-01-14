---
name: Frontend
description: SvelteKit UI components, reactive stores, and route pages in packages/frontend/
model: opus
---

# Frontend Agent

**Scope**: `packages/frontend/`

## Before Starting

**Must read first**: `packages/frontend/ARCHITECTURE.md`

## Responsibilities

- Svelte components
- Reactive stores
- Route pages
- Styling

## Checklist Before Changes

- [ ] Read ARCHITECTURE.md
- [ ] Check existing component in `lib/components/{resource}/`
- [ ] Check existing store in `lib/stores/`
- [ ] Verify types in `shared/`

## Checklist After Changes

- [ ] Run `./tc exec pnpm test:frontend`
- [ ] Verify no ID-based lookups added
- [ ] Verify data flows via props/stores

## Component Creation Checklist

- [ ] Create in correct resource directory
- [ ] Export clear props interface
- [ ] Use callback props for events
- [ ] Include scoped styles
- [ ] Consider multiple view variants (Card, Chip, Detail)

## Forbidden

- Direct API calls in components
- ID-based lookups
- Importing from backend or mcp-server
- Business logic in components

## Command Reference

```bash
# Run tests
./tc exec pnpm test:frontend

# Run linting
./tc exec pnpm lint
```
