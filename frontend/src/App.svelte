<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { today } from './lib/date-utils';
  import {
    getAuthStatus,
    getLoginURL,
    logout,
    listActivities,
    createActivity,
    updateActivity,
    deleteActivity,
    bulkDeleteActivities,
    listTrips,
    createTrip,
    updateTrip,
    deleteTrip,
    type Activity,
    type ActivityType,
    type AuthStatus,
    type TripSummary,
  } from './lib/api';
  import AgendaView from './components/AgendaView.svelte';
  import TripEditModal from './components/TripEditModal.svelte';
  import MonthView from './components/MonthView.svelte';
  import DayView from './components/DayView.svelte';
  import YearView from './components/YearView.svelte';
  import ActivityModal from './components/ActivityModal.svelte';
  import SharePanel from './components/SharePanel.svelte';
  import SharedCalendarView from './components/SharedCalendarView.svelte';
  import OverlaySidebar from './components/OverlaySidebar.svelte';
  import SourcesPanel from './components/SourcesPanel.svelte';
  import {
    listSharedWithMe,
    fetchSharedWithMeActivities,
    type OverlayCalendar,
    type SharedWithMeEntry,
    type ActivityType as AType,
  } from './lib/api';

  type View = 'month' | 'year' | 'day' | 'agenda';

  // Check if we're viewing someone else's shared calendar: /view/{email}
  const viewMatch = window.location.pathname.match(/^\/view\/([^/]+)/);
  const viewEmail = viewMatch ? decodeURIComponent(viewMatch[1]) : null;

  let auth = $state<AuthStatus>({ loggedIn: false });
  let activities = $state<Activity[]>([]);
  let tripsCache = $state<TripSummary[]>([]);
  let loading = $state(true);
  let error = $state('');

  // Parse initial state from URL
  const { view: initialView, date: initialDate } = parseURL(window.location.pathname);
  let currentView = $state<View>(initialView);
  let focusDate = $state(initialDate);

  // Unified modal state
  let modalOpen = $state(false);
  let modalMode = $state<'create' | 'edit'>('create');
  let modalFocusText = $state(false);
  let modalActivity = $state<Activity | null>(null);
  let modalPrefill = $state<{ startDate?: string; endDate?: string }>({});
  let ghostDates = $state<{ startDate: string; endDate: string; type: ActivityType } | null>(null);

  // Trip edit state
  let editingTrip = $state<TripSummary | null>(null);

  // Share panel state
  let showSharePanel = $state(false);

  // Sources panel state
  let showSourcesPanel = $state(false);

  // Search state
  let searchOpen = $state(false);
  let searchQuery = $state('');
  let viewBeforeSearch = $state<View>('month');
  let searchInput: HTMLInputElement;

  let searchResults = $derived.by(() => {
    if (!searchQuery.trim()) return activities;
    const q = searchQuery.toLowerCase();
    return activities.filter(a =>
      a.title.toLowerCase().includes(q) ||
      (a.location && a.location.toLowerCase().includes(q)) ||
      (a.notes && a.notes.toLowerCase().includes(q)) ||
      a.type.toLowerCase().includes(q)
    );
  });

  function openSearch() {
    if (!searchOpen) {
      viewBeforeSearch = currentView;
    }
    searchOpen = true;
    currentView = 'agenda';
    // Focus after DOM update
    setTimeout(() => searchInput?.focus(), 0);
  }

  function closeSearch() {
    searchOpen = false;
    searchQuery = '';
    currentView = viewBeforeSearch;
  }

  // Overlay state
  const OVERLAY_COLORS = ['#e07b53', '#5cbcb6', '#c75ca2', '#d4a843', '#8b6cc1', '#c95454', '#5a8f5a'];
  let overlayCalendars = $state<OverlayCalendar[]>([]);

  let visibleOverlayActivities = $derived(
    overlayCalendars
      .filter(c => c.visible)
      .flatMap(c => c.activities)
  );

  // View refs
  let monthView = $state<MonthView>();
  let dayView = $state<DayView>();
  let yearView = $state<YearView>();

  // --- URL routing ---

  function parseURL(path: string): { view: View; date: string } {
    const parts = path.split('/').filter(Boolean);
    const viewMap: Record<string, View> = { month: 'month', year: 'year', day: 'day', agenda: 'agenda' };
    const view = viewMap[parts[0]] ?? 'month';
    const date = parts[1] ?? today();
    // For month URLs like /month/2026-10, expand to first of month
    const normalizedDate = date.length === 7 ? date + '-01' : date;
    return { view, date: normalizedDate };
  }

  function buildURL(view: View, date: string): string {
    if (view === 'agenda') return '/agenda';
    // Month: /month/2026-10, Year: /year/2026-10, Day: /day/2026-10-05
    if (view === 'month' || view === 'year') return `/${view}/${date.slice(0, 7)}`;
    return `/${view}/${date}`;
  }

  function pushURL() {
    const url = buildURL(currentView, focusDate);
    if (window.location.pathname !== url) {
      history.pushState({ view: currentView, date: focusDate }, '', url);
    }
  }

  function replaceURL() {
    const url = buildURL(currentView, focusDate);
    if (window.location.pathname !== url) {
      history.replaceState({ view: currentView, date: focusDate }, '', url);
    }
  }

  function handlePopState(e: PopStateEvent) {
    if (e.state?.view && e.state?.date) {
      currentView = e.state.view;
      focusDate = e.state.date;
    } else {
      const { view, date } = parseURL(window.location.pathname);
      currentView = view;
      focusDate = date;
    }
  }

  // Push URL on view switch (use $effect to track changes)
  let prevView = currentView;
  let prevFocusDate = focusDate;
  $effect(() => {
    if (currentView !== prevView) {
      prevView = currentView;
      prevFocusDate = focusDate;
      pushURL();
    } else if (focusDate !== prevFocusDate) {
      prevFocusDate = focusDate;
      replaceURL();
    }
  });

  onMount(async () => {
    // Set initial history state
    history.replaceState({ view: currentView, date: focusDate }, '', buildURL(currentView, focusDate));

    try {
      auth = await getAuthStatus();
      if (auth.loggedIn) {
        await refreshActivities();
        await loadOverlays();
      }
    } catch {
      error = 'Failed to connect to server';
    }
    loading = false;
  });

  // Help and go-to-date state
  let showShortcutHelp = $state(false);
  let showGoToDate = $state(false);
  let goToDateValue = $state('');
  let goToDateInput: HTMLInputElement;

  // Global keyboard shortcuts
  function handleGlobalKeydown(e: KeyboardEvent) {
    // ESC closes popovers
    if (e.key === 'Escape') {
      if (searchOpen) { closeSearch(); return; }
      if (showGoToDate) { showGoToDate = false; return; }
      if (showShortcutHelp) { showShortcutHelp = false; return; }
    }

    // Don't trigger shortcuts if typing in an input/textarea or modal/popover is open
    if (modalOpen || showGoToDate || showShortcutHelp) return;
    const tag = (e.target as HTMLElement)?.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

    switch (e.key) {
      case 'n':
        e.preventDefault();
        openQuickAdd();
        break;
      case 'm':
        e.preventDefault();
        currentView = 'month';
        break;
      case 'y':
        e.preventDefault();
        currentView = 'year';
        break;
      case 'd':
        e.preventDefault();
        currentView = 'day';
        break;
      case 'a':
        e.preventDefault();
        currentView = 'agenda';
        break;
      case 't':
        e.preventDefault();
        scrollCurrentViewToToday();
        break;
      case 'g':
        e.preventDefault();
        openGoToDate();
        break;
      case 'j':
        e.preventDefault();
        scrollCurrentView('pageDown');
        break;
      case 'k':
        e.preventDefault();
        scrollCurrentView('pageUp');
        break;
      case 'J':
        e.preventDefault();
        scrollCurrentView('nextActivity');
        break;
      case 'K':
        e.preventDefault();
        scrollCurrentView('prevActivity');
        break;
      case '/':
        e.preventDefault();
        openSearch();
        break;
      case '?':
        e.preventDefault();
        showShortcutHelp = !showShortcutHelp;
        break;
    }
  }

  function scrollCurrentViewToToday() {
    focusDate = today();
    monthView?.scrollToToday();
    dayView?.scrollToToday();
    yearView?.scrollToToday();
  }

  function scrollCurrentView(action: 'pageDown' | 'pageUp' | 'nextActivity' | 'prevActivity') {
    if (currentView === 'month') monthView?.scrollAction(action);
    else if (currentView === 'day') dayView?.scrollAction(action);
    else if (currentView === 'year') yearView?.scrollAction(action);
  }

  function openGoToDate() {
    goToDateValue = '';
    showGoToDate = true;
    tick().then(() => goToDateInput?.focus());
  }

  function handleGoToDateSubmit() {
    if (!goToDateValue) return;
    focusDate = goToDateValue;
    showGoToDate = false;
    monthView?.scrollToDate(goToDateValue);
    dayView?.scrollToDate(goToDateValue);
    yearView?.scrollToDate(goToDateValue);
  }

  async function refreshActivities() {
    activities = await listActivities();
    tripsCache = await listTrips();
  }

  async function loadOverlays() {
    try {
      const entries = await listSharedWithMe();
      // Preserve existing visibility state
      const prevState = new Map(overlayCalendars.map(c => [c.email, c.visible]));

      overlayCalendars = await Promise.all(entries.map(async (entry, i) => {
        const data = await fetchSharedWithMeActivities(entry.ownerEmail);
        // Map SharedActivity to Activity shape for views
        const acts: Activity[] = data.activities.map((a, j) => ({
          id: `overlay-${entry.ownerEmail}-${j}`,
          userId: entry.ownerEmail,
          title: a.title || a.location || String(a.type),
          type: (a.type as AType),
          startDate: a.startDate,
          endDate: a.endDate,
          location: a.location,
          tripId: a.tripName ? `overlay-trip-${a.tripName}` : undefined,
          source: 'manual' as const,
          createdAt: '',
        }));
        return {
          email: entry.ownerEmail,
          color: OVERLAY_COLORS[i % OVERLAY_COLORS.length],
          visible: prevState.get(entry.ownerEmail) ?? false,
          activities: acts,
        };
      }));
    } catch {
      // Overlay loading failure is non-fatal
    }
  }

  function toggleOverlay(email: string) {
    overlayCalendars = overlayCalendars.map(c =>
      c.email === email ? { ...c, visible: !c.visible } : c
    );
  }

  async function handleLogin() {
    try {
      const url = await getLoginURL();
      window.location.href = url;
    } catch {
      error = 'Login not available';
    }
  }

  async function handleLogout() {
    await logout();
    auth = { loggedIn: false };
    activities = [];
  }

  // --- Modal entry points ---

  function openQuickAdd() {
    modalMode = 'create';
    modalActivity = null;
    modalPrefill = {};
    modalFocusText = true;
    modalOpen = true;
  }

  function handleDayClick(date: string) {
    focusDate = date;
    modalMode = 'create';
    modalActivity = null;
    modalPrefill = { startDate: date, endDate: date };
    modalFocusText = false;
    modalOpen = true;
  }

  function handleDragSelect(startDate: string, endDate: string) {
    focusDate = startDate;
    modalMode = 'create';
    modalActivity = null;
    modalPrefill = { startDate, endDate };
    modalFocusText = false;
    modalOpen = true;
  }

  function handleEdit(activity: Activity) {
    modalMode = 'edit';
    modalActivity = activity;
    modalPrefill = {};
    modalFocusText = false;
    modalOpen = true;
  }

  function closeModal() {
    modalOpen = false;
    modalActivity = null;
    modalPrefill = {};
    ghostDates = null;
  }

  function handleModalChange(data: { title: string; type: ActivityType; startDate: string; endDate: string }) {
    ghostDates = { startDate: data.startDate, endDate: data.endDate, type: data.type };
  }

  // --- CRUD ---

  async function resolveTripId(tripName: string, existingTripId?: string): Promise<string | undefined> {
    if (!tripName) return undefined;
    // If we already have a trip ID and the name matches, keep it
    if (existingTripId) {
      const existing = tripsCache.find(t => t.id === existingTripId);
      if (existing && existing.name === tripName) return existingTripId;
    }
    // Look up by name
    const found = tripsCache.find(t => t.name === tripName);
    if (found) return found.id;
    // Create new trip
    const newTrip = await createTrip({ name: tripName });
    return newTrip.id;
  }

  async function handleModalSubmit(data: {
    title: string;
    type: ActivityType;
    startDate: string;
    endDate: string;
    location: string;
    placeId: string;
    notes: string;
    tripId: string;
    tripName: string;
    parseHistoryId?: string;
  }) {
    try {
      const tripId = await resolveTripId(data.tripName, data.tripId || undefined);

      if (modalMode === 'create') {
        await createActivity({
          title: data.title,
          type: data.type,
          startDate: data.startDate,
          endDate: data.endDate !== data.startDate ? data.endDate : undefined,
          location: data.location || undefined,
          placeId: data.placeId || undefined,
          notes: data.notes || undefined,
          tripId: tripId,
          parseHistoryId: data.parseHistoryId,
        });
      } else if (modalActivity) {
        await updateActivity(modalActivity.id, {
          title: data.title,
          type: data.type,
          startDate: data.startDate,
          endDate: data.endDate,
          location: data.location || undefined,
          placeId: data.placeId || undefined,
          notes: data.notes || undefined,
          tripId: tripId ?? '',
        });
      }
      closeModal();
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to save activity';
    }
  }

  async function handleDelete() {
    if (!modalActivity) return;
    try {
      await deleteActivity(modalActivity.id);
      closeModal();
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to delete activity';
    }
  }

  async function handleBulkDelete(ids: string[]) {
    try {
      await bulkDeleteActivities(ids);
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to delete activities';
    }
  }

  async function handleResize(activityId: string, startDate: string, endDate: string) {
    try {
      await updateActivity(activityId, { startDate, endDate });
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to resize activity';
    }
  }

  async function handleMove(activityId: string, startDate: string, endDate: string) {
    try {
      await updateActivity(activityId, { startDate, endDate });
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to move activity';
    }
  }

  // --- Trip management ---

  function handleEditTrip(tripId: string) {
    const trip = tripsCache.find(t => t.id === tripId);
    if (trip) editingTrip = trip;
  }

  async function handleTripUpdate(data: { name: string; color: string; startDate?: string; endDate?: string }) {
    if (!editingTrip) return;
    try {
      await updateTrip(editingTrip.id, {
        name: data.name,
        color: data.color,
        startDate: data.startDate,
        endDate: data.endDate,
      });
      editingTrip = null;
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to update trip';
    }
  }

  // Unassigned activities that overlap the editing trip's date range
  let tripUnassigned = $derived.by(() => {
    if (!editingTrip) return [];
    return activities.filter(a =>
      !a.tripId &&
      a.startDate <= editingTrip!.endDate &&
      (a.endDate || a.startDate) >= editingTrip!.startDate
    );
  });

  async function handleTripAssign(activityId: string) {
    if (!editingTrip) return;
    try {
      await updateActivity(activityId, { tripId: editingTrip.id });
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to assign activity';
    }
  }

  async function handleAssignToTrip(activityId: string, tripId: string) {
    try {
      await updateActivity(activityId, { tripId });
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to assign activity';
    }
  }

  async function handleTripDelete() {
    if (!editingTrip) return;
    try {
      await deleteTrip(editingTrip.id);
      editingTrip = null;
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to delete trip';
    }
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

<svelte:window onkeydown={handleGlobalKeydown} onpopstate={handlePopState} />

<main class:wide={currentView === 'month' || currentView === 'year'}>
  <header>
    <h1>Travel Calendar</h1>
    {#if auth.loggedIn}
      <div class="user">
        {auth.email}
        <button class="link" onclick={handleLogout}>Sign out</button>
      </div>
    {/if}
  </header>

  {#if loading}
    <p class="muted">Loading...</p>
  {:else if !auth.loggedIn}
    <div class="login-card">
      <p>Sign in to manage your travel plans</p>
      <button onclick={handleLogin}>Sign in with Google</button>
    </div>
  {:else if viewEmail}
    <SharedCalendarView email={viewEmail} />
  {:else}
    <nav class="view-tabs">
      {#each views as v}
        <button
          class="tab"
          class:active={currentView === v.id}
          onclick={() => currentView = v.id}
        >{v.label}</button>
      {/each}
      <div class="tab-spacer"></div>
      {#if searchOpen}
        <div class="search-bar">
          <input
            type="text"
            class="search-input"
            placeholder="Search activities..."
            bind:value={searchQuery}
            bind:this={searchInput}
            onkeydown={(e) => e.key === 'Escape' && closeSearch()}
          />
          <button class="search-clear" onclick={closeSearch}>&times;</button>
        </div>
      {:else}
        <button class="search-btn" onclick={openSearch} title="Search (/)">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="6.5" cy="6.5" r="5"/>
            <line x1="10.5" y1="10.5" x2="15" y2="15"/>
          </svg>
        </button>
      {/if}
      <button class="share-btn" onclick={() => showSourcesPanel = true} title="Calendar sources">Import</button>
      <button class="share-btn" onclick={() => showSharePanel = true} title="Sharing settings">Share</button>
      <button class="add-btn" onclick={openQuickAdd} title="New activity (n)">+ Add</button>
      {#if currentView === 'month' || currentView === 'day' || currentView === 'year'}
        <button class="today-btn" onclick={scrollCurrentViewToToday}>Today</button>
      {/if}
    </nav>

    <!-- Go-to-date popover -->
    {#if showGoToDate}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <div class="goto-overlay" onclick={() => showGoToDate = false}>
        <div class="goto-popover" onclick={(e) => e.stopPropagation()}>
          <label>
            <span>Go to date</span>
            <input
              type="date"
              bind:value={goToDateValue}
              bind:this={goToDateInput}
              onkeydown={(e) => {
                if (e.key === 'Enter') { e.preventDefault(); handleGoToDateSubmit(); }
                if (e.key === 'Escape') { showGoToDate = false; }
              }}
            />
          </label>
          <button class="goto-btn" onclick={handleGoToDateSubmit} disabled={!goToDateValue}>Go</button>
        </div>
      </div>
    {/if}

    <!-- Keyboard shortcut help -->
    {#if showShortcutHelp}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <div class="help-overlay" onclick={() => showShortcutHelp = false}>
        <div class="help-popover" onclick={(e) => e.stopPropagation()}>
          <h3>Keyboard Shortcuts</h3>
          <div class="shortcut-grid">
            <kbd>m</kbd><span>Month view</span>
            <kbd>y</kbd><span>Year view</span>
            <kbd>d</kbd><span>Day view</span>
            <kbd>a</kbd><span>Agenda view</span>
            <kbd>t</kbd><span>Go to today</span>
            <kbd>g</kbd><span>Go to date</span>
            <kbd>/</kbd><span>Search</span>
            <kbd>n</kbd><span>New activity</span>
            <kbd>j</kbd><span>Page down</span>
            <kbd>k</kbd><span>Page up</span>
            <kbd>J</kbd><span>Next activity</span>
            <kbd>K</kbd><span>Previous activity</span>
            <kbd>?</kbd><span>Show this help</span>
            <kbd>Esc</kbd><span>Close modal / dismiss</span>
          </div>
        </div>
      </div>
    {/if}

    {#if error}
      <p class="error">{error}</p>
    {/if}

    <OverlaySidebar overlays={overlayCalendars} ontoggle={toggleOverlay} />

    {#if currentView === 'month'}
      <MonthView
        bind:this={monthView}
        {activities}
        trips={tripsCache}
        {ghostDates}
        overlayActivities={visibleOverlayActivities}
        {overlayCalendars}
        initialDate={focusDate}
        onedit={handleEdit}
        ondayclick={handleDayClick}
        ondragselect={handleDragSelect}
        onresize={handleResize}
        onmove={handleMove}
        onedittrip={handleEditTrip}
        onassigntotrip={handleAssignToTrip}
        onfocusdate={handleFocusDate}
      />
    {:else if currentView === 'year'}
      <YearView
        bind:this={yearView}
        {activities}
        trips={tripsCache}
        initialDate={focusDate}
        onedit={handleEdit}
        ondayclick={handleDayClick}
        ondragselect={handleDragSelect}
        onswitchtomonth={handleSwitchToMonth}
        onedittrip={handleEditTrip}
        onassigntotrip={handleAssignToTrip}
        onfocusdate={handleFocusDate}
      />
    {:else if currentView === 'day'}
      <DayView
        bind:this={dayView}
        {activities}
        trips={tripsCache}
        overlayActivities={visibleOverlayActivities}
        {overlayCalendars}
        initialDate={focusDate}
        onedit={handleEdit}
        ondayclick={handleDayClick}
        ondragselect={handleDragSelect}
        onfocusdate={handleFocusDate}
      />
    {:else if currentView === 'agenda'}
      <AgendaView activities={searchOpen ? searchResults : activities} trips={tripsCache} overlayActivities={searchOpen ? [] : visibleOverlayActivities} overlayCalendars={searchOpen ? [] : overlayCalendars} onedit={handleEdit} onedittrip={handleEditTrip} onbulkdelete={handleBulkDelete} />
    {/if}
  {/if}

  <!-- Unified activity modal -->
  {#if modalOpen}
    <ActivityModal
      mode={modalMode}
      title={modalMode === 'edit' ? modalActivity?.title : undefined}
      type={modalMode === 'edit' ? modalActivity?.type : undefined}
      startDate={modalMode === 'edit' ? modalActivity?.startDate : modalPrefill.startDate}
      endDate={modalMode === 'edit' ? modalActivity?.endDate : modalPrefill.endDate}
      location={modalMode === 'edit' ? (modalActivity?.location ?? '') : undefined}
      placeId={modalMode === 'edit' ? (modalActivity?.placeId ?? '') : undefined}
      notes={modalMode === 'edit' ? (modalActivity?.notes ?? '') : undefined}
      tripId={modalMode === 'edit' ? (modalActivity?.tripId ?? '') : undefined}
      tripName={modalMode === 'edit' ? (tripsCache.find(t => t.id === modalActivity?.tripId)?.name ?? '') : undefined}
      trips={tripsCache}
      focusText={modalFocusText}
      onsubmit={handleModalSubmit}
      oncancel={closeModal}
      ondelete={modalMode === 'edit' ? handleDelete : undefined}
      onchange={handleModalChange}
    />
  {/if}

  {#if showSourcesPanel}
    <SourcesPanel
      onclose={() => showSourcesPanel = false}
      onimported={refreshActivities}
    />
  {/if}

  {#if showSharePanel}
    <SharePanel onclose={() => showSharePanel = false} />
  {/if}

  {#if editingTrip}
    <TripEditModal
      name={editingTrip.name}
      color={editingTrip.color}
      startDate={editingTrip.startDate}
      endDate={editingTrip.endDate}
      unassignedActivities={tripUnassigned}
      onsubmit={handleTripUpdate}
      ondelete={handleTripDelete}
      oncancel={() => editingTrip = null}
      onassign={handleTripAssign}
    />
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

  .user {
    color: #666;
    font-size: 0.85rem;
  }

  .user .link {
    background: none;
    border: none;
    color: #0066cc;
    cursor: pointer;
    font-size: inherit;
    padding: 0;
    margin-left: 0.5rem;
  }

  .login-card {
    background: white;
    border-radius: 12px;
    padding: 2rem;
    text-align: center;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  }

  .login-card button {
    padding: 0.75rem 1.5rem;
    border: 1px solid #ddd;
    border-radius: 8px;
    background: white;
    font-size: 1rem;
    cursor: pointer;
  }

  .login-card button:hover { background: #f0f0f0; }

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

  .search-btn {
    background: none;
    border: none;
    cursor: pointer;
    color: #888;
    padding: 0.3rem;
    margin-bottom: 2px;
    display: flex;
    align-items: center;
  }

  .search-btn:hover { color: #333; }

  .search-bar {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    margin-bottom: 2px;
  }

  .search-input {
    padding: 0.3rem 0.6rem;
    border: 1px solid #333;
    border-radius: 6px;
    font-size: 0.8rem;
    font-family: inherit;
    width: 200px;
  }

  .search-input:focus {
    outline: none;
  }

  .search-clear {
    background: none;
    border: none;
    cursor: pointer;
    color: #888;
    font-size: 1.1rem;
    padding: 0;
    line-height: 1;
  }

  .search-clear:hover { color: #333; }

  .share-btn {
    padding: 0.3rem 0.75rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    background: white;
    font-size: 0.8rem;
    cursor: pointer;
    color: #555;
    margin-bottom: 2px;
    margin-right: 0.5rem;
  }

  .share-btn:hover {
    background: #f5f5f5;
    color: #333;
  }

  .add-btn {
    padding: 0.3rem 0.75rem;
    border: 1px solid #333;
    border-radius: 6px;
    background: #333;
    color: white;
    font-size: 0.8rem;
    cursor: pointer;
    margin-bottom: 2px;
    margin-right: 0.5rem;
  }

  .add-btn:hover {
    background: #555;
    border-color: #555;
  }

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

  .goto-overlay {
    position: fixed;
    inset: 0;
    z-index: 90;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 20vh;
  }

  .goto-popover {
    background: white;
    border-radius: 10px;
    padding: 1rem 1.25rem;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
    display: flex;
    align-items: flex-end;
    gap: 0.5rem;
  }

  .goto-popover label {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .goto-popover label span {
    font-size: 0.75rem;
    font-weight: 600;
    color: #555;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .goto-popover input {
    padding: 0.5rem 0.6rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    font-size: 0.95rem;
    font-family: inherit;
  }

  .goto-popover input:focus {
    outline: none;
    border-color: #333;
  }

  .goto-btn {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 6px;
    background: #333;
    color: white;
    font-size: 0.9rem;
    cursor: pointer;
  }

  .goto-btn:hover { background: #555; }
  .goto-btn:disabled { opacity: 0.4; cursor: default; }

  .help-overlay {
    position: fixed;
    inset: 0;
    z-index: 90;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 15vh;
    background: rgba(0, 0, 0, 0.2);
  }

  .help-popover {
    background: white;
    border-radius: 10px;
    padding: 1.25rem 1.5rem;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
    min-width: 240px;
  }

  .help-popover h3 {
    margin: 0 0 0.75rem;
    font-size: 1rem;
  }

  .shortcut-grid {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 0.35rem 0.75rem;
    align-items: center;
  }

  .shortcut-grid kbd {
    background: #f3f4f6;
    border: 1px solid #ddd;
    border-radius: 4px;
    padding: 0.15rem 0.4rem;
    font-family: monospace;
    font-size: 0.8rem;
    color: #555;
    text-align: center;
    min-width: 24px;
  }

  .shortcut-grid span {
    font-size: 0.85rem;
    color: #666;
  }

  .error {
    color: #dc2626;
    font-size: 0.85rem;
  }

  .muted {
    color: #999;
    text-align: center;
    padding: 2rem 0;
  }
</style>
