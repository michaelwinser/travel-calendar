<script lang="ts">
  import { onMount } from 'svelte';
  import {
    getSourceFilters, updateSourceFilters,
    listStagedEvents, importStagedEvents, hideStagedEvents,
    syncSource,
    ACTIVITY_COLORS,
    type SourceFilter, type StagedEvent, type ImportSource, type ActivityType,
  } from '../lib/api';

  interface Props {
    source: ImportSource;
    onclose: () => void;
    onimported: () => void;
  }

  let { source, onclose, onimported }: Props = $props();

  let filters = $state<SourceFilter[]>([]);
  let events = $state<StagedEvent[]>([]);
  let selectedIds = $state<Set<string>>(new Set());
  let hoveredFilter = $state<string | null>(null);
  let newFilterPattern = $state('');
  let newFilterType = $state<'exclude' | 'include'>('exclude');
  let saving = $state(false);
  let error = $state('');
  let searchQuery = $state('');

  onMount(async () => {
    await refresh();
  });

  async function refresh() {
    try {
      [filters, events] = await Promise.all([
        getSourceFilters(source.id),
        listStagedEvents({ sourceId: source.id }),
      ]);
      error = '';
    } catch (e: any) {
      error = e.message;
    }
  }

  async function handleSync() {
    try {
      await syncSource(source.id);
      await refresh();
    } catch (e: any) {
      error = e.message;
    }
  }

  async function toggleFilter(idx: number) {
    filters[idx].enabled = !filters[idx].enabled;
    await saveFilters();
  }

  async function addFilter() {
    if (!newFilterPattern.trim()) return;
    filters = [...filters, {
      pattern: newFilterPattern.trim(),
      type: newFilterType,
      enabled: true,
      builtin: false,
    }];
    newFilterPattern = '';
    await saveFilters();
  }

  async function removeFilter(idx: number) {
    filters = filters.filter((_, i) => i !== idx);
    await saveFilters();
  }

  async function saveFilters() {
    saving = true;
    try {
      filters = await updateSourceFilters(source.id, filters);
    } catch (e: any) {
      error = e.message;
    }
    saving = false;
  }

  function matchesPattern(event: StagedEvent, pattern: string): boolean {
    const lower = pattern.toLowerCase();
    return (event.title?.toLowerCase().includes(lower)) ||
      (event.location?.toLowerCase().includes(lower));
  }

  // Events matching the hovered filter
  let highlightedIds = $derived.by(() => {
    if (!hoveredFilter) return new Set<string>();
    return new Set(events.filter(e => matchesPattern(e, hoveredFilter!)).map(e => e.id));
  });

  // Search-filtered events
  let filteredEvents = $derived.by(() => {
    let result = events;
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      result = result.filter(e =>
        e.title.toLowerCase().includes(q) ||
        (e.location && e.location.toLowerCase().includes(q)) ||
        (e.notes && e.notes.toLowerCase().includes(q))
      );
    }
    return result;
  });

  // Counts
  let excludeCount = $derived(filters.filter(f => f.type === 'exclude' && f.enabled).length);
  let includeCount = $derived(filters.filter(f => f.type === 'include' && f.enabled).length);

  async function handleImport() {
    if (selectedIds.size === 0) return;
    try {
      await importStagedEvents([...selectedIds]);
      await refresh();
      onimported();
    } catch (e: any) { error = e.message; }
  }

  async function handleHide() {
    if (selectedIds.size === 0) return;
    try {
      await hideStagedEvents([...selectedIds]);
      await refresh();
    } catch (e: any) { error = e.message; }
  }

  function toggleSelect(id: string) {
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id); else next.add(id);
    selectedIds = next;
  }

  function selectAllFiltered() {
    selectedIds = new Set(filteredEvents.filter(e => e.state === 'new').map(e => e.id));
  }

  function formatDate(start: string, end: string): string {
    if (!end || end === start) return start;
    return `${start} → ${end}`;
  }
</script>

<div class="fullscreen">
  <div class="toolbar">
    <h2>{source.name}</h2>
    <span class="event-count">{events.length} events ({filteredEvents.length} shown)</span>
    <div class="toolbar-spacer"></div>
    <button class="btn" onclick={handleSync}>Sync</button>
    <button class="btn btn-close" onclick={onclose}>Close</button>
  </div>

  {#if error}
    <p class="error">{error}</p>
  {/if}

  <div class="layout">
    <!-- Left: Events list -->
    <div class="events-pane">
      <div class="events-toolbar">
        <input type="text" class="search" placeholder="Search events..." bind:value={searchQuery} />
        <div class="bulk">
          <button class="btn-sm" onclick={selectAllFiltered} disabled={filteredEvents.length === 0}>
            Select new ({filteredEvents.filter(e => e.state === 'new').length})
          </button>
          <button class="btn-sm btn-primary-sm" onclick={handleImport} disabled={selectedIds.size === 0}>
            Import ({selectedIds.size})
          </button>
          <button class="btn-sm" onclick={handleHide} disabled={selectedIds.size === 0}>Hide</button>
        </div>
      </div>

      <div class="events-list">
        {#each filteredEvents as event (event.id)}
          <label
            class="event-row"
            class:highlighted={highlightedIds.has(event.id)}
            class:imported={event.state === 'imported'}
            class:hidden-ev={event.state === 'hidden'}
          >
            <input type="checkbox"
              checked={selectedIds.has(event.id)}
              onchange={() => toggleSelect(event.id)}
              disabled={event.state === 'imported'} />
            <span class="dot" style="background: {ACTIVITY_COLORS[event.type as ActivityType] ?? '#999'}"></span>
            <span class="ev-date">{formatDate(event.startDate, event.endDate)}</span>
            <span class="ev-title">{event.title}</span>
            {#if event.location}
              <span class="ev-loc">{event.location}</span>
            {/if}
            {#if event.state !== 'new'}
              <span class="state-tag" class:tag-imported={event.state === 'imported'} class:tag-hidden={event.state === 'hidden'}>
                {event.state}
              </span>
            {/if}
          </label>
        {/each}
      </div>
    </div>

    <!-- Right: Filters -->
    <div class="filters-pane">
      <h3>Filters <span class="filter-summary">{excludeCount} exclude, {includeCount} include</span></h3>

      <div class="filter-section">
        <h4>Exclude patterns</h4>
        {#each filters as filter, idx}
          {#if filter.type === 'exclude'}
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="filter-row"
              onmouseenter={() => hoveredFilter = filter.pattern}
              onmouseleave={() => hoveredFilter = null}
            >
              <input type="checkbox" checked={filter.enabled} onchange={() => toggleFilter(idx)} />
              <span class="filter-pattern" class:disabled={!filter.enabled}>{filter.pattern}</span>
              {#if filter.builtin}
                <span class="builtin-tag">built-in</span>
              {:else}
                <button class="remove-btn" onclick={() => removeFilter(idx)}>&times;</button>
              {/if}
            </div>
          {/if}
        {/each}
      </div>

      <div class="filter-section">
        <h4>Include patterns</h4>
        {#each filters as filter, idx}
          {#if filter.type === 'include'}
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="filter-row"
              onmouseenter={() => hoveredFilter = filter.pattern}
              onmouseleave={() => hoveredFilter = null}
            >
              <input type="checkbox" checked={filter.enabled} onchange={() => toggleFilter(idx)} />
              <span class="filter-pattern" class:disabled={!filter.enabled}>{filter.pattern}</span>
              {#if filter.builtin}
                <span class="builtin-tag">built-in</span>
              {:else}
                <button class="remove-btn" onclick={() => removeFilter(idx)}>&times;</button>
              {/if}
            </div>
          {/if}
        {/each}
      </div>

      <div class="add-filter">
        <input type="text" placeholder="New pattern..." bind:value={newFilterPattern}
          onkeydown={(e) => e.key === 'Enter' && addFilter()} />
        <select bind:value={newFilterType}>
          <option value="exclude">Exclude</option>
          <option value="include">Include</option>
        </select>
        <button class="btn-sm" onclick={addFilter} disabled={!newFilterPattern.trim()}>Add</button>
      </div>

      <p class="hint">
        Hover a filter to highlight matching events. Filters apply on the next sync.
      </p>
    </div>
  </div>
</div>

<style>
  .fullscreen {
    position: fixed; inset: 0; z-index: 200;
    background: #f8f9fa; display: flex; flex-direction: column;
  }

  .toolbar {
    display: flex; align-items: center; gap: 0.75rem;
    padding: 0.75rem 1.25rem; background: white;
    border-bottom: 1px solid #eee;
  }
  .toolbar h2 { margin: 0; font-size: 1.1rem; }
  .event-count { font-size: 0.8rem; color: #888; }
  .toolbar-spacer { flex: 1; }

  .btn {
    padding: 0.3rem 0.75rem; border: 1px solid #ddd; border-radius: 6px;
    background: white; font-size: 0.8rem; cursor: pointer; color: #555;
  }
  .btn:hover { background: #f5f5f5; color: #333; }
  .btn-close { font-weight: 600; }

  .error {
    color: #dc2626; font-size: 0.8rem; margin: 0;
    padding: 0.4rem 1.25rem; background: #fef2f2;
  }

  .layout {
    display: flex; flex: 1; overflow: hidden;
  }

  /* Left pane: events */
  .events-pane {
    flex: 1; display: flex; flex-direction: column;
    border-right: 1px solid #eee;
  }

  .events-toolbar {
    display: flex; align-items: center; gap: 0.5rem;
    padding: 0.5rem 0.75rem; background: white; border-bottom: 1px solid #eee;
    flex-wrap: wrap;
  }

  .search {
    flex: 1; min-width: 150px; padding: 0.3rem 0.5rem;
    border: 1px solid #ddd; border-radius: 6px; font-size: 0.8rem;
    font-family: inherit;
  }
  .search:focus { outline: none; border-color: #333; }

  .bulk { display: flex; gap: 0.3rem; }

  .btn-sm {
    padding: 0.2rem 0.5rem; border: 1px solid #ddd; border-radius: 4px;
    background: white; font-size: 0.7rem; cursor: pointer; color: #555;
  }
  .btn-sm:hover { border-color: #999; color: #333; }
  .btn-sm:disabled { opacity: 0.4; cursor: default; }
  .btn-primary-sm { background: #333; color: white; border-color: #333; }
  .btn-primary-sm:hover { background: #555; }

  .events-list {
    flex: 1; overflow-y: auto; background: white;
  }

  .event-row {
    display: flex; align-items: center; gap: 0.5rem;
    padding: 0.35rem 0.75rem; border-bottom: 1px solid #f3f4f6;
    font-size: 0.8rem; cursor: pointer;
    transition: background 0.1s;
  }
  .event-row:hover { background: #f8f9fa; }
  .event-row.highlighted { background: #fef3c7; }
  .event-row.imported { opacity: 0.5; }
  .event-row.hidden-ev { opacity: 0.35; text-decoration: line-through; }
  .event-row input { margin: 0; flex-shrink: 0; }

  .dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
  .ev-date { font-size: 0.7rem; color: #666; min-width: 75px; white-space: nowrap; flex-shrink: 0; }
  .ev-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 500; }
  .ev-loc { font-size: 0.7rem; color: #888; max-width: 150px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .state-tag {
    font-size: 0.6rem; padding: 0.05rem 0.3rem; border-radius: 3px;
    white-space: nowrap; flex-shrink: 0;
  }
  .tag-imported { background: #dcfce7; color: #166534; }
  .tag-hidden { background: #f3f4f6; color: #9ca3af; }

  /* Right pane: filters */
  .filters-pane {
    width: 320px; padding: 0.75rem; overflow-y: auto; background: white;
  }
  .filters-pane h3 {
    margin: 0 0 0.75rem; font-size: 0.95rem;
    display: flex; align-items: baseline; gap: 0.5rem;
  }
  .filter-summary { font-size: 0.7rem; color: #888; font-weight: 400; }

  .filter-section { margin-bottom: 1rem; }
  .filter-section h4 {
    font-size: 0.75rem; color: #999; text-transform: uppercase;
    letter-spacing: 0.05em; margin: 0 0 0.35rem; font-weight: 600;
  }

  .filter-row {
    display: flex; align-items: center; gap: 0.4rem;
    padding: 0.25rem 0; font-size: 0.8rem; cursor: default;
  }
  .filter-row input { margin: 0; }
  .filter-pattern { flex: 1; }
  .filter-pattern.disabled { color: #ccc; text-decoration: line-through; }

  .builtin-tag {
    font-size: 0.6rem; color: #aaa; background: #f3f4f6;
    padding: 0.05rem 0.25rem; border-radius: 3px;
  }

  .remove-btn {
    background: none; border: none; color: #ccc; cursor: pointer;
    font-size: 1rem; padding: 0 0.2rem; line-height: 1;
  }
  .remove-btn:hover { color: #dc2626; }

  .add-filter {
    display: flex; gap: 0.3rem; margin-top: 0.5rem;
  }
  .add-filter input {
    flex: 1; padding: 0.25rem 0.4rem; border: 1px solid #ddd;
    border-radius: 4px; font-size: 0.8rem; font-family: inherit;
  }
  .add-filter input:focus { outline: none; border-color: #333; }
  .add-filter select {
    padding: 0.25rem 0.3rem; border: 1px solid #ddd;
    border-radius: 4px; font-size: 0.75rem;
  }

  .hint {
    font-size: 0.7rem; color: #aaa; margin-top: 0.75rem;
  }
</style>
