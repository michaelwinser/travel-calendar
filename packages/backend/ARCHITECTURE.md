# Backend Architecture

**Read this file completely before making any changes to the backend.**

## Overview

The backend is a REST API built with:
- **Hono** - lightweight HTTP framework
- **Drizzle ORM** - type-safe database access
- **SQLite** - local-first database
- **Zod** - request/response validation

## Directory Structure

```
packages/backend/
├── ARCHITECTURE.md          # This file - read first!
├── src/
│   ├── index.ts             # App entry point, server setup
│   ├── routes/              # Route handlers by resource
│   │   ├── trips.ts         # /api/trips/*
│   │   ├── items.ts         # /api/items/*
│   │   └── documents.ts     # /api/documents/*
│   ├── entities/            # Database entities (Drizzle schemas)
│   │   ├── trip.ts
│   │   ├── item.ts
│   │   └── document.ts
│   ├── services/            # Business logic
│   │   ├── trip.service.ts
│   │   ├── item.service.ts
│   │   └── document.service.ts
│   ├── db/
│   │   ├── index.ts         # Database connection
│   │   ├── schema.ts        # Combined schema export
│   │   └── migrations/      # SQL migrations
│   └── lib/                 # Utilities
│       ├── errors.ts        # Error types
│       └── validation.ts    # Zod schemas
├── tests/
│   ├── trips.test.ts
│   ├── items.test.ts
│   └── fixtures/            # Test data
└── package.json
```

## Core Principles

### 1. Resource-Oriented REST

Every endpoint maps to a resource. No RPC-style endpoints.

```
✓ GET    /api/trips              # List trips
✓ POST   /api/trips              # Create trip
✓ GET    /api/trips/:id          # Get trip
✓ PATCH  /api/trips/:id          # Update trip
✓ DELETE /api/trips/:id          # Delete trip
✓ GET    /api/trips/:id/items    # List items for trip

✗ POST   /api/trips/:id/addFlight    # NO - RPC style
✗ GET    /api/getUpcomingTrips       # NO - RPC style
```

### 2. Entity Design

Entities are database-focused. They define:
- Table schema (Drizzle)
- Database constraints
- Indexes

```typescript
// entities/trip.ts
import { sqliteTable, text } from 'drizzle-orm/sqlite-core';

export const trips = sqliteTable('trips', {
  id: text('id').primaryKey(),
  name: text('name').notNull(),
  purpose: text('purpose').notNull(),
  startDate: text('start_date').notNull(),
  endDate: text('end_date').notNull(),
  // ... database columns only
});
```

### 3. Services Own Business Logic

Services contain all business logic. Routes are thin - they:
1. Parse request
2. Call service
3. Return response

```typescript
// routes/trips.ts - THIN
app.post('/api/trips', async (c) => {
  const body = await c.req.json();
  const validated = CreateTripSchema.parse(body);
  const trip = await tripService.create(validated);
  return c.json(trip, 201);
});

// services/trip.service.ts - LOGIC HERE
export class TripService {
  async create(input: CreateTripInput): Promise<Trip> {
    // Validation, business rules, database operations
  }
}
```

### 4. Validation at Boundaries

All input is validated with Zod at the route level:

```typescript
// lib/validation.ts
export const CreateTripSchema = z.object({
  name: z.string().min(1).max(200),
  purpose: z.enum(['conference', 'work', 'vacation', 'family', 'personal']),
  startDate: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
  endDate: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
});
```

### 5. Consistent Error Responses

All errors follow this format:

```typescript
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input",
    "details": { ... }
  }
}
```

Error codes: `VALIDATION_ERROR`, `NOT_FOUND`, `CONFLICT`, `INTERNAL_ERROR`

## Adding a New Resource

1. **Create entity** in `entities/`
2. **Create service** in `services/`
3. **Create routes** in `routes/`
4. **Add validation schemas** in `lib/validation.ts`
5. **Export types** to `shared/` package
6. **Write tests** in `tests/`

## Testing

Tests use Vitest and test the service layer directly:

```typescript
// tests/trips.test.ts
describe('TripService', () => {
  it('creates a trip with valid input', async () => {
    const trip = await tripService.create({
      name: 'FOSDEM 2025',
      purpose: 'conference',
      startDate: '2025-01-29',
      endDate: '2025-02-02',
    });
    expect(trip.id).toBeDefined();
    expect(trip.name).toBe('FOSDEM 2025');
  });
});
```

Run tests: `pnpm test:backend`

## Forbidden Patterns

- ❌ Importing from `frontend` or `mcp-server`
- ❌ UI-related code (HTML, CSS, Svelte)
- ❌ Direct database queries in routes (use services)
- ❌ Business logic in entities
- ❌ RPC-style endpoints
- ❌ Returning database entities directly (map to response types)
