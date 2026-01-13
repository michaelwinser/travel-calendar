# Travel Calendar - Product Requirements Document

## Vision

A personal travel management system centered on **Trips** as first-class entities, with two primary interfaces:
1. **MCP Server** for conversational interaction with LLMs (queries, suggestions, document retrieval)
2. **Web UI** for visual at-a-glance views and direct manipulation

## Problem Statement

Current tools fail at holistic trip management:
- **Google Calendar**: Views overloaded, editing cumbersome, no trip abstraction
- **Spreadsheets**: Work for visualization but poor for querying, no document attachment
- **Travel apps** (TripIt, etc.): Focus on itinerary, not personal planning; vendor lock-in
- **Email**: Receipts and confirmations scattered, hard to find later

The core issue: **No single system treats "Trip" as a first-class entity** that encompasses dates, locations, purpose, travel segments, documents, and expenses.

## Core Concept: The Trip

A **Trip** is the central data model:

```
Trip
├── Identity
│   ├── name: "FOSDEM 2025"
│   ├── purpose: conference | vacation | family | work
│   └── status: planned | confirmed | completed | cancelled
│
├── When & Where
│   ├── startDate: 2025-01-29
│   ├── endDate: 2025-02-02
│   └── locations: [
│         { city: "London", dates: "Jan 29" },
│         { city: "Brussels", dates: "Jan 30 - Feb 2" }
│       ]
│
├── Travel Segments
│   └── segments: [
│         { type: "flight", from: "EWR", to: "LHR", date: "Jan 29", flight: "UA123" },
│         { type: "train", from: "London", to: "Brussels", date: "Jan 30" },
│         { type: "flight", from: "BRU", to: "EWR", date: "Feb 2" }
│       ]
│
├── Documents
│   └── documents: [
│         { type: "confirmation", source: "email", file: "...", date: "..." },
│         { type: "receipt", vendor: "United", amount: 850.00, file: "..." },
│         { type: "hotel", name: "Hotel Metropole", file: "..." }
│       ]
│
├── Expenses (Phase 2+)
│   └── expenses: [...]
│
└── Metadata
    ├── calendarEventIds: [...]  // Linked Google Calendar events
    ├── createdAt, updatedAt
    └── notes: "..."
```

## User Interfaces

### 1. MCP Server (Primary Query Interface)

Natural language interaction with an LLM for:

**Querying**
- "When am I going to FOSDEM?"
- "What's my next trip?"
- "How many days am I traveling in Q1?"
- "When was I last in San Francisco?"

**Conflict Detection**
- "What home appointments do I have that conflict with my Brussels trip?"
- "Do I have any overlapping trips?"

**Document Retrieval**
- "Find all receipts for my London trip last October"
- "Show me my flight confirmation for FOSDEM"
- "What hotel am I staying at in Brussels?"

**Trip Management**
- "Create a trip to NYC next Tuesday through Thursday for the board meeting"
- "Move my Portland trip back one week"
- "Cancel the Denver trip"

**Suggestions**
- "Based on my calendar, it looks like you might be traveling to Raleigh April 9-11. Want me to create a trip?"

### 2. Web UI (Visual Interface)

**Calendar Views**
- **Months-at-a-glance**: See 3-6 months with trips visualized as colored bars/blocks
- **Year view**: Full year overview showing travel density
- **Month view**: Traditional calendar with trip overlays

**Trip Management**
- Trip list with filters (upcoming, past, by purpose)
- Trip detail view with all associated information
- Create/edit trip with form or quick entry
- Drag to adjust trip dates
- Visual timeline for multi-leg trips

**Document Management**
- Upload documents (drag-drop, file picker)
- Email forwarding address for automatic capture
- Document viewer with metadata editing
- Link documents to trips

**Dashboard**
- Upcoming trips
- Recent documents needing association
- Travel statistics (days traveled, locations visited)

## Data Sources & Integration

### Google Calendar (Read)
- Scan for events with location data
- Identify potential trips to suggest
- Detect conflicts with "home" events

### Google Calendar (Write, Optional)
- Create/update events for trips
- Use dedicated "Travel" calendar
- Sync trip changes bidirectionally

### Email Integration (Phase 2+)
- Forward travel emails to dedicated address
- Parse confirmation numbers, dates, amounts
- Auto-associate with trips by date/location matching

### Document Storage
- Local file storage initially
- Cloud storage (Google Drive, S3) later
- PDF parsing for key information extraction

## MCP Server Design

### Tools

```typescript
// Trip queries
get_trips(filters?: { upcoming?: boolean, past?: boolean, location?: string, purpose?: string, dateRange?: [string, string] })
get_trip(tripId: string)
search_trips(query: string)  // Full-text search

// Trip management
create_trip(trip: TripInput)
update_trip(tripId: string, changes: Partial<TripInput>)
delete_trip(tripId: string)

// Calendar integration
get_calendar_conflicts(tripId: string)  // Find home events that conflict
suggest_trips_from_calendar()  // Analyze calendar for potential trips

// Document operations
get_documents(tripId?: string, type?: string)
search_documents(query: string)
associate_document(documentId: string, tripId: string)
```

### Resources

```typescript
// Expose trip data as resources
trips://list
trips://{tripId}
trips://{tripId}/documents
documents://unassociated  // Documents not yet linked to a trip
calendar://conflicts/{tripId}
```

### Example Interactions

**User**: "When am I going to FOSDEM?"
**LLM** → calls `search_trips({ query: "FOSDEM" })`
**Response**: "You're attending FOSDEM from January 30 to February 2, 2025. You fly into London on January 29th, take the Eurostar to Brussels on the 30th, and fly home on February 2nd."

**User**: "What home appointments need rescheduling for my Brussels trip?"
**LLM** → calls `get_trip("fosdem-2025")` then `get_calendar_conflicts("fosdem-2025")`
**Response**: "You have 3 events during your Brussels trip: Dentist appointment (Jan 30), Team standup (Jan 31), and Coffee with Sarah (Feb 1). The dentist appointment is in-person and should be rescheduled."

**User**: "Find receipts from my London trip last October"
**LLM** → calls `search_trips({ location: "London", dateRange: ["2024-10-01", "2024-10-31"] })` then `get_documents(tripId, "receipt")`
**Response**: "I found 4 receipts from your London trip (Oct 15-18): United flight ($892), Marriott hotel ($645), Restaurant receipts ($234 total)."

## Technical Decisions

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| **Primary entity** | Trip | Everything relates to trips, not individual days |
| **Storage** | SQLite + file system | Local-first, portable, queryable |
| **MCP Server** | TypeScript/Node | First-class MCP SDK support |
| **Web UI** | SvelteKit | Lightweight, good DX, SSR capable |
| **Calendar** | Google Calendar API | Most common, good API |
| **Home days** | Implicit | Days without trips are "home" |

## Implementation Phases

### Phase 1: Core MVP
- [ ] Data model and SQLite storage
- [ ] MCP server with trip CRUD and queries
- [ ] Basic web UI: trip list, create/edit trip, calendar view
- [ ] Document upload and association
- [ ] JSON import/export

### Phase 2: Calendar Integration
- [ ] Google Calendar OAuth setup
- [ ] Read calendar for conflict detection
- [ ] Suggest trips from calendar events with locations
- [ ] Optional: write trips to dedicated calendar

### Phase 3: Document Intelligence
- [ ] Email forwarding for document capture
- [ ] PDF parsing for key fields (confirmation #, amounts)
- [ ] Auto-association of documents to trips
- [ ] Receipt/expense extraction

### Phase 4: Advanced Features
- [ ] Expense tracking and reporting
- [ ] Multi-device sync
- [ ] Sharing/collaboration
- [ ] Mobile app or PWA

## Success Criteria

1. **Query speed**: "When am I going to X?" answered in < 2 seconds
2. **Trip creation**: New trip created in < 30 seconds
3. **Document retrieval**: "Find receipt for X" returns results in < 3 seconds
4. **Conflict detection**: Accurately identifies calendar conflicts
5. **At-a-glance clarity**: Calendar view immediately shows travel periods

## Open Questions

1. **Location granularity**: City level? Venue level? Multiple locations per trip day?
2. **Multi-leg trips**: One trip with segments, or separate linked trips?
3. **Recurring trips**: Template system for regular travel (e.g., quarterly NYC visits)?
4. **Shared trips**: Family vacations with shared documents/expenses?
5. **Offline support**: How important is offline access?

## Appendix: Current Spreadsheet Analysis

The existing spreadsheet approach:
- **Daily table** (365 rows): Source of truth, easy editing via copy-paste
- **Month view**: Derived view using formulas for at-a-glance visualization
- **Apps Script**: Creates Google Calendar events

Key insight: The daily table was a workaround for lack of a proper Trip entity. With explicit Trips, the data entry becomes "create trip with date range" rather than "edit N daily rows."
