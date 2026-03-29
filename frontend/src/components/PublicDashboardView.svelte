<script lang="ts">
  import { onMount } from 'svelte';
  import { today } from '../lib/date-utils';
  import { fetchPublicDashboard, type Activity, type ActivityType, type TripSummary, type SharedCalendarResponse } from '../lib/api';
  import MonthView from './MonthView.svelte';

  interface Props {
    handle: string;
  }

  let { handle }: Props = $props();

  let data = $state<SharedCalendarResponse | null>(null);
  let loading = $state(true);
  let error = $state('');
  let monthView = $state<MonthView>();

  onMount(async () => {
    try {
      data = await fetchPublicDashboard(handle);
    } catch (e: any) {
      error = e.message || 'Failed to load public dashboard';
    }
    loading = false;
  });

  // Map SharedActivity[] to Activity[] for MonthView
  let activities = $derived.by((): Activity[] => {
    if (!data) return [];
    return data.activities.map((a, i) => ({
      id: `public-${i}`,
      userId: '',
      title: a.location || String(a.type),
      type: a.type as ActivityType,
      startDate: a.startDate,
      endDate: a.endDate,
      location: a.location,
      tripId: a.tripName ? `trip-${a.tripName}` : undefined,
      source: 'manual' as const,
      createdAt: '',
    }));
  });

  // Build TripSummary[] from unique trip names
  let trips = $derived.by((): TripSummary[] => {
    if (!data) return [];
    const tripMap = new Map<string, { color: string; activities: typeof data.activities }>();
    for (const a of data.activities) {
      if (!a.tripName) continue;
      const existing = tripMap.get(a.tripName);
      if (existing) {
        existing.activities.push(a);
      } else {
        tripMap.set(a.tripName, { color: a.tripColor ?? '#999', activities: [a] });
      }
    }
    const result: TripSummary[] = [];
    for (const [name, info] of tripMap) {
      const acts = info.activities;
      const startDate = acts.reduce((min, a) => a.startDate < min ? a.startDate : min, acts[0].startDate);
      const endDate = acts.reduce((max, a) => a.endDate > max ? a.endDate : max, acts[0].endDate);
      const locations = [...new Set(acts.map(a => a.location).filter(Boolean))] as string[];
      result.push({
        id: `trip-${name}`,
        name,
        color: info.color,
        status: 'confirmed',
        startDate,
        endDate,
        activityCount: acts.length,
        locations: locations.length > 0 ? locations : undefined,
      });
    }
    return result;
  });

  // No-op handlers for read-only mode
  function noopActivity(_activity: Activity) {}
  function noopDate(_date: string) {}
  function noopDragSelect(_start: string, _end: string) {}

  function scrollToToday() {
    monthView?.scrollToToday();
  }
</script>

<main class="wide">
  <header>
    <h1>{data?.label ?? `Where is ${handle}?`}</h1>
    <div class="header-right">
      <span class="badge">Public</span>
      <button class="today-btn" onclick={scrollToToday}>Today</button>
    </div>
  </header>

  {#if loading}
    <p class="muted">Loading...</p>
  {:else if error}
    <div class="error-card">
      <p>{error}</p>
    </div>
  {:else if data}
    {#if activities.length === 0}
      <p class="muted">No upcoming activities.</p>
    {:else}
      <MonthView
        bind:this={monthView}
        {activities}
        {trips}
        initialDate={today()}
        onedit={noopActivity}
        ondayclick={noopDate}
        ondragselect={noopDragSelect}
      />
    {/if}
  {/if}
</main>

<style>
  :global(body) {
    font-family: system-ui, -apple-system, sans-serif;
    margin: 0;
    padding: 0;
    background: #f8f9fa;
    color: #333;
  }

  main {
    max-width: 800px;
    margin: 0 auto;
    padding: 1.5rem 1rem;
  }

  main.wide {
    max-width: 1100px;
  }

  header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    margin-bottom: 1.5rem;
  }

  h1 {
    font-size: 1.5rem;
    margin: 0;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .badge {
    font-size: 0.7rem;
    color: #888;
    background: #f3f4f6;
    border: 1px solid #e5e7eb;
    border-radius: 4px;
    padding: 0.15rem 0.5rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .today-btn {
    padding: 0.3rem 0.75rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    background: white;
    font-size: 0.8rem;
    cursor: pointer;
    color: #555;
  }

  .today-btn:hover {
    background: #f5f5f5;
    color: #333;
  }

  .muted {
    color: #999;
    text-align: center;
    padding: 3rem 0;
  }

  .error-card {
    background: #fef2f2;
    border: 1px solid #fecaca;
    border-radius: 8px;
    padding: 1.5rem;
    text-align: center;
    color: #dc2626;
  }
</style>
