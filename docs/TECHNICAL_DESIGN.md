# Travel Calendar - Technical Design

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Clients                                         │
├─────────────────────────────────┬───────────────────────────────────────────┤
│         MCP Client              │              Web Browser                   │
│   (Claude, other LLMs)          │                                           │
│                                 │                                           │
│   "When is my next trip?"       │    ┌─────────────────────────────────┐   │
│   "Find London receipts"        │    │      SvelteKit Web UI           │   │
│                                 │    │  - Calendar views               │   │
└────────────┬────────────────────┘    │  - Trip management              │   │
             │                         │  - Document upload              │   │
             │ stdio                   └──────────────┬──────────────────┘   │
             │                                        │                       │
             ▼                                        │ HTTP                  │
┌─────────────────────────────────┐                  │                       │
│        MCP Server               │                  │                       │
│   (Node.js + MCP SDK)           │◄─────────────────┘                       │
│                                 │                                           │
│   Tools:                        │                                           │
│   - get_trips, search_trips     │                                           │
│   - create_trip, update_trip    │                                           │
│   - get_documents               │                                           │
│   - get_calendar_conflicts      │                                           │
│                                 │                                           │
│   Resources:                    │                                           │
│   - trips://list                │                                           │
│   - trips://{id}                │                                           │
└────────────┬────────────────────┘                                           │
             │                                                                 │
             ▼                                                                 │
┌─────────────────────────────────────────────────────────────────────────────┤
│                           Core Service Layer                                 │
│                                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │ TripService  │  │  DocService  │  │CalendarService│ │ SearchService│    │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘    │
│         │                 │                 │                 │             │
│         ▼                 ▼                 ▼                 ▼             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Data Access Layer                             │   │
│  │                     (better-sqlite3 + Drizzle)                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
└────────────────────────────────────┼────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Storage                                         │
│                                                                              │
│    ┌─────────────────┐              ┌─────────────────┐                     │
│    │    SQLite DB    │              │   File System   │                     │
│    │                 │              │                 │                     │
│    │  - trips        │              │  /documents/    │                     │
│    │  - locations    │              │    ├── {tripId}/│                     │
│    │  - segments     │              │    │   ├── receipt.pdf               │
│    │  - documents    │              │    │   └── confirmation.pdf          │
│    │  (metadata)     │              │    └── unassociated/                 │
│    └─────────────────┘              └─────────────────┘                     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Data Model

### Database Schema (SQLite + Drizzle ORM)

```typescript
// src/lib/db/schema.ts

import { sqliteTable, text, integer, real } from 'drizzle-orm/sqlite-core';

// ============================================
// TRIPS
// ============================================

export const trips = sqliteTable('trips', {
  id: text('id').primaryKey(),  // nanoid or uuid
  name: text('name').notNull(),
  purpose: text('purpose', { enum: ['conference', 'work', 'vacation', 'family', 'personal'] }).notNull(),
  status: text('status', { enum: ['planned', 'confirmed', 'completed', 'cancelled'] }).default('planned'),
  startDate: text('start_date').notNull(),  // ISO date: "2025-01-29"
  endDate: text('end_date').notNull(),
  notes: text('notes'),
  createdAt: text('created_at').notNull(),
  updatedAt: text('updated_at').notNull(),
});

// ============================================
// LOCATIONS (a trip can have multiple locations)
// ============================================

export const locations = sqliteTable('locations', {
  id: text('id').primaryKey(),
  tripId: text('trip_id').notNull().references(() => trips.id, { onDelete: 'cascade' }),
  city: text('city').notNull(),
  country: text('country'),
  venue: text('venue'),  // Optional: "Marriott Downtown", "FOSDEM venue"
  startDate: text('start_date').notNull(),
  endDate: text('end_date').notNull(),
  sortOrder: integer('sort_order').default(0),
});

// ============================================
// TRAVEL SEGMENTS (flights, trains, drives)
// ============================================

export const segments = sqliteTable('segments', {
  id: text('id').primaryKey(),
  tripId: text('trip_id').notNull().references(() => trips.id, { onDelete: 'cascade' }),
  type: text('type', { enum: ['flight', 'train', 'drive', 'bus', 'ferry', 'other'] }).notNull(),
  fromLocation: text('from_location').notNull(),  // "EWR" or "New York"
  toLocation: text('to_location').notNull(),
  departureDate: text('departure_date').notNull(),
  departureTime: text('departure_time'),  // "14:30"
  arrivalDate: text('arrival_date'),
  arrivalTime: text('arrival_time'),
  carrier: text('carrier'),  // "United", "Eurostar"
  flightNumber: text('flight_number'),  // "UA123"
  confirmationNumber: text('confirmation_number'),
  sortOrder: integer('sort_order').default(0),
});

// ============================================
// DOCUMENTS
// ============================================

export const documents = sqliteTable('documents', {
  id: text('id').primaryKey(),
  tripId: text('trip_id').references(() => trips.id, { onDelete: 'set null' }),  // Nullable for unassociated docs
  type: text('type', { enum: ['confirmation', 'receipt', 'ticket', 'hotel', 'visa', 'insurance', 'other'] }).notNull(),
  name: text('name').notNull(),
  filePath: text('file_path').notNull(),  // Relative path in documents folder
  mimeType: text('mime_type'),
  fileSize: integer('file_size'),
  vendor: text('vendor'),  // "United", "Marriott"
  amount: real('amount'),  // For receipts
  currency: text('currency').default('USD'),
  documentDate: text('document_date'),  // Date on the document
  sourceEmail: text('source_email'),  // If captured from email
  notes: text('notes'),
  createdAt: text('created_at').notNull(),
});

// ============================================
// CALENDAR LINKS (for Google Calendar sync)
// ============================================

export const calendarLinks = sqliteTable('calendar_links', {
  id: text('id').primaryKey(),
  tripId: text('trip_id').notNull().references(() => trips.id, { onDelete: 'cascade' }),
  calendarId: text('calendar_id').notNull(),
  eventId: text('event_id').notNull(),
  syncedAt: text('synced_at').notNull(),
});

// ============================================
// Full-text search virtual table
// ============================================
// Created via raw SQL:
// CREATE VIRTUAL TABLE trips_fts USING fts5(name, notes, content=trips, content_rowid=rowid);
```

### TypeScript Types

```typescript
// src/lib/types.ts

export type TripPurpose = 'conference' | 'work' | 'vacation' | 'family' | 'personal';
export type TripStatus = 'planned' | 'confirmed' | 'completed' | 'cancelled';
export type SegmentType = 'flight' | 'train' | 'drive' | 'bus' | 'ferry' | 'other';
export type DocumentType = 'confirmation' | 'receipt' | 'ticket' | 'hotel' | 'visa' | 'insurance' | 'other';

export interface Trip {
  id: string;
  name: string;
  purpose: TripPurpose;
  status: TripStatus;
  startDate: string;
  endDate: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
  // Populated via joins
  locations?: Location[];
  segments?: Segment[];
  documents?: Document[];
}

export interface Location {
  id: string;
  tripId: string;
  city: string;
  country?: string;
  venue?: string;
  startDate: string;
  endDate: string;
  sortOrder: number;
}

export interface Segment {
  id: string;
  tripId: string;
  type: SegmentType;
  fromLocation: string;
  toLocation: string;
  departureDate: string;
  departureTime?: string;
  arrivalDate?: string;
  arrivalTime?: string;
  carrier?: string;
  flightNumber?: string;
  confirmationNumber?: string;
  sortOrder: number;
}

export interface Document {
  id: string;
  tripId?: string;
  type: DocumentType;
  name: string;
  filePath: string;
  mimeType?: string;
  fileSize?: number;
  vendor?: string;
  amount?: number;
  currency: string;
  documentDate?: string;
  sourceEmail?: string;
  notes?: string;
  createdAt: string;
}

// Input types for creation (without auto-generated fields)
export interface TripInput {
  name: string;
  purpose: TripPurpose;
  status?: TripStatus;
  startDate: string;
  endDate: string;
  notes?: string;
  locations?: Omit<Location, 'id' | 'tripId'>[];
  segments?: Omit<Segment, 'id' | 'tripId'>[];
}

export interface TripFilters {
  upcoming?: boolean;
  past?: boolean;
  status?: TripStatus;
  purpose?: TripPurpose;
  location?: string;
  dateRange?: [string, string];
  search?: string;
}
```

## MCP Server Implementation

### Server Setup

```typescript
// src/mcp/server.ts

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListResourcesRequestSchema,
  ListToolsRequestSchema,
  ReadResourceRequestSchema,
} from '@modelcontextprotocol/sdk/types.js';
import { TripService } from '../services/trip.service.js';
import { DocumentService } from '../services/document.service.js';
import { CalendarService } from '../services/calendar.service.js';

const tripService = new TripService();
const documentService = new DocumentService();
const calendarService = new CalendarService();

const server = new Server(
  { name: 'travel-calendar', version: '1.0.0' },
  { capabilities: { tools: {}, resources: {} } }
);

// ============================================
// TOOLS
// ============================================

server.setRequestHandler(ListToolsRequestSchema, async () => ({
  tools: [
    {
      name: 'get_trips',
      description: 'Get trips with optional filters. Returns upcoming trips by default.',
      inputSchema: {
        type: 'object',
        properties: {
          upcoming: { type: 'boolean', description: 'Only future trips' },
          past: { type: 'boolean', description: 'Only past trips' },
          status: { type: 'string', enum: ['planned', 'confirmed', 'completed', 'cancelled'] },
          purpose: { type: 'string', enum: ['conference', 'work', 'vacation', 'family', 'personal'] },
          location: { type: 'string', description: 'Filter by location/city name' },
          dateRange: {
            type: 'array',
            items: { type: 'string' },
            minItems: 2,
            maxItems: 2,
            description: 'Date range [startDate, endDate] in ISO format'
          },
          search: { type: 'string', description: 'Full-text search query' }
        }
      }
    },
    {
      name: 'get_trip',
      description: 'Get a single trip by ID with all details (locations, segments, documents)',
      inputSchema: {
        type: 'object',
        properties: {
          tripId: { type: 'string', description: 'Trip ID' }
        },
        required: ['tripId']
      }
    },
    {
      name: 'search_trips',
      description: 'Full-text search across trip names, locations, and notes',
      inputSchema: {
        type: 'object',
        properties: {
          query: { type: 'string', description: 'Search query' }
        },
        required: ['query']
      }
    },
    {
      name: 'create_trip',
      description: 'Create a new trip with locations and travel segments',
      inputSchema: {
        type: 'object',
        properties: {
          name: { type: 'string', description: 'Trip name (e.g., "FOSDEM 2025")' },
          purpose: { type: 'string', enum: ['conference', 'work', 'vacation', 'family', 'personal'] },
          startDate: { type: 'string', description: 'Start date (ISO format)' },
          endDate: { type: 'string', description: 'End date (ISO format)' },
          notes: { type: 'string' },
          locations: {
            type: 'array',
            items: {
              type: 'object',
              properties: {
                city: { type: 'string' },
                country: { type: 'string' },
                startDate: { type: 'string' },
                endDate: { type: 'string' }
              },
              required: ['city', 'startDate', 'endDate']
            }
          },
          segments: {
            type: 'array',
            items: {
              type: 'object',
              properties: {
                type: { type: 'string', enum: ['flight', 'train', 'drive', 'bus', 'ferry', 'other'] },
                fromLocation: { type: 'string' },
                toLocation: { type: 'string' },
                departureDate: { type: 'string' },
                carrier: { type: 'string' },
                flightNumber: { type: 'string' }
              },
              required: ['type', 'fromLocation', 'toLocation', 'departureDate']
            }
          }
        },
        required: ['name', 'purpose', 'startDate', 'endDate']
      }
    },
    {
      name: 'update_trip',
      description: 'Update an existing trip',
      inputSchema: {
        type: 'object',
        properties: {
          tripId: { type: 'string' },
          changes: {
            type: 'object',
            properties: {
              name: { type: 'string' },
              purpose: { type: 'string' },
              status: { type: 'string' },
              startDate: { type: 'string' },
              endDate: { type: 'string' },
              notes: { type: 'string' }
            }
          }
        },
        required: ['tripId', 'changes']
      }
    },
    {
      name: 'delete_trip',
      description: 'Delete a trip and all associated data',
      inputSchema: {
        type: 'object',
        properties: {
          tripId: { type: 'string' }
        },
        required: ['tripId']
      }
    },
    {
      name: 'get_documents',
      description: 'Get documents, optionally filtered by trip or type',
      inputSchema: {
        type: 'object',
        properties: {
          tripId: { type: 'string', description: 'Filter by trip ID' },
          type: { type: 'string', enum: ['confirmation', 'receipt', 'ticket', 'hotel', 'visa', 'insurance', 'other'] },
          unassociated: { type: 'boolean', description: 'Only documents not linked to any trip' }
        }
      }
    },
    {
      name: 'search_documents',
      description: 'Search documents by name, vendor, or notes',
      inputSchema: {
        type: 'object',
        properties: {
          query: { type: 'string' }
        },
        required: ['query']
      }
    },
    {
      name: 'get_calendar_conflicts',
      description: 'Find calendar events that conflict with a trip (requires calendar integration)',
      inputSchema: {
        type: 'object',
        properties: {
          tripId: { type: 'string' }
        },
        required: ['tripId']
      }
    },
    {
      name: 'suggest_trips_from_calendar',
      description: 'Analyze calendar for events with locations that might be trips',
      inputSchema: {
        type: 'object',
        properties: {
          dateRange: {
            type: 'array',
            items: { type: 'string' },
            minItems: 2,
            maxItems: 2
          }
        }
      }
    }
  ]
}));

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;

  switch (name) {
    case 'get_trips':
      return { content: [{ type: 'text', text: JSON.stringify(await tripService.getTrips(args), null, 2) }] };

    case 'get_trip':
      return { content: [{ type: 'text', text: JSON.stringify(await tripService.getTrip(args.tripId), null, 2) }] };

    case 'search_trips':
      return { content: [{ type: 'text', text: JSON.stringify(await tripService.searchTrips(args.query), null, 2) }] };

    case 'create_trip':
      return { content: [{ type: 'text', text: JSON.stringify(await tripService.createTrip(args), null, 2) }] };

    case 'update_trip':
      return { content: [{ type: 'text', text: JSON.stringify(await tripService.updateTrip(args.tripId, args.changes), null, 2) }] };

    case 'delete_trip':
      await tripService.deleteTrip(args.tripId);
      return { content: [{ type: 'text', text: 'Trip deleted successfully' }] };

    case 'get_documents':
      return { content: [{ type: 'text', text: JSON.stringify(await documentService.getDocuments(args), null, 2) }] };

    case 'search_documents':
      return { content: [{ type: 'text', text: JSON.stringify(await documentService.searchDocuments(args.query), null, 2) }] };

    case 'get_calendar_conflicts':
      return { content: [{ type: 'text', text: JSON.stringify(await calendarService.getConflicts(args.tripId), null, 2) }] };

    case 'suggest_trips_from_calendar':
      return { content: [{ type: 'text', text: JSON.stringify(await calendarService.suggestTrips(args.dateRange), null, 2) }] };

    default:
      throw new Error(`Unknown tool: ${name}`);
  }
});

// ============================================
// RESOURCES
// ============================================

server.setRequestHandler(ListResourcesRequestSchema, async () => ({
  resources: [
    { uri: 'trips://list', name: 'All Trips', description: 'List of all trips', mimeType: 'application/json' },
    { uri: 'trips://upcoming', name: 'Upcoming Trips', description: 'Future trips', mimeType: 'application/json' },
    { uri: 'documents://unassociated', name: 'Unassociated Documents', description: 'Documents not linked to trips', mimeType: 'application/json' },
  ]
}));

server.setRequestHandler(ReadResourceRequestSchema, async (request) => {
  const { uri } = request.params;

  if (uri === 'trips://list') {
    const trips = await tripService.getTrips({});
    return { contents: [{ uri, mimeType: 'application/json', text: JSON.stringify(trips, null, 2) }] };
  }

  if (uri === 'trips://upcoming') {
    const trips = await tripService.getTrips({ upcoming: true });
    return { contents: [{ uri, mimeType: 'application/json', text: JSON.stringify(trips, null, 2) }] };
  }

  if (uri.startsWith('trips://') && uri !== 'trips://list' && uri !== 'trips://upcoming') {
    const tripId = uri.replace('trips://', '');
    const trip = await tripService.getTrip(tripId);
    return { contents: [{ uri, mimeType: 'application/json', text: JSON.stringify(trip, null, 2) }] };
  }

  if (uri === 'documents://unassociated') {
    const docs = await documentService.getDocuments({ unassociated: true });
    return { contents: [{ uri, mimeType: 'application/json', text: JSON.stringify(docs, null, 2) }] };
  }

  throw new Error(`Unknown resource: ${uri}`);
});

// Start server
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error('Travel Calendar MCP server running on stdio');
}

main().catch(console.error);
```

### Trip Service

```typescript
// src/services/trip.service.ts

import { db } from '../lib/db/index.js';
import { trips, locations, segments, documents } from '../lib/db/schema.js';
import { eq, and, gte, lte, like, or, desc } from 'drizzle-orm';
import { nanoid } from 'nanoid';
import type { Trip, TripInput, TripFilters } from '../lib/types.js';

export class TripService {
  async getTrips(filters: TripFilters): Promise<Trip[]> {
    const conditions = [];
    const today = new Date().toISOString().split('T')[0];

    if (filters.upcoming) {
      conditions.push(gte(trips.startDate, today));
    }
    if (filters.past) {
      conditions.push(lte(trips.endDate, today));
    }
    if (filters.status) {
      conditions.push(eq(trips.status, filters.status));
    }
    if (filters.purpose) {
      conditions.push(eq(trips.purpose, filters.purpose));
    }
    if (filters.dateRange) {
      conditions.push(
        and(
          lte(trips.startDate, filters.dateRange[1]),
          gte(trips.endDate, filters.dateRange[0])
        )
      );
    }

    let result = await db.query.trips.findMany({
      where: conditions.length > 0 ? and(...conditions) : undefined,
      orderBy: [desc(trips.startDate)],
      with: {
        locations: true,
        segments: true,
      }
    });

    // Filter by location if specified (requires join)
    if (filters.location) {
      result = result.filter(trip =>
        trip.locations?.some(loc =>
          loc.city.toLowerCase().includes(filters.location!.toLowerCase())
        )
      );
    }

    return result;
  }

  async getTrip(tripId: string): Promise<Trip | null> {
    const result = await db.query.trips.findFirst({
      where: eq(trips.id, tripId),
      with: {
        locations: { orderBy: (loc, { asc }) => [asc(loc.sortOrder)] },
        segments: { orderBy: (seg, { asc }) => [asc(seg.sortOrder)] },
        documents: true,
      }
    });
    return result ?? null;
  }

  async searchTrips(query: string): Promise<Trip[]> {
    // Use FTS5 for full-text search
    const searchResults = await db.all(
      `SELECT id FROM trips_fts WHERE trips_fts MATCH ? ORDER BY rank`,
      [query]
    );

    const tripIds = searchResults.map((r: any) => r.id);
    if (tripIds.length === 0) return [];

    return this.getTrips({ /* filter by IDs */ });
  }

  async createTrip(input: TripInput): Promise<Trip> {
    const tripId = nanoid();
    const now = new Date().toISOString();

    await db.transaction(async (tx) => {
      // Insert trip
      await tx.insert(trips).values({
        id: tripId,
        name: input.name,
        purpose: input.purpose,
        status: input.status ?? 'planned',
        startDate: input.startDate,
        endDate: input.endDate,
        notes: input.notes,
        createdAt: now,
        updatedAt: now,
      });

      // Insert locations
      if (input.locations?.length) {
        await tx.insert(locations).values(
          input.locations.map((loc, i) => ({
            id: nanoid(),
            tripId,
            city: loc.city,
            country: loc.country,
            venue: loc.venue,
            startDate: loc.startDate,
            endDate: loc.endDate,
            sortOrder: i,
          }))
        );
      }

      // Insert segments
      if (input.segments?.length) {
        await tx.insert(segments).values(
          input.segments.map((seg, i) => ({
            id: nanoid(),
            tripId,
            type: seg.type,
            fromLocation: seg.fromLocation,
            toLocation: seg.toLocation,
            departureDate: seg.departureDate,
            departureTime: seg.departureTime,
            arrivalDate: seg.arrivalDate,
            arrivalTime: seg.arrivalTime,
            carrier: seg.carrier,
            flightNumber: seg.flightNumber,
            confirmationNumber: seg.confirmationNumber,
            sortOrder: i,
          }))
        );
      }
    });

    return this.getTrip(tripId) as Promise<Trip>;
  }

  async updateTrip(tripId: string, changes: Partial<TripInput>): Promise<Trip> {
    const now = new Date().toISOString();

    await db.update(trips)
      .set({ ...changes, updatedAt: now })
      .where(eq(trips.id, tripId));

    return this.getTrip(tripId) as Promise<Trip>;
  }

  async deleteTrip(tripId: string): Promise<void> {
    await db.delete(trips).where(eq(trips.id, tripId));
    // Cascade delete handles locations, segments
    // Documents are set to null (unassociated)
  }
}
```

## Web UI Structure

```
src/
├── lib/
│   ├── db/
│   │   ├── index.ts          # Database connection
│   │   └── schema.ts         # Drizzle schema
│   ├── services/
│   │   ├── trip.service.ts
│   │   ├── document.service.ts
│   │   └── calendar.service.ts
│   ├── components/
│   │   ├── calendar/
│   │   │   ├── YearView.svelte
│   │   │   ├── MonthsView.svelte    # Multi-month at-a-glance
│   │   │   ├── MonthGrid.svelte
│   │   │   └── TripBar.svelte       # Colored bar spanning trip dates
│   │   ├── trips/
│   │   │   ├── TripList.svelte
│   │   │   ├── TripCard.svelte
│   │   │   ├── TripDetail.svelte
│   │   │   ├── TripForm.svelte
│   │   │   └── SegmentTimeline.svelte
│   │   ├── documents/
│   │   │   ├── DocumentList.svelte
│   │   │   ├── DocumentCard.svelte
│   │   │   └── UploadZone.svelte
│   │   └── ui/
│   │       ├── Header.svelte
│   │       ├── Sidebar.svelte
│   │       └── Modal.svelte
│   └── types.ts
├── routes/
│   ├── +layout.svelte
│   ├── +page.svelte              # Dashboard / upcoming trips
│   ├── calendar/
│   │   └── +page.svelte          # Multi-month calendar view
│   ├── trips/
│   │   ├── +page.svelte          # Trip list
│   │   ├── new/+page.svelte      # Create trip
│   │   └── [id]/
│   │       ├── +page.svelte      # Trip detail
│   │       └── edit/+page.svelte # Edit trip
│   ├── documents/
│   │   └── +page.svelte          # Document management
│   └── api/
│       ├── trips/
│       │   ├── +server.ts        # GET/POST trips
│       │   └── [id]/+server.ts   # GET/PUT/DELETE trip
│       └── documents/
│           └── +server.ts
└── mcp/
    └── server.ts                 # MCP server entry point
```

## Calendar View Design

The "months at a glance" view shows trips as colored horizontal bars:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ◀ 2025                                                              ▶      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  JANUARY                                                                     │
│  ┌───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬...─┬───┐   │
│  │ 1 │ 2 │ 3 │ 4 │ 5 │ 6 │ 7 │ 8 │ 9 │10 │11 │12 │13 │14 │15 │   │31 │   │
│  ├───┴───┴───┴───┴───┴───┴───┴───┼───┴───┴───┴───┴───┴───┴───┴───┴───┤   │
│  │                                │░░░░░░░░░ La Ventana ░░░░░░░░░░░░░│   │
│  │                         NYC ■──┤                                   │   │
│  └────────────────────────────────┴───────────────────────────────────┘   │
│                                                                              │
│  FEBRUARY                                                                    │
│  ┌───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬...─┬───┐   │
│  │ 1 │ 2 │ 3 │ 4 │ 5 │ 6 │ 7 │ 8 │ 9 │10 │11 │12 │13 │14 │15 │   │28 │   │
│  ├───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┤   │
│  │                    Hartford ■                                      │   │
│  │                              ████ Westport ████                    │   │
│  └────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  MARCH                                                                       │
│  ...                                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

Each trip is a colored bar positioned on the date axis. Clicking a bar opens the trip detail.

## Project Structure

```
travel-calendar/
├── docs/
│   ├── PRD.md
│   └── TECHNICAL_DESIGN.md
├── packages/
│   ├── core/                    # Shared code
│   │   ├── src/
│   │   │   ├── db/
│   │   │   │   ├── index.ts
│   │   │   │   ├── schema.ts
│   │   │   │   └── migrations/
│   │   │   ├── services/
│   │   │   │   ├── trip.service.ts
│   │   │   │   ├── document.service.ts
│   │   │   │   └── calendar.service.ts
│   │   │   └── types.ts
│   │   └── package.json
│   │
│   ├── mcp-server/              # MCP server
│   │   ├── src/
│   │   │   └── index.ts
│   │   └── package.json
│   │
│   └── web/                     # SvelteKit UI
│       ├── src/
│       │   ├── lib/
│       │   ├── routes/
│       │   └── app.html
│       ├── package.json
│       └── svelte.config.js
│
├── data/                        # SQLite DB and documents
│   ├── travel.db
│   └── documents/
│
├── package.json                 # Workspace root
├── pnpm-workspace.yaml
└── tsconfig.json
```

## Implementation Order

### Phase 1: Foundation (MVP)

1. **Project setup**
   - pnpm workspace with three packages (core, mcp-server, web)
   - TypeScript configuration
   - SQLite + Drizzle setup

2. **Core data layer**
   - Database schema and migrations
   - TripService with CRUD operations
   - DocumentService basics

3. **MCP server**
   - Basic tools: get_trips, get_trip, create_trip, search_trips
   - Resource endpoints: trips://list, trips://upcoming
   - Test with Claude Desktop

4. **Web UI basics**
   - Trip list page
   - Trip create/edit forms
   - Basic calendar view (months at a glance)

5. **Document management**
   - File upload to local storage
   - Associate documents with trips
   - Document list view

### Phase 2: Calendar Integration

6. **Google Calendar OAuth**
   - Auth flow in web UI
   - Token storage

7. **Calendar reading**
   - Fetch events in date range
   - Conflict detection tool
   - Trip suggestions from calendar

### Phase 3: Intelligence

8. **Enhanced search**
   - FTS5 full-text search
   - Search across trips and documents

9. **Document parsing**
   - PDF text extraction
   - Email parsing (forwarded confirmations)
