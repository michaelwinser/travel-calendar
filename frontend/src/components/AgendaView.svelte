<script lang="ts">
  import { ACTIVITY_COLORS, type Activity, type TripSummary, type OverlayCalendar } from '../lib/api';

  interface Props {
    activities: Activity[];
    trips: TripSummary[];
    overlayActivities?: Activity[];
    overlayCalendars?: OverlayCalendar[];
    onedit: (activity: Activity) => void;
    onedittrip?: (tripId: string) => void;
  }

  let { activities, trips, overlayActivities, overlayCalendars, onedit, onedittrip }: Props = $props();

  // Group overlay activities by owner email
  let overlayGroups = $derived.by(() => {
    if (!overlayActivities?.length || !overlayCalendars?.length) return [];
    const colorMap = new Map<string, string>();
    for (const c of overlayCalendars) if (c.visible) colorMap.set(c.email, c.color);
    const byOwner = new Map<string, Activity[]>();
    for (const a of overlayActivities) {
      const list = byOwner.get(a.userId);
      if (list) list.push(a); else byOwner.set(a.userId, [a]);
    }
    return [...byOwner.entries()].map(([email, acts]) => ({
      email,
      color: colorMap.get(email) ?? '#999',
      activities: acts.sort((a, b) => a.startDate < b.startDate ? -1 : 1),
    }));
  });

  interface AgendaGroup {
    tripId: string | null;
    tripName: string | null;
    tripColor: string | null;
    activities: Activity[];
    startDate: string;
    endDate: string;
    locations: string[];
  }

  let groups = $derived.by(() => {
    const tripMap = new Map<string, Activity[]>();
    const standalone: Activity[] = [];

    for (const a of activities) {
      if (a.tripId) {
        const list = tripMap.get(a.tripId);
        if (list) list.push(a);
        else tripMap.set(a.tripId, [a]);
      } else {
        standalone.push(a);
      }
    }

    const result: AgendaGroup[] = [];

    for (const [tripId, acts] of tripMap) {
      const trip = trips.find(t => t.id === tripId);
      result.push({
        tripId,
        tripName: trip?.name ?? 'Unknown trip',
        tripColor: trip?.color ?? '#999',
        activities: acts,
        startDate: acts.reduce((min, a) => a.startDate < min ? a.startDate : min, acts[0].startDate),
        endDate: acts.reduce((max, a) => a.endDate > max ? a.endDate : max, acts[0].endDate),
        locations: [...new Set(acts.map(a => a.location).filter(Boolean))] as string[],
      });
    }

    for (const a of standalone) {
      result.push({
        tripId: null,
        tripName: null,
        tripColor: null,
        activities: [a],
        startDate: a.startDate,
        endDate: a.endDate,
        locations: a.location ? [a.location] : [],
      });
    }

    result.sort((a, b) => a.startDate < b.startDate ? -1 : 1);
    return result;
  });

  function formatDates(a: Activity): string {
    if (a.startDate === a.endDate) return a.startDate;
    return `${a.startDate} \u2192 ${a.endDate}`;
  }

  function formatDateRange(start: string, end: string): string {
    if (start === end) return start;
    return `${start} \u2192 ${end}`;
  }
</script>

{#if activities.length === 0}
  <p class="empty">No activities yet. Press <kbd>n</kbd> to create one.</p>
{:else}
  <div class="agenda">
    {#each groups as group}
      {#if group.tripId}
        <!-- Trip group -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div class="trip-header" style="border-left: 4px solid {group.tripColor};">
          <span class="trip-name">{group.tripName}</span>
          <span class="trip-dates">{formatDateRange(group.startDate, group.endDate)}</span>
          {#if group.locations.length > 0}
            <span class="trip-locations">{group.locations.join(', ')}</span>
          {/if}
          {#if onedittrip && group.tripId}
            <button class="trip-edit-btn" onclick={() => onedittrip!(group.tripId!)}>Edit</button>
          {/if}
        </div>
        {#each group.activities as activity (activity.id)}
          <button class="activity-row indented" onclick={() => onedit(activity)}>
            <span class="type-dot" style="background: {ACTIVITY_COLORS[activity.type]}"></span>
            <span class="dates">{formatDates(activity)}</span>
            <span class="title">{activity.title}</span>
            {#if activity.location}
              <span class="location">{activity.location}</span>
            {/if}
            <span class="type-label">{activity.type}</span>
          </button>
        {/each}
      {:else}
        <!-- Standalone activity -->
        {#each group.activities as activity (activity.id)}
          <button class="activity-row" onclick={() => onedit(activity)}>
            <span class="type-dot" style="background: {ACTIVITY_COLORS[activity.type]}"></span>
            <span class="dates">{formatDates(activity)}</span>
            <span class="title">{activity.title}</span>
            {#if activity.location}
              <span class="location">{activity.location}</span>
            {/if}
            <span class="type-label">{activity.type}</span>
          </button>
        {/each}
      {/if}
    {/each}
  </div>
{/if}

{#if overlayGroups.length > 0}
  {#each overlayGroups as group}
    <div class="overlay-section">
      <div class="overlay-header" style="border-left: 4px solid {group.color};">
        <span class="overlay-email">{group.email}</span>
      </div>
      {#each group.activities as activity}
        <div class="overlay-row">
          <span class="type-dot" style="background: {group.color}"></span>
          <span class="dates">{formatDates(activity)}</span>
          {#if activity.location}
            <span class="location">{activity.location}</span>
          {/if}
          <span class="type-label">{activity.type}</span>
        </div>
      {/each}
    </div>
  {/each}
{/if}

<style>
  .empty {
    color: #999;
    text-align: center;
    padding: 3rem 0;
  }

  .empty kbd {
    background: #f3f4f6;
    border: 1px solid #ddd;
    border-radius: 4px;
    padding: 0.1rem 0.3rem;
    font-family: monospace;
    font-size: 0.85rem;
  }

  .agenda {
    display: flex;
    flex-direction: column;
    gap: 1px;
    background: #eee;
    border-radius: 8px;
    overflow: hidden;
  }

  .trip-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.5rem 0.85rem;
    background: #f8f9fa;
    border-top: 2px solid #e5e7eb;
  }

  .trip-name {
    font-weight: 700;
    font-size: 0.85rem;
    color: #333;
  }

  .trip-dates {
    font-size: 0.75rem;
    color: #888;
  }

  .trip-locations {
    font-size: 0.75rem;
    color: #aaa;
  }

  .trip-edit-btn {
    margin-left: auto;
    padding: 0.15rem 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    background: white;
    font-size: 0.7rem;
    color: #888;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .trip-header:hover .trip-edit-btn {
    opacity: 1;
  }

  .trip-edit-btn:hover {
    color: #333;
    border-color: #999;
  }

  .activity-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.65rem 0.85rem;
    background: white;
    border: none;
    text-align: left;
    font-family: inherit;
    font-size: 0.9rem;
    cursor: pointer;
    width: 100%;
  }

  .activity-row.indented {
    padding-left: 1.5rem;
  }

  .activity-row:hover {
    background: #f8f9fa;
  }

  .type-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .dates {
    color: #666;
    font-size: 0.8rem;
    min-width: 120px;
    white-space: nowrap;
  }

  .title {
    flex: 1;
    font-weight: 500;
  }

  .location {
    color: #888;
    font-size: 0.85rem;
  }

  .type-label {
    font-size: 0.75rem;
    color: #aaa;
    text-transform: capitalize;
  }

  .overlay-section {
    margin-top: 1rem;
    opacity: 0.75;
  }

  .overlay-header {
    display: flex;
    align-items: center;
    padding: 0.4rem 0.85rem;
    background: #f8f9fa;
    border-radius: 6px 6px 0 0;
  }

  .overlay-email {
    font-size: 0.8rem;
    font-weight: 600;
    color: #666;
  }

  .overlay-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.45rem 0.85rem 0.45rem 1.5rem;
    background: white;
    border: 1px solid #eee;
    border-top: none;
    font-size: 0.85rem;
  }

  .overlay-row:last-child {
    border-radius: 0 0 6px 6px;
  }
</style>
