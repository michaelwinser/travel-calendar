# Shared Package Architecture

**Read this file completely before making any changes to shared.**

## Overview

The `shared` package contains **TypeScript types generated from the OpenAPI specification**. These types are auto-generated and should not be edited manually.

## Source of Truth

The OpenAPI specification (`packages/api/openapi.yaml`) is the single source of truth for all types. This ensures consistency between:
- Backend Go API (generates server types)
- CLI Go client (generates client types)
- Frontend TypeScript (generates via this package)
- MCP server tools

## Directory Structure

```
packages/shared/
├── ARCHITECTURE.md           # This file - read first!
├── src/
│   ├── index.ts              # Convenience exports and type aliases
│   └── api.ts                # Generated types (DO NOT EDIT)
└── package.json
```

## Regenerating Types

When the OpenAPI spec changes, regenerate the types:

```bash
# From packages/shared/
pnpm generate

# Or from root
npx openapi-typescript packages/api/openapi.yaml -o packages/shared/src/api.ts
```

## Available Types

The package exports convenience aliases for commonly used types:

```typescript
// Entity types
import type { Trip, Item, Document } from '@travel-calendar/shared';

// Enum types
import type { TripPurpose, TripStatus, ItemType } from '@travel-calendar/shared';

// Request types
import type { CreateTripRequest, UpdateTripRequest, CreateItemRequest } from '@travel-calendar/shared';

// Full API types (paths, operations, etc.)
import type { paths, components, operations } from '@travel-calendar/shared';
```

## Usage Examples

### Frontend

```typescript
import type { Trip, TripPurpose, CreateTripRequest } from '@travel-calendar/shared';

// Use in components
const trip: Trip = await fetchTrip(id);

// Use in forms
const newTrip: CreateTripRequest = {
  name: 'New Trip',
  purpose: 'vacation' as TripPurpose,
  status: 'planning',
};
```

### API Client Types

For strongly-typed API clients, use the generated `paths` and `operations`:

```typescript
import type { paths } from '@travel-calendar/shared';

type ListTripsResponse = paths['/api/trips']['get']['responses']['200']['content']['application/json'];
```

## Core Principles

### 1. Generated Types Only

The `api.ts` file is auto-generated. **Never edit it manually.**

### 2. Convenience Aliases in index.ts

`index.ts` provides friendly aliases for commonly used types:

```typescript
// In index.ts
export type Trip = components['schemas']['Trip'];
```

You can add more aliases as needed, but keep them simple re-exports.

### 3. No Runtime Code

This package contains types only - no functions, classes, or constants.

### 4. Minimal Dependencies

Only `openapi-typescript` as a dev dependency for generation.

## Workflow

1. **Edit OpenAPI spec** (`packages/api/openapi.yaml`)
2. **Validate spec** (`npx @redocly/cli lint packages/api/openapi.yaml`)
3. **Regenerate types** (`pnpm generate` in shared package)
4. **Update consumers** if type structure changed significantly

## Forbidden Patterns

- ❌ Editing `api.ts` directly
- ❌ Adding runtime code (functions, classes)
- ❌ Adding external dependencies
- ❌ Default exports
- ❌ Hand-written type definitions (use OpenAPI spec instead)
