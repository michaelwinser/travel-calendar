# MCP Server Architecture

**Read this file completely before making any changes to the MCP server.**

## Overview

The MCP server exposes travel calendar functionality to LLMs via the Model Context Protocol:
- **Tools** - Actions the LLM can take (CRUD operations, queries)
- **Resources** - Data the LLM can read (trip lists, documents)

## Directory Structure

```
packages/mcp-server/
├── ARCHITECTURE.md           # This file - read first!
├── src/
│   ├── index.ts              # Server entry point
│   ├── tools/                # Tool implementations
│   │   ├── trips.ts          # Trip-related tools
│   │   ├── items.ts          # Item-related tools
│   │   ├── documents.ts      # Document tools
│   │   └── calendar.ts       # Calendar integration tools
│   ├── resources/            # Resource providers
│   │   ├── trips.ts
│   │   └── documents.ts
│   └── lib/
│       ├── backend-client.ts # HTTP client for backend API
│       └── formatters.ts     # Response formatting for LLMs
├── tests/
│   └── tools/
└── package.json
```

## Core Principles

### 1. MCP Server is a Facade

The MCP server does NOT contain business logic. It:
1. Translates MCP tool calls to backend API calls
2. Formats responses for LLM consumption
3. Provides semantic tool descriptions

```typescript
// ✓ CORRECT: Delegate to backend
async function handleGetTrips(args: GetTripsArgs) {
  const trips = await backendClient.get('/api/trips', { params: args });
  return formatTripsForLLM(trips);
}

// ✗ WRONG: Business logic in MCP server
async function handleGetTrips(args: GetTripsArgs) {
  const db = getDatabase();
  const trips = await db.query.trips.findMany();  // NO direct DB!
  // ... filtering logic here  // NO business logic!
}
```

### 2. Tools Are Verb-Oriented

Tool names describe actions. Keep them simple and consistent.

```typescript
// Tool naming pattern: {verb}_{resource}[_{qualifier}]

// ✓ Good tool names
'get_trips'           // List/query trips
'get_trip'            // Get single trip
'create_trip'         // Create new trip
'update_trip'         // Update existing trip
'delete_trip'         // Delete trip
'search_trips'        // Full-text search
'get_trip_conflicts'  // Get calendar conflicts for a trip

// ✗ Bad tool names
'trips'               // Not a verb
'fetchAllTrips'       // camelCase
'trip_list'           // Noun-oriented
```

### 3. Tool Descriptions Are for LLMs

Write descriptions that help the LLM choose the right tool:

```typescript
{
  name: 'get_trips',
  description: `List trips with optional filters.

Use this tool to:
- Find upcoming trips: get_trips({ upcoming: true })
- Find trips to a location: get_trips({ location: "Brussels" })
- Find trips in a date range: get_trips({ dateRange: ["2025-01-01", "2025-03-31"] })

Returns an array of trips with their items.`,
  inputSchema: { ... }
}
```

### 4. Responses Are LLM-Friendly

Format responses for natural language generation, not raw JSON:

```typescript
// formatters.ts
export function formatTripsForLLM(trips: Trip[]): string {
  if (trips.length === 0) {
    return 'No trips found matching your criteria.';
  }

  return trips.map(trip => {
    const duration = daysBetween(trip.startDate, trip.endDate);
    const locations = trip.items
      .filter(i => i.type === 'hotel')
      .map(i => i.location)
      .join(' → ');

    return `**${trip.name}** (${trip.purpose})
${trip.startDate} to ${trip.endDate} (${duration} days)
Locations: ${locations || 'Not specified'}
Items: ${trip.items.length} (${summarizeItems(trip.items)})`;
  }).join('\n\n');
}
```

### 5. Resources for Static/Reference Data

Resources are for data the LLM might want to reference:

```typescript
// Resources (read-only reference data)
'trips://upcoming'      // Upcoming trips
'trips://list'          // All trips
'trips://{id}'          // Single trip detail
'documents://recent'    // Recently added documents
```

## Tool Implementation Pattern

```typescript
// tools/trips.ts
import { z } from 'zod';
import { backendClient } from '../lib/backend-client';
import { formatTripsForLLM, formatTripForLLM } from '../lib/formatters';

export const tripTools = [
  {
    name: 'get_trips',
    description: `List trips with optional filters.

Filters:
- upcoming: true/false - only future trips
- past: true/false - only completed trips
- purpose: conference|work|vacation|family|personal
- location: city name to search
- dateRange: [startDate, endDate] in YYYY-MM-DD format`,
    inputSchema: {
      type: 'object',
      properties: {
        upcoming: { type: 'boolean' },
        past: { type: 'boolean' },
        purpose: {
          type: 'string',
          enum: ['conference', 'work', 'vacation', 'family', 'personal']
        },
        location: { type: 'string' },
        dateRange: {
          type: 'array',
          items: { type: 'string' },
          minItems: 2,
          maxItems: 2
        }
      }
    },
    handler: async (args: unknown) => {
      const params = GetTripsSchema.parse(args);
      const trips = await backendClient.get('/api/trips', { params });
      return { content: [{ type: 'text', text: formatTripsForLLM(trips) }] };
    }
  },
  // ... more tools
];
```

## Testing

Test tools by verifying:
1. Correct backend API calls are made
2. Responses are properly formatted
3. Input validation works

```typescript
// tests/tools/trips.test.ts
import { describe, it, expect, vi } from 'vitest';
import { tripTools } from '../../src/tools/trips';
import { backendClient } from '../../src/lib/backend-client';

vi.mock('../../src/lib/backend-client');

describe('get_trips tool', () => {
  const getTool = tripTools.find(t => t.name === 'get_trips')!;

  it('calls backend with filters', async () => {
    vi.mocked(backendClient.get).mockResolvedValue([]);

    await getTool.handler({ upcoming: true, location: 'Brussels' });

    expect(backendClient.get).toHaveBeenCalledWith('/api/trips', {
      params: { upcoming: true, location: 'Brussels' }
    });
  });

  it('formats empty results', async () => {
    vi.mocked(backendClient.get).mockResolvedValue([]);

    const result = await getTool.handler({});

    expect(result.content[0].text).toContain('No trips found');
  });
});
```

Run tests: `pnpm test:mcp`

## Forbidden Patterns

- ❌ Direct database access (use backend API)
- ❌ Business logic (belongs in backend)
- ❌ UI code
- ❌ Modifying data without going through backend
- ❌ Raw JSON responses (format for LLM readability)
- ❌ Tools that combine multiple unrelated operations
