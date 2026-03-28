<script lang="ts">
  import { onMount } from 'svelte';
  import {
    getGlobalFilters, updateGlobalFilters, applyGlobalFilters,
    listStagedEvents, importStagedEvents, hideStagedEvents, unhideStagedEvents,
    ACTIVITY_COLORS,
    type SourceFilter, type StagedEvent, type ActivityType,
  } from '../lib/api';

  interface Props {
    onclose: () => void;
    onimported: () => void;
  }

  let { onclose, onimported }: Props = $props();

  let filters = $state<SourceFilter[]>([]);
  let events = $state<StagedEvent[]>([]);
  let selectedIds = $state<Set<string>>(new Set());
  let hoveredFilter = $state<string | null>(null);
  let newHidePattern = $state('');
  let newSelectPattern = $state('');
  let saving = $state(false);
  let dirty = $state(false);
  let error = $state('');
  let searchQuery = $state('');
  let showHidden = $state(false);

  onMount(async () => {
    await refresh();
  });

  async function refresh() {
    try {
      [filters, events] = await Promise.all([
        getGlobalFilters(),
        listStagedEvents({}),
      ]);
      // Pre-select events matching "select" filters
      autoSelect();
      error = '';
    } catch (e: any) {
      error = e.message;
    }
  }

  function autoSelect() {
    const selectPatterns = filters
      .filter(f => (f.type === 'select' || f.type === 'include') && f.enabled)
      .map(f => f.pattern.toLowerCase());

    const ids = new Set<string>();
    for (const e of events) {
      if (e.state !== 'new') continue;
      for (const p of selectPatterns) {
        if (e.title?.toLowerCase().includes(p) || e.location?.toLowerCase().includes(p)) {
          ids.add(e.id);
          break;
        }
      }
    }
    selectedIds = ids;
  }

  async function handleSync() {
    try {
      await syncSource(source.id);
      await refresh();
    } catch (e: any) {
      error = e.message;
    }
  }

  function toggleFilter(idx: number) {
    filters[idx].enabled = !filters[idx].enabled;
    dirty = true;
  }

  function addHideFilter() {
    if (!newHidePattern.trim()) return;
    filters = [...filters, { pattern: newHidePattern.trim(), type: 'hide', enabled: true, builtin: false }];
    newHidePattern = '';
    dirty = true;
  }

  function addSelectFilter() {
    if (!newSelectPattern.trim()) return;
    filters = [...filters, { pattern: newSelectPattern.trim(), type: 'select', enabled: true, builtin: false }];
    newSelectPattern = '';
    dirty = true;
  }

  function removeFilter(idx: number) {
    filters = filters.filter((_, i) => i !== idx);
    dirty = true;
  }

  async function applyFilters() {
    saving = true;
    try {
      filters = await updateGlobalFilters(filters);
      await applyGlobalFilters();
      events = await listStagedEvents({}) ?? [];
      autoSelect();
      dirty = false;
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

  let highlightedIds = $derived.by(() => {
    if (!hoveredFilter) return new Set<string>();
    return new Set(events.filter(e => matchesPattern(e, hoveredFilter!)).map(e => e.id));
  });

  let filteredEvents = $derived.by(() => {
    let result = events;
    // Hide "hidden" events unless showHidden is on
    if (!showHidden) {
      result = result.filter(e => e.state !== 'hidden');
    }
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

  let counts = $derived.by(() => {
    let newCount = 0, hiddenCount = 0, importedCount = 0;
    for (const e of events) {
      if (e.state === 'new') newCount++;
      else if (e.state === 'hidden') hiddenCount++;
      else if (e.state === 'imported') importedCount++;
    }
    return { newCount, hiddenCount, importedCount, total: events.length };
  });

  let hideFilters = $derived(filters.filter(f => f.type === 'hide' || f.type === 'exclude'));
  let selectFilters = $derived(filters.filter(f => f.type === 'select' || f.type === 'include'));

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

  async function handleUnhide() {
    if (selectedIds.size === 0) return;
    try {
      await unhideStagedEvents([...selectedIds]);
      await refresh();
    } catch (e: any) { error = e.message; }
  }

  function toggleSelect(id: string) {
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id); else next.add(id);
    selectedIds = next;
  }

  function selectAll() {
    selectedIds = new Set(filteredEvents.filter(e => e.state !== 'imported').map(e => e.id));
  }

  function selectNone() {
    selectedIds = new Set();
  }

  function formatDate(start: string, end: string): string {
    if (!end || end === start) return start;
    return `${start} → ${end}`;
  }
</script>

<div class="fullscreen">
  <div class="toolbar">
    <h2>Import Filters</h2>
    <span class="counts">
      {counts.newCount} new · {counts.hiddenCount} hidden · {counts.importedCount} imported · {counts.total} total
    </span>
    <div class="toolbar-spacer"></div>
    <button class="close-btn" onclick={onclose}>&times;</button>
  </div>

  {#if error}
    <p class="error">{error}</p>
  {/if}

  <div class="layout">
    <!-- Left: Events list -->
    <div class="events-pane">
      <div class="events-toolbar">
        <input type="text" class="search" placeholder="Search events..." bind:value={searchQuery} />
        <label class="show-hidden-toggle">
          <input type="checkbox" bind:checked={showHidden} />
          Show hidden
        </label>
      </div>
      <div class="bulk-bar">
        <label class="select-all">
          <input type="checkbox"
            checked={selectedIds.size === filteredEvents.length && filteredEvents.length > 0}
            onchange={() => selectedIds.size === filteredEvents.length ? selectNone() : selectAll()} />
          {selectedIds.size}/{filteredEvents.length}
        </label>
        <div class="bulk-actions">
          <button class="btn-sm btn-primary-sm" onclick={handleImport} disabled={selectedIds.size === 0}>Import</button>
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
      <h3>Filters</h3>

      <div class="filter-section">
        <h4>Hide patterns</h4>
        <p class="filter-hint">Matching events are hidden from view.</p>
        {#each hideFilters as filter}
          {@const idx = filters.indexOf(filter)}
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
        {/each}
        <div class="add-inline">
          <input type="text" placeholder="Add hide pattern..." bind:value={newHidePattern}
            onkeydown={(e) => e.key === 'Enter' && addHideFilter()} />
          <button class="btn-sm" onclick={addHideFilter} disabled={!newHidePattern.trim()}>+</button>
        </div>
      </div>

      <div class="filter-section">
        <h4>Select patterns</h4>
        <p class="filter-hint">Matching events are pre-selected for import.</p>
        {#each selectFilters as filter}
          {@const idx = filters.indexOf(filter)}
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
        {/each}
        <div class="add-inline">
          <input type="text" placeholder="Add select pattern..." bind:value={newSelectPattern}
            onkeydown={(e) => e.key === 'Enter' && addSelectFilter()} />
          <button class="btn-sm" onclick={addSelectFilter} disabled={!newSelectPattern.trim()}>+</button>
        </div>
      </div>

      <button class="apply-btn" onclick={applyFilters} disabled={!dirty || saving}>
        {saving ? 'Applying...' : 'Apply Filters'}
      </button>

      <p class="hint">
        Hover a filter to highlight matching events. Click Apply to update event states.
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
  .counts { font-size: 0.75rem; color: #888; }
  .toolbar-spacer { flex: 1; }

  .close-btn {
    background: none; border: none; font-size: 1.5rem;
    cursor: pointer; color: #888; padding: 0 0.25rem; line-height: 1;
  }
  .close-btn:hover { color: #333; }

  .error {
    color: #dc2626; font-size: 0.8rem; margin: 0;
    padding: 0.4rem 1.25rem; background: #fef2f2;
  }

  .layout { display: flex; flex: 1; overflow: hidden; }

  .events-pane {
    flex: 1; display: flex; flex-direction: column;
    border-right: 1px solid #eee;
  }

  .events-toolbar {
    display: flex; align-items: center; gap: 0.5rem;
    padding: 0.5rem 0.75rem; background: white; border-bottom: 1px solid #eee;
  }

  .search {
    flex: 1; min-width: 150px; padding: 0.3rem 0.5rem;
    border: 1px solid #ddd; border-radius: 6px; font-size: 0.8rem;
    font-family: inherit;
  }
  .search:focus { outline: none; border-color: #333; }

  .show-hidden-toggle {
    display: flex; align-items: center; gap: 0.3rem;
    font-size: 0.75rem; color: #888; cursor: pointer; white-space: nowrap;
  }
  .show-hidden-toggle input { margin: 0; }

  .bulk-bar {
    display: flex; align-items: center; justify-content: space-between;
    padding: 0.35rem 0.75rem; background: #f8f9fa;
    border-bottom: 1px solid #eee;
  }

  .select-all {
    display: flex; align-items: center; gap: 0.3rem;
    font-size: 0.75rem; color: #666; cursor: pointer;
  }
  .select-all input { margin: 0; }

  .bulk-actions { display: flex; gap: 0.3rem; }

  .btn-sm {
    padding: 0.2rem 0.5rem; border: 1px solid #ddd; border-radius: 4px;
    background: white; font-size: 0.7rem; cursor: pointer; color: #555;
  }
  .btn-sm:hover { border-color: #999; color: #333; }
  .btn-sm:disabled { opacity: 0.4; cursor: default; }
  .btn-primary-sm { background: #333; color: white; border-color: #333; }
  .btn-primary-sm:hover { background: #555; }

  .events-list { flex: 1; overflow-y: auto; background: white; }

  .event-row {
    display: flex; align-items: center; gap: 0.5rem;
    padding: 0.35rem 0.75rem; border-bottom: 1px solid #f3f4f6;
    font-size: 0.8rem; cursor: pointer;
    transition: background 0.1s;
  }
  .event-row:hover { background: #f8f9fa; }
  .event-row.highlighted { background: #fef3c7; }
  .event-row.imported { opacity: 0.5; }
  .event-row.hidden-ev { opacity: 0.4; }
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

  .filters-pane {
    width: 320px; padding: 0.75rem; overflow-y: auto; background: white;
  }
  .filters-pane h3 { margin: 0 0 0.75rem; font-size: 0.95rem; }

  .filter-section { margin-bottom: 1rem; }
  .filter-section h4 {
    font-size: 0.75rem; color: #999; text-transform: uppercase;
    letter-spacing: 0.05em; margin: 0 0 0.15rem; font-weight: 600;
  }
  .filter-hint { font-size: 0.7rem; color: #bbb; margin: 0 0 0.35rem; }

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

  .add-inline { display: flex; gap: 0.3rem; margin-top: 0.35rem; }
  .add-inline input {
    flex: 1; padding: 0.25rem 0.4rem; border: 1px solid #ddd;
    border-radius: 4px; font-size: 0.8rem; font-family: inherit;
  }
  .add-inline input:focus { outline: none; border-color: #333; }

  .apply-btn {
    width: 100%; padding: 0.5rem; margin-top: 0.75rem;
    border: none; border-radius: 6px;
    background: #333; color: white; font-size: 0.85rem;
    font-family: inherit; cursor: pointer; font-weight: 500;
  }
  .apply-btn:hover { background: #555; }
  .apply-btn:disabled { opacity: 0.4; cursor: default; }

  .hint { font-size: 0.7rem; color: #aaa; margin-top: 0.75rem; }
</style>
