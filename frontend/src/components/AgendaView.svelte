<script lang="ts">
  import { ACTIVITY_COLORS, type Activity } from '../lib/api';

  interface Props {
    activities: Activity[];
    onedit: (activity: Activity) => void;
  }

  let { activities, onedit }: Props = $props();

  function formatDates(a: Activity): string {
    if (a.startDate === a.endDate) return a.startDate;
    return `${a.startDate} \u2192 ${a.endDate}`;
  }
</script>

{#if activities.length === 0}
  <p class="empty">No activities yet. Use quick add above to create one.</p>
{:else}
  <div class="agenda">
    {#each activities as activity (activity.id)}
      <button
        class="activity-row"
        onclick={() => onedit(activity)}
      >
        <span class="type-dot" style="background: {ACTIVITY_COLORS[activity.type]}"></span>
        <span class="dates">{formatDates(activity)}</span>
        <span class="title">{activity.title}</span>
        {#if activity.location}
          <span class="location">{activity.location}</span>
        {/if}
        <span class="type-label">{activity.type}</span>
      </button>
    {/each}
  </div>
{/if}

<style>
  .empty {
    color: #999;
    text-align: center;
    padding: 3rem 0;
  }

  .agenda {
    display: flex;
    flex-direction: column;
    gap: 1px;
    background: #eee;
    border-radius: 8px;
    overflow: hidden;
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
