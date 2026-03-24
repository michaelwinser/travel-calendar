<script lang="ts">
  import { ACTIVITY_COLORS, type Activity } from '../lib/api';

  interface Props {
    activities: Activity[];
    onedit: (activity: Activity) => void;
  }

  let { activities, onedit }: Props = $props();

  // Group activities: trip activities grouped under headers, standalone at top level
  interface AgendaGroup {
    tripName: string | null;
    activities: Activity[];
    startDate: string;
    endDate: string;
    locations: string[];
  }

  let groups = $derived.by(() => {
    const tripMap = new Map<string, Activity[]>();
    const standalone: Activity[] = [];

    for (const a of activities) {
      if (a.tripName) {
        const list = tripMap.get(a.tripName);
        if (list) list.push(a);
        else tripMap.set(a.tripName, [a]);
      } else {
        standalone.push(a);
      }
    }

    const result: AgendaGroup[] = [];

    // Trips first, sorted by earliest start date
    const trips = Array.from(tripMap.entries())
      .map(([name, acts]) => ({
        tripName: name,
        activities: acts,
        startDate: acts.reduce((min, a) => a.startDate < min ? a.startDate : min, acts[0].startDate),
        endDate: acts.reduce((max, a) => a.endDate > max ? a.endDate : max, acts[0].endDate),
        locations: [...new Set(acts.map(a => a.location).filter(Boolean))],
      }))
      .sort((a, b) => a.startDate < b.startDate ? -1 : 1);

    result.push(...trips);

    // Standalone activities as individual groups
    for (const a of standalone) {
      result.push({
        tripName: null,
        activities: [a],
        startDate: a.startDate,
        endDate: a.endDate,
        locations: a.location ? [a.location] : [],
      });
    }

    // Sort all groups by start date
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
      {#if group.tripName}
        <!-- Trip group -->
        <div class="trip-header">
          <span class="trip-name">{group.tripName}</span>
          <span class="trip-dates">{formatDateRange(group.startDate, group.endDate)}</span>
          {#if group.locations.length > 0}
            <span class="trip-locations">{group.locations.join(', ')}</span>
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
    margin-left: auto;
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
</style>
