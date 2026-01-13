# Frontend Architecture

**Read this file completely before making any changes to the frontend.**

## Overview

The frontend is a SvelteKit application following MVC principles:
- **Model**: Reactive stores connected to API
- **View**: Svelte components (presentation only)
- **Controller**: Route handlers and store actions

## Directory Structure

```
packages/frontend/
├── ARCHITECTURE.md           # This file - read first!
├── src/
│   ├── routes/               # SvelteKit routes (Controllers)
│   │   ├── +layout.svelte
│   │   ├── +page.svelte      # Dashboard
│   │   ├── calendar/
│   │   ├── trips/
│   │   │   ├── +page.svelte  # Trip list
│   │   │   ├── [id]/
│   │   │   │   └── +page.svelte
│   │   │   └── new/
│   │   └── documents/
│   ├── lib/
│   │   ├── stores/           # Model - reactive state
│   │   │   ├── trips.ts
│   │   │   ├── items.ts
│   │   │   └── documents.ts
│   │   ├── components/       # View - UI components
│   │   │   ├── trip/
│   │   │   │   ├── TripCard.svelte
│   │   │   │   ├── TripChip.svelte
│   │   │   │   ├── TripDetail.svelte
│   │   │   │   └── TripForm.svelte
│   │   │   ├── item/
│   │   │   │   ├── ItemCard.svelte
│   │   │   │   ├── FlightCard.svelte
│   │   │   │   ├── HotelCard.svelte
│   │   │   │   └── ...
│   │   │   ├── calendar/
│   │   │   └── ui/           # Generic UI primitives
│   │   ├── api/              # API client
│   │   │   └── client.ts
│   │   └── utils/
│   ├── app.html
│   └── app.css
├── tests/
│   └── components/
└── package.json
```

## Core Principles

### 1. Components Map to API Resources

Every API resource type has a corresponding component directory:

```
API Resource      →  Component Directory
/api/trips        →  lib/components/trip/
/api/items        →  lib/components/item/
/api/documents    →  lib/components/document/
```

### 2. Multiple Views per Resource

Each resource has multiple presentation forms. **Same data, different views.**

```
trip/
├── TripCard.svelte      # List view - medium detail
├── TripChip.svelte      # Inline reference - minimal (name + dates)
├── TripDetail.svelte    # Full page - all details
├── TripForm.svelte      # Edit form
└── TripCalendarBar.svelte  # Calendar view - just a colored bar
```

### 3. Reactive Data via Stores (NO ID LOOKUPS)

Components receive data via props or stores. **Never fetch by ID inside a component.**

```svelte
<!-- ✓ CORRECT: Data flows in via props -->
<script lang="ts">
  export let trip: Trip;
</script>

<!-- ✗ WRONG: Fetching inside component -->
<script lang="ts">
  export let tripId: string;
  let trip: Trip;
  onMount(async () => {
    trip = await api.getTrip(tripId);  // NO!
  });
</script>
```

### 4. Stores Are the Single Source of Truth

```typescript
// stores/trips.ts
import { writable, derived } from 'svelte/store';
import type { Trip } from '@travel-calendar/shared';
import { api } from '../api/client';

function createTripsStore() {
  const { subscribe, set, update } = writable<Trip[]>([]);

  return {
    subscribe,

    async load() {
      const trips = await api.trips.list();
      set(trips);
    },

    async create(input: CreateTripInput) {
      const trip = await api.trips.create(input);
      update(trips => [...trips, trip]);
      return trip;
    },

    // ... other actions
  };
}

export const trips = createTripsStore();

// Derived stores for filtered views
export const upcomingTrips = derived(trips, $trips =>
  $trips.filter(t => new Date(t.startDate) > new Date())
);
```

### 5. Components Are Self-Contained

Components handle their own:
- Styling (scoped CSS or Tailwind)
- Layout
- Loading/error states for their view

They do NOT:
- Fetch data
- Modify global state directly
- Know about routing

```svelte
<!-- TripCard.svelte - Self-contained -->
<script lang="ts">
  import type { Trip } from '@travel-calendar/shared';
  import TripChip from './TripChip.svelte';

  export let trip: Trip;
  export let onSelect: (trip: Trip) => void = () => {};
</script>

<article class="trip-card" on:click={() => onSelect(trip)}>
  <h3>{trip.name}</h3>
  <p>{trip.startDate} – {trip.endDate}</p>
  <!-- Can compose other views of same resource -->
</article>

<style>
  .trip-card { /* scoped styles */ }
</style>
```

### 6. Route Pages Orchestrate

Route `+page.svelte` files are orchestrators. They:
1. Load data into stores
2. Compose components
3. Handle navigation

```svelte
<!-- routes/trips/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { trips } from '$lib/stores/trips';
  import { goto } from '$app/navigation';
  import TripCard from '$lib/components/trip/TripCard.svelte';

  onMount(() => trips.load());

  function handleSelect(trip: Trip) {
    goto(`/trips/${trip.id}`);
  }
</script>

{#each $trips as trip (trip.id)}
  <TripCard {trip} onSelect={handleSelect} />
{/each}
```

## Component Patterns

### Props Interface

Every component exports a clear props interface:

```svelte
<script lang="ts">
  import type { Trip } from '@travel-calendar/shared';

  // Required props
  export let trip: Trip;

  // Optional props with defaults
  export let showDates: boolean = true;
  export let variant: 'default' | 'compact' = 'default';

  // Callbacks
  export let onSelect: (trip: Trip) => void = () => {};
  export let onEdit: (trip: Trip) => void = () => {};
</script>
```

### Event Handling

Components emit events via callback props, not custom events:

```svelte
<!-- ✓ Callback props (preferred) -->
<script>
  export let onDelete: (id: string) => void;
</script>
<button on:click={() => onDelete(trip.id)}>Delete</button>

<!-- ✗ Custom events (avoid) -->
<script>
  import { createEventDispatcher } from 'svelte';
  const dispatch = createEventDispatcher();
</script>
<button on:click={() => dispatch('delete', trip.id)}>Delete</button>
```

## Testing

Components are tested with Svelte Testing Library:

```typescript
// tests/components/TripCard.test.ts
import { render, screen } from '@testing-library/svelte';
import TripCard from '$lib/components/trip/TripCard.svelte';

const mockTrip = {
  id: '1',
  name: 'FOSDEM 2025',
  startDate: '2025-01-29',
  endDate: '2025-02-02',
  purpose: 'conference',
};

test('renders trip name', () => {
  render(TripCard, { props: { trip: mockTrip } });
  expect(screen.getByText('FOSDEM 2025')).toBeInTheDocument();
});
```

Run tests: `pnpm test:frontend`

## Forbidden Patterns

- ❌ Importing from `backend` or `mcp-server`
- ❌ Direct API calls inside components (use stores)
- ❌ ID-based lookups (`trips.find(t => t.id === tripId)`)
- ❌ Copying data between components (pass by reference)
- ❌ Business logic in components
- ❌ Global mutable state outside stores
- ❌ `createEventDispatcher` for parent communication (use callback props)
