<script lang="ts">
  import { onMount } from 'svelte';
  import {
    listSources, createSource, syncSource, deleteSource,
    listStagedEvents, importStagedEvents, hideStagedEvents, unhideStagedEvents,
    ACTIVITY_COLORS,
    type ImportSource, type StagedEvent, type ActivityType,
  } from '../lib/api';

  interface Props {
    onclose: () => void;
    onimported: () => void; // callback when activities are imported (to refresh main view)
  }

  let { onclose, onimported }: Props = $props();

  // Sources
  let sources = $state<ImportSource[]>([]);
  let newSourceName = $state('');
  let newSourceURL = $state('');
  let addingSource = $state(false);

  // Staged events
  let stagedEvents = $state<StagedEvent[]>([]);
  let selectedIds = $state<Set<string>>(new Set());
  let stateFilter = $state<'' | 'new' | 'imported' | 'hidden'>('new');
  let sourceFilter = $state('');
  let loading = $state(true);
  let error = $state('');

  onMount(async () => {
    await refresh();
    loading = false;
  });

  async function refresh() {
    try {
      sources = await listSources();
      await refreshStaged();
      error = '';
    } catch (e: any) {
      error = e.message;
    }
  }

  async function refreshStaged() {
    const params: { sourceId?: string; state?: string } = {};
    if (stateFilter) params.state = stateFilter;
    if (sourceFilter) params.sourceId = sourceFilter;
    stagedEvents = await listStagedEvents(params) ?? [];
    selectedIds = new Set();
  }

  async function handleAddSource() {
    if (!newSourceName || !newSourceURL) return;
    addingSource = true;
    try {
      await createSource(newSourceName, newSourceURL);
      newSourceName = '';
      newSourceURL = '';
      await refresh();
    } catch (e: any) {
      error = e.message;
    }
    addingSource = false;
  }

  async function handleSync(id: string) {
    try {
      await syncSource(id);
      await refresh();
    } catch (e: any) {
      error = e.message;
    }
  }

  async function handleDeleteSource(id: string) {
    const src = sources.find(s => s.id === id);
    const hasImported = src && src.importedCount > 0;
    let deleteActivities = false;

    if (hasImported) {
      deleteActivities = confirm(
        `This source has ${src!.importedCount} imported activities.\n\nAlso delete those activities?`
      );
    }

    try {
      const url = `/api/sources/${id}${deleteActivities ? '?deleteActivities=true' : ''}`;
      const res = await fetch(url, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete source');
      await refresh();
      if (deleteActivities) onimported(); // refresh main view
    } catch (e: any) {
      error = e.message;
    }
  }

  async function handleImportSelected() {
    if (selectedIds.size === 0) return;
    try {
      await importStagedEvents([...selectedIds]);
      await refresh();
      onimported();
    } catch (e: any) {
      error = e.message;
    }
  }

  async function handleHideSelected() {
    if (selectedIds.size === 0) return;
    try {
      await hideStagedEvents([...selectedIds]);
      await refresh();
    } catch (e: any) {
      error = e.message;
    }
  }

  async function handleUnhideSelected() {
    if (selectedIds.size === 0) return;
    try {
      await unhideStagedEvents([...selectedIds]);
      await refresh();
    } catch (e: any) {
      error = e.message;
    }
  }

  function toggleSelect(id: string) {
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selectedIds = next;
  }

  function selectAll() {
    selectedIds = new Set(stagedEvents.map(e => e.id));
  }

  function selectNone() {
    selectedIds = new Set();
  }


  function formatDate(start: string, end: string): string {
    if (!end || end === start) return start;
    return `${start} → ${end}`;
  }

  function lastSyncLabel(ts: string): string {
    if (!ts) return 'never';
    try {
      const d = new Date(ts);
      const now = new Date();
      const mins = Math.floor((now.getTime() - d.getTime()) / 60000);
      if (mins < 1) return 'just now';
      if (mins < 60) return `${mins}m ago`;
      const hrs = Math.floor(mins / 60);
      if (hrs < 24) return `${hrs}h ago`;
      return d.toLocaleDateString();
    } catch { return ts; }
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="overlay" onclick={onclose}>
  <div class="panel" onclick={(e) => e.stopPropagation()}>
    <div class="panel-header">
      <h2>Calendar Sources</h2>
      <button class="close-btn" onclick={onclose}>&times;</button>
    </div>

    {#if error}
      <p class="error">{error}</p>
    {/if}

    <!-- Add Source -->
    <section>
      <h3>Add Source</h3>
      <div class="add-form">
        <input type="text" placeholder="Name" bind:value={newSourceName} />
        <input type="url" placeholder="Calendar URL (.ics)" bind:value={newSourceURL}
          onkeydown={(e) => e.key === 'Enter' && handleAddSource()} />
        <button class="btn-primary" onclick={handleAddSource}
          disabled={addingSource || !newSourceName || !newSourceURL}>
          {addingSource ? 'Adding...' : 'Add'}
        </button>
      </div>
    </section>

    <!-- Sources List -->
    {#if sources.length > 0}
      <section>
        <h3>Sources</h3>
        <div class="source-list">
          {#each sources as src (src.id)}
            <div class="source-row">
              <div class="source-info">
                <span class="source-name">{src.name}</span>
                <span class="source-meta">
                  {src.newCount} new · {src.importedCount} imported · {src.hiddenCount} hidden · synced {lastSyncLabel(src.lastSyncAt)}
                </span>
              </div>
              <div class="source-actions">
                <button class="btn-small" onclick={() => handleSync(src.id)}>Sync</button>
                <button class="btn-small btn-danger" onclick={() => handleDeleteSource(src.id)}>Remove</button>
              </div>
            </div>
          {/each}
        </div>
      </section>
    {/if}

    <!-- Staged Events -->
    <section>
      <div class="staged-header">
        <h3>Staged Events</h3>
        <div class="staged-filters">
          <select bind:value={stateFilter} onchange={refreshStaged}>
            <option value="new">New</option>
            <option value="">All</option>
            <option value="imported">Imported</option>
            <option value="hidden">Hidden</option>
          </select>
          {#if sources.length > 1}
            <select bind:value={sourceFilter} onchange={refreshStaged}>
              <option value="">All sources</option>
              {#each sources as src}
                <option value={src.id}>{src.name}</option>
              {/each}
            </select>
          {/if}
        </div>
      </div>

      {#if loading}
        <p class="muted">Loading...</p>
      {:else if stagedEvents.length === 0}
        <p class="muted">No staged events{stateFilter ? ` (${stateFilter})` : ''}.</p>
      {:else}
        <!-- Bulk actions -->
        <div class="bulk-bar">
          <label class="select-all">
            <input type="checkbox"
              checked={selectedIds.size === stagedEvents.length && stagedEvents.length > 0}
              onchange={() => selectedIds.size === stagedEvents.length ? selectNone() : selectAll()} />
            {selectedIds.size}/{stagedEvents.length} selected
          </label>
          <div class="bulk-actions">
            {#if stateFilter === 'new' || stateFilter === ''}
              <button class="btn-small btn-primary-small" onclick={handleImportSelected}
                disabled={selectedIds.size === 0}>Import</button>
              <button class="btn-small" onclick={handleHideSelected}
                disabled={selectedIds.size === 0}>Hide</button>
            {/if}
            {#if stateFilter === 'hidden'}
              <button class="btn-small" onclick={handleUnhideSelected}
                disabled={selectedIds.size === 0}>Unhide</button>
            {/if}
          </div>
        </div>

        <!-- Event list -->
        <div class="event-list">
          {#each stagedEvents as event (event.id)}
            <label class="event-row" class:imported={event.state === 'imported'} class:hidden-event={event.state === 'hidden'}>
              <input type="checkbox"
                checked={selectedIds.has(event.id)}
                onchange={() => toggleSelect(event.id)}
                disabled={event.state === 'imported'} />
              <span class="type-dot" style="background: {ACTIVITY_COLORS[event.type as ActivityType] ?? '#999'}"></span>
              <span class="event-dates">{formatDate(event.startDate, event.endDate)}</span>
              <span class="event-title">{event.title}</span>
              {#if event.location}
                <span class="event-location">{event.location}</span>
              {/if}
              {#if event.state === 'imported'}
                <span class="state-badge imported-badge">imported</span>
              {:else if event.state === 'hidden'}
                <span class="state-badge hidden-badge">hidden</span>
              {/if}
            </label>
          {/each}
        </div>
      {/if}
    </section>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    z-index: 100;
    background: rgba(0, 0, 0, 0.3);
    display: flex;
    justify-content: flex-end;
  }

  .panel {
    width: 550px;
    max-width: 95vw;
    background: white;
    height: 100%;
    overflow-y: auto;
    padding: 1.25rem;
    box-shadow: -4px 0 20px rgba(0, 0, 0, 0.15);
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1rem;
  }

  .panel-header h2 { margin: 0; font-size: 1.2rem; }

  .close-btn {
    background: none; border: none; font-size: 1.5rem;
    cursor: pointer; color: #888; padding: 0 0.25rem; line-height: 1;
  }
  .close-btn:hover { color: #333; }

  section { margin-bottom: 1.25rem; }
  h3 { font-size: 0.9rem; margin: 0 0 0.5rem; color: #333; }

  .add-form {
    display: flex; flex-direction: column; gap: 0.4rem;
  }

  .add-form input {
    padding: 0.4rem 0.6rem; border: 1px solid #ddd; border-radius: 6px;
    font-size: 0.85rem; font-family: inherit;
  }
  .add-form input:focus { outline: none; border-color: #333; }

  .btn-primary {
    padding: 0.4rem 0.75rem; border: none; border-radius: 6px;
    background: #333; color: white; font-size: 0.8rem; cursor: pointer;
    align-self: flex-start;
  }
  .btn-primary:hover { background: #555; }
  .btn-primary:disabled { opacity: 0.4; cursor: default; }

  .source-list {
    display: flex; flex-direction: column; gap: 1px;
    background: #eee; border-radius: 6px; overflow: hidden;
  }

  .source-row {
    display: flex; align-items: center; justify-content: space-between;
    padding: 0.5rem 0.65rem; background: white; gap: 0.5rem;
  }

  .source-info { flex: 1; min-width: 0; }
  .source-name { font-size: 0.85rem; font-weight: 600; display: block; }
  .source-meta { font-size: 0.7rem; color: #888; }

  .source-actions { display: flex; gap: 0.3rem; flex-shrink: 0; }

  .btn-small {
    padding: 0.2rem 0.5rem; border: 1px solid #ddd; border-radius: 4px;
    background: white; font-size: 0.7rem; cursor: pointer; color: #555;
  }
  .btn-small:hover { border-color: #999; color: #333; }

  .btn-danger { color: #dc2626; border-color: #fecaca; }
  .btn-danger:hover { color: #b91c1c; border-color: #dc2626; background: #fef2f2; }

  .btn-primary-small {
    background: #333; color: white; border-color: #333;
  }
  .btn-primary-small:hover { background: #555; border-color: #555; }
  .btn-primary-small:disabled { opacity: 0.4; }

  .staged-header {
    display: flex; align-items: center; justify-content: space-between;
    margin-bottom: 0.5rem;
  }
  .staged-header h3 { margin: 0; }

  .staged-filters { display: flex; gap: 0.3rem; }
  .staged-filters select {
    padding: 0.2rem 0.4rem; border: 1px solid #ddd; border-radius: 4px;
    font-size: 0.75rem; background: white;
  }

  .bulk-bar {
    display: flex; align-items: center; justify-content: space-between;
    padding: 0.4rem 0.5rem; background: #f8f9fa; border-radius: 6px 6px 0 0;
    border: 1px solid #eee; border-bottom: none;
  }

  .select-all {
    display: flex; align-items: center; gap: 0.3rem;
    font-size: 0.75rem; color: #666; cursor: pointer;
  }
  .select-all input { margin: 0; }

  .bulk-actions { display: flex; gap: 0.3rem; }

  .event-list {
    display: flex; flex-direction: column; gap: 1px;
    background: #eee; border-radius: 0 0 6px 6px; overflow: hidden;
    border: 1px solid #eee; border-top: none;
  }

  .event-row {
    display: flex; align-items: center; gap: 0.5rem;
    padding: 0.4rem 0.5rem; background: white; font-size: 0.8rem;
    cursor: pointer;
  }
  .event-row:hover { background: #f8f9fa; }
  .event-row input { margin: 0; flex-shrink: 0; }

  .event-row.imported { opacity: 0.5; }
  .event-row.hidden-event { opacity: 0.4; text-decoration: line-through; }

  .type-dot {
    width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;
  }

  .event-dates {
    font-size: 0.7rem; color: #666; min-width: 80px; white-space: nowrap;
    flex-shrink: 0;
  }

  .event-title {
    flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
    font-weight: 500;
  }

  .event-location {
    font-size: 0.7rem; color: #888; white-space: nowrap;
    overflow: hidden; text-overflow: ellipsis; max-width: 120px;
  }

  .state-badge {
    font-size: 0.6rem; padding: 0.1rem 0.3rem; border-radius: 3px;
    white-space: nowrap; flex-shrink: 0;
  }
  .imported-badge { background: #dcfce7; color: #166534; }
  .hidden-badge { background: #f3f4f6; color: #9ca3af; }

  .error {
    color: #dc2626; font-size: 0.8rem; margin: 0 0 0.75rem;
    padding: 0.4rem 0.6rem; background: #fef2f2; border-radius: 6px;
  }

  .muted { color: #999; font-size: 0.8rem; padding: 0.5rem 0; }
</style>
