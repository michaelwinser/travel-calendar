<script lang="ts">
  import { onMount } from 'svelte';
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
  import QuickAdd from './components/QuickAdd.svelte';
  import AgendaView from './components/AgendaView.svelte';
  import MonthView from './components/MonthView.svelte';
  import DayView from './components/DayView.svelte';
  import YearView from './components/YearView.svelte';
  import ActivityForm from './components/ActivityForm.svelte';

  type View = 'month' | 'year' | 'day' | 'agenda';

  let auth = $state<AuthStatus>({ loggedIn: false });
  let activities = $state<Activity[]>([]);
  let currentView = $state<View>('month');
  let loading = $state(true);
  let error = $state('');

  // Edit modal state
  let editingActivity = $state<Activity | null>(null);

  // Click-to-add / drag-to-add state
  let prefillDates = $state<{ startDate: string; endDate: string } | null>(null);

  // View refs for Today button
  let monthView = $state<MonthView>();
  let dayView = $state<DayView>();
  let yearView = $state<YearView>();

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

  async function handleCreate(data: {
    title: string;
    type: ActivityType;
    startDate: string;
    endDate: string;
    location: string;
    notes: string;
    parseHistoryId?: string;
  }) {
    try {
      await createActivity({
        title: data.title,
        type: data.type,
        startDate: data.startDate,
        endDate: data.endDate !== data.startDate ? data.endDate : undefined,
        location: data.location || undefined,
        notes: data.notes || undefined,
        parseHistoryId: data.parseHistoryId,
      });
      prefillDates = null;
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to create activity';
    }
  }

  async function handleUpdate(data: {
    title: string;
    type: ActivityType;
    startDate: string;
    endDate: string;
    location: string;
    notes: string;
  }) {
    if (!editingActivity) return;
    try {
      await updateActivity(editingActivity.id, {
        title: data.title,
        type: data.type,
        startDate: data.startDate,
        endDate: data.endDate,
        location: data.location || undefined,
        notes: data.notes || undefined,
      });
      editingActivity = null;
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to update activity';
    }
  }

  async function handleDelete() {
    if (!editingActivity) return;
    try {
      await deleteActivity(editingActivity.id);
      editingActivity = null;
      await refreshActivities();
      error = '';
    } catch (e: any) {
      error = e.message || 'Failed to delete activity';
    }
  }

  function handleEdit(activity: Activity) {
    editingActivity = activity;
  }

  function handleSwitchToMonth(date: string) {
    // TODO: pass focusDate to month view (#42)
    currentView = 'month';
  }

  function handleDayClick(date: string) {
    prefillDates = { startDate: date, endDate: date };
  }

  function handleDragSelect(startDate: string, endDate: string) {
    prefillDates = { startDate, endDate };
  }

  const views: { id: View; label: string }[] = [
    { id: 'month', label: 'Month' },
    { id: 'year', label: 'Year' },
    { id: 'day', label: 'Day' },
    { id: 'agenda', label: 'Agenda' },
  ];
</script>

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
    <QuickAdd oncreate={handleCreate} />

    <!-- Prefill form (click-to-add / drag-to-add) -->
    {#if prefillDates}
      <ActivityForm
        mode="create"
        startDate={prefillDates.startDate}
        endDate={prefillDates.endDate}
        onsubmit={handleCreate}
        oncancel={() => prefillDates = null}
      />
    {/if}

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
        <button class="today-btn" onclick={() => {
          monthView?.scrollToToday();
          dayView?.scrollToToday();
          yearView?.scrollToToday();
        }}>Today</button>
      {/if}
    </nav>

    {#if error}
      <p class="error">{error}</p>
    {/if}

    {#if currentView === 'month'}
      <MonthView
        bind:this={monthView}
        {activities}
        onedit={handleEdit}
        ondayclick={handleDayClick}
        ondragselect={handleDragSelect}
      />
    {:else if currentView === 'year'}
      <YearView
        bind:this={yearView}
        {activities}
        onswitchtomonth={handleSwitchToMonth}
      />
    {:else if currentView === 'day'}
      <DayView
        bind:this={dayView}
        {activities}
        onedit={handleEdit}
        ondayclick={handleDayClick}
      />
    {:else if currentView === 'agenda'}
      <AgendaView {activities} onedit={handleEdit} />
    {:else}
      <p class="muted">
        {currentView.charAt(0).toUpperCase() + currentView.slice(1)} view coming soon.
      </p>
    {/if}
  {/if}

  <!-- Edit modal -->
  {#if editingActivity}
    <ActivityForm
      mode="edit"
      title={editingActivity.title}
      type={editingActivity.type}
      startDate={editingActivity.startDate}
      endDate={editingActivity.endDate}
      location={editingActivity.location ?? ''}
      notes={editingActivity.notes ?? ''}
      onsubmit={handleUpdate}
      oncancel={() => editingActivity = null}
      ondelete={handleDelete}
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
