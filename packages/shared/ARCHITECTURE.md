# Shared Package Architecture

**Read this file completely before making any changes to shared.**

## Overview

The `shared` package contains **TypeScript types only**. No runtime code, no logic, no dependencies.

## Purpose

Provides type definitions that ensure consistency across:
- Backend API responses
- Frontend data models
- MCP server interfaces

## Directory Structure

```
packages/shared/
├── ARCHITECTURE.md           # This file - read first!
├── src/
│   ├── index.ts              # Main export
│   ├── trip.ts               # Trip types
│   ├── item.ts               # Item types (flight, hotel, etc.)
│   ├── document.ts           # Document types
│   └── api.ts                # API request/response types
├── package.json
└── tsconfig.json
```

## Core Principles

### 1. Types Only - No Runtime Code

```typescript
// ✓ ALLOWED: Type definitions
export interface Trip {
  id: string;
  name: string;
  purpose: TripPurpose;
}

export type TripPurpose = 'conference' | 'work' | 'vacation' | 'family' | 'personal';

// ✗ FORBIDDEN: Runtime code
export function formatTrip(trip: Trip): string {  // NO!
  return `${trip.name} (${trip.purpose})`;
}

// ✗ FORBIDDEN: Constants that could be types
export const TRIP_PURPOSES = ['conference', 'work', ...] as const;  // NO!
```

### 2. No Dependencies

`package.json` should have zero dependencies:

```json
{
  "name": "@travel-calendar/shared",
  "dependencies": {},
  "devDependencies": {
    "typescript": "^5.0.0"
  }
}
```

### 3. Entity Types Mirror Backend

Types should match backend entity shapes:

```typescript
// trip.ts - Mirrors backend/src/entities/trip.ts
export interface Trip {
  id: string;
  name: string;
  purpose: TripPurpose;
  status: TripStatus;
  startDate: string;  // ISO date
  endDate: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export type TripPurpose = 'conference' | 'work' | 'vacation' | 'family' | 'personal';
export type TripStatus = 'planned' | 'confirmed' | 'completed' | 'cancelled';
```

### 4. API Types for Request/Response

```typescript
// api.ts
import type { Trip, TripPurpose } from './trip';

// Input types (for creation/updates)
export interface CreateTripInput {
  name: string;
  purpose: TripPurpose;
  startDate: string;
  endDate: string;
  notes?: string;
}

export interface UpdateTripInput {
  name?: string;
  purpose?: TripPurpose;
  status?: TripStatus;
  startDate?: string;
  endDate?: string;
  notes?: string;
}

// Query types
export interface TripFilters {
  upcoming?: boolean;
  past?: boolean;
  purpose?: TripPurpose;
  location?: string;
  dateRange?: [string, string];
}

// Response types (if different from entity)
export interface TripWithItems extends Trip {
  items: Item[];
}
```

### 5. Discriminated Unions for Item Types

```typescript
// item.ts
interface BaseItem {
  id: string;
  tripId: string;
  date: string;
  createdAt: string;
}

export interface FlightItem extends BaseItem {
  type: 'flight';
  from: string;
  to: string;
  departureTime?: string;
  arrivalTime?: string;
  carrier?: string;
  flightNumber?: string;
  confirmation?: string;
}

export interface HotelItem extends BaseItem {
  type: 'hotel';
  name: string;
  location: string;
  checkIn: string;
  checkOut: string;
  confirmation?: string;
}

// ... other item types

export type Item = FlightItem | HotelItem | TrainItem | DriveItem | EventItem;
export type ItemType = Item['type'];
```

## Usage in Other Packages

```typescript
// In backend
import type { Trip, CreateTripInput } from '@travel-calendar/shared';

// In frontend
import type { Trip, TripWithItems } from '@travel-calendar/shared';

// In mcp-server
import type { Trip, TripFilters } from '@travel-calendar/shared';
```

## Adding New Types

1. Create type file in `src/`
2. Export from `src/index.ts`
3. Update consumers as needed

**Do not add:**
- Validation logic (use Zod in backend)
- Formatting functions (use component utilities)
- Constants (use type literals)

## Forbidden Patterns

- ❌ Any runtime code (functions, classes)
- ❌ Dependencies in package.json
- ❌ Importing from other packages
- ❌ `const` declarations (except `as const` for type inference)
- ❌ Default exports
- ❌ Re-exporting from external packages
