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
    type Activity,
    type ActivityType,
    type AuthStatus,
  } from './lib/api';
  import AgendaView from './components/AgendaView.svelte';
  import MonthView from './components/MonthView.svelte';
  import DayView from './components/DayView.svelte';
  import YearView from './components/YearView.svelte';
  import ActivityModal from './components/ActivityModal.svelte';

  type View = 'month' | 'year' | 'day' | 'agenda';

  let auth = $state<AuthStatus>({ loggedIn: false });
  let activities = $state<Activity[]>([]);
  let currentView = $state<View>('month');
  let loading = $state(true);
  let error = $state('');

  // Unified modal state
  let modalOpen = $state(false);
  let modalMode = $state<'create' | 'edit'>('create');
  let modalFocusText = $state(false);
  let modalActivity = $state<Activity | null>(null); // for edit
  let modalPrefill = $state<{ startDate?: string; endDate?: string }>({});

  // View refs and focus date
  let monthView = $state<MonthView>();
  let dayView = $state<DayView>();
  let yearView = $state<YearView>();
  let focusDate = $state(today());

  onMount(async () => {
    try {
      auth = await getAuthStatus();
      if (auth.loggedIn) {
        await refreshActivities();
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
  }

  // --- CRUD ---

  async function handleModalSubmit(data: {
    title: string;
    type: ActivityType;
    startDate: string;
    endDate: string;
    location: string;
    notes: string;
    parseHistoryId?: string;
  }) {
    try {
      if (modalMode === 'create') {
        await createActivity({
          title: data.title,
          type: data.type,
          startDate: data.startDate,
          endDate: data.endDate !== data.startDate ? data.endDate : undefined,
          location: data.location || undefined,
          notes: data.notes || undefined,
          parseHistoryId: data.parseHistoryId,
        });
      } else if (modalActivity) {
        await updateActivity(modalActivity.id, {
          title: data.title,
          type: data.type,
          startDate: data.startDate,
          endDate: data.endDate,
          location: data.location || undefined,
          notes: data.notes || undefined,
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

  function handleSwitchToMonth(date: string) {
    // TODO: pass focusDate to month view (#42)
    currentView = 'month';
  }

  const views: { id: View; label: string }[] = [
    { id: 'month', label: 'Month' },
    { id: 'year', label: 'Year' },
    { id: 'day', label: 'Day' },
    { id: 'agenda', label: 'Agenda' },
  ];
</script>

<svelte:window onkeydown={handleGlobalKeydown} />

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
            <kbd>n</kbd><span>New activity</span>
            <kbd>?</kbd><span>Show this help</span>
            <kbd>Esc</kbd><span>Close modal / dismiss</span>
          </div>
        </div>
      </div>
    {/if}

    {#if error}
      <p class="error">{error}</p>
    {/if}

    {#if currentView === 'month'}
      <MonthView
        bind:this={monthView}
        {activities}
        initialDate={focusDate}
        onedit={handleEdit}
        ondayclick={handleDayClick}
        ondragselect={handleDragSelect}
      />
    {:else if currentView === 'year'}
      <YearView
        bind:this={yearView}
        {activities}
        initialDate={focusDate}
        onedit={handleEdit}
        ondayclick={handleDayClick}
        ondragselect={handleDragSelect}
        onswitchtomonth={handleSwitchToMonth}
      />
    {:else if currentView === 'day'}
      <DayView
        bind:this={dayView}
        {activities}
        initialDate={focusDate}
        onedit={handleEdit}
        ondayclick={handleDayClick}
        ondragselect={handleDragSelect}
      />
    {:else if currentView === 'agenda'}
      <AgendaView {activities} onedit={handleEdit} />
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
      notes={modalMode === 'edit' ? (modalActivity?.notes ?? '') : undefined}
      focusText={modalFocusText}
      onsubmit={handleModalSubmit}
      oncancel={closeModal}
      ondelete={modalMode === 'edit' ? handleDelete : undefined}
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
