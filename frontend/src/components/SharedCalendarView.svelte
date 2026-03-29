<script lang="ts">
  import { onMount } from 'svelte';
  import { today } from '../lib/date-utils';
  import { fetchSharedCalendar, fetchSharedWithMeCalendar, type Activity, type ActivityType, type TripSummary, type SharedCalendarResponse } from '../lib/api';
  import MonthView from './MonthView.svelte';
  import YearView from './YearView.svelte';
  import DayView from './DayView.svelte';
  import AgendaView from './AgendaView.svelte';

  type View = 'month' | 'year' | 'day' | 'agenda';

  interface Props {
    token?: string;
    email?: string;
  }

  let { token, email }: Props = $props();

  let data = $state<SharedCalendarResponse | null>(null);
  let loading = $state(true);
  let error = $state('');

  // --- URL routing: /shared/{token}[/view[/date]] or /view/{email}[/view[/date]] ---

  const basePath = token ? `/shared/${token}` : `/view/${email}`;

  function parseSharedURL(path: string): { view: View; date: string } {
    // Strip the /shared/{token} prefix
    const rest = path.slice(basePath.length).replace(/^\//, '');
    const parts = rest.split('/').filter(Boolean);
    const viewMap: Record<string, View> = { month: 'month', year: 'year', day: 'day', agenda: 'agenda' };
    const view = viewMap[parts[0]] ?? 'month';
    const date = parts[1] ?? today();
    const normalizedDate = date.length === 7 ? date + '-01' : date;
    return { view, date: normalizedDate };
  }

  function buildSharedURL(view: View, date: string): string {
    if (view === 'agenda') return `${basePath}/agenda`;
    if (view === 'month' || view === 'year') return `${basePath}/${view}/${date.slice(0, 7)}`;
    return `${basePath}/${view}/${date}`;
  }

  const { view: initialView, date: initialDate } = parseSharedURL(window.location.pathname);
  let currentView = $state<View>(initialView);
  let focusDate = $state(initialDate);

  // View refs for scroll actions
  let monthView = $state<MonthView>();
  let dayView = $state<DayView>();
  let yearView = $state<YearView>();

  // Track changes and push/replace URL
  let prevView = currentView;
  let prevFocusDate = focusDate;
  $effect(() => {
    if (currentView !== prevView) {
      prevView = currentView;
      prevFocusDate = focusDate;
      const url = buildSharedURL(currentView, focusDate);
      if (window.location.pathname !== url) {
        history.pushState({ view: currentView, date: focusDate }, '', url);
      }
    } else if (focusDate !== prevFocusDate) {
      prevFocusDate = focusDate;
      const url = buildSharedURL(currentView, focusDate);
      if (window.location.pathname !== url) {
        history.replaceState({ view: currentView, date: focusDate }, '', url);
      }
    }
  });

  function handlePopState(e: PopStateEvent) {
    if (e.state?.view && e.state?.date) {
      currentView = e.state.view;
      focusDate = e.state.date;
    } else {
      const { view, date } = parseSharedURL(window.location.pathname);
      currentView = view;
      focusDate = date;
    }
  }

  onMount(async () => {
    // Set initial history state with token-prefixed URL
    history.replaceState(
      { view: currentView, date: focusDate },
      '',
      buildSharedURL(currentView, focusDate),
    );

    try {
      data = token
        ? await fetchSharedCalendar(token)
        : await fetchSharedWithMeCalendar(email!);
    } catch (e: any) {
      error = e.message || 'Failed to load shared calendar';
    }
    loading = false;
  });

  // Map SharedActivity[] to Activity[] so existing views work
  let activities = $derived.by((): Activity[] => {
    if (!data) return [];
    return data.activities.map((a, i) => ({
      id: `shared-${i}`,
      userId: '',
      title: a.title || a.type,
      type: a.type as ActivityType,
      startDate: a.startDate,
      endDate: a.endDate,
      location: a.location,
      tripId: a.tripName ? `trip-${a.tripName}` : undefined,
      source: 'manual' as const,
      createdAt: '',
    }));
  });

  // Build TripSummary[] from unique trip names in the data
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
  function noop() {}
  function noopActivity(_activity: Activity) {}
  function noopDate(_date: string) {}
  function noopDragSelect(_start: string, _end: string) {}

  function scrollToToday() {
    focusDate = today();
    monthView?.scrollToToday();
    dayView?.scrollToToday();
    yearView?.scrollToToday();
  }

  function handleFocusDate(date: string) {
    focusDate = date;
  }

  function handleSwitchToMonth(date: string) {
    focusDate = date;
    currentView = 'month';
  }

  const views: { id: View; label: string }[] = [
    { id: 'month', label: 'Month' },
    { id: 'year', label: 'Year' },
    { id: 'day', label: 'Day' },
    { id: 'agenda', label: 'Agenda' },
  ];
</script>

<svelte:window onpopstate={handlePopState} />

<main class:wide={currentView === 'month' || currentView === 'year'}>
  <header>
    <div class="header-left">
      <h1>{data?.label ?? 'Shared Calendar'}</h1>
      {#if data?.ownerEmail}
        <span class="owner">{data.ownerEmail}</span>
      {/if}
    </div>
    <span class="badge">Read-only</span>
  </header>

  {#if loading}
    <p class="muted">Loading shared calendar...</p>
  {:else if error}
    <div class="error-card">
      <p>{error}</p>
    </div>
  {:else if data}
    <nav class="view-tabs">
      {#each views as v}
        <button
          class="tab"
          class:active={currentView === v.id}
          onclick={() => currentView = v.id}
        >{v.label}</button>
      {/each}
      <div class="tab-spacer"></div>
      {#if currentView === 'month' || currentView === 'day' || currentView === 'year'}
        <button class="today-btn" onclick={scrollToToday}>Today</button>
      {/if}
    </nav>

    {#if activities.length === 0}
      <p class="muted">No activities in this shared calendar.</p>
    {:else if currentView === 'month'}
      <MonthView
        bind:this={monthView}
        {activities}
        {trips}
        initialDate={focusDate}
        onedit={noopActivity}
        ondayclick={noopDate}
        ondragselect={noopDragSelect}
        onfocusdate={handleFocusDate}
      />
    {:else if currentView === 'year'}
      <YearView
        bind:this={yearView}
        {activities}
        {trips}
        initialDate={focusDate}
        onedit={noopActivity}
        ondayclick={noopDate}
        ondragselect={noopDragSelect}
        onswitchtomonth={handleSwitchToMonth}
        onfocusdate={handleFocusDate}
      />
    {:else if currentView === 'day'}
      <DayView
        bind:this={dayView}
        {activities}
        {trips}
        initialDate={focusDate}
        onedit={noopActivity}
        ondayclick={noopDate}
        ondragselect={noopDragSelect}
        onfocusdate={handleFocusDate}
      />
    {:else if currentView === 'agenda'}
      <AgendaView {activities} {trips} onedit={noopActivity} />
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

  .header-left {
    display: flex;
    align-items: baseline;
    gap: 0.75rem;
  }

  h1 {
    font-size: 1.5rem;
    margin: 0;
  }

  .owner {
    color: #666;
    font-size: 0.85rem;
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

  .view-tabs {
    display: flex;
    gap: 0;
    margin-bottom: 1rem;
    border-bottom: 2px solid #eee;
    align-items: center;
  }

  .tab {
    padding: 0.5rem 1rem;
    border: none;
    background: none;
    font-size: 0.9rem;
    color: #888;
    cursor: pointer;
    border-bottom: 2px solid transparent;
    margin-bottom: -2px;
  }

  .tab:hover { color: #555; }
  .tab.active {
    color: #333;
    border-bottom-color: #333;
    font-weight: 600;
  }

  .tab-spacer { flex: 1; }

  .today-btn {
    padding: 0.3rem 0.75rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    background: white;
    font-size: 0.8rem;
    cursor: pointer;
    color: #555;
    margin-bottom: 2px;
  }

  .today-btn:hover {
    background: #f5f5f5;
    color: #333;
  }
</style>
