<script lang="ts">
  import { ACTIVITY_COLORS, type Activity } from '../lib/api';

  interface Props {
    tripId?: string | null;
    tripName: string;
    color: string;
    activities: Activity[];
    x: number;
    y: number;
    onedit: (activity: Activity) => void;
    onedittrip?: (tripId: string) => void;
    onclose: () => void;
  }

  let { tripId, tripName, color, activities, x, y, onedit, onedittrip, onclose }: Props = $props();

  function formatDates(a: Activity): string {
    if (a.startDate === a.endDate) return a.startDate;
    return `${a.startDate} \u2192 ${a.endDate}`;
  }

  // Sort activities by start date
  let sorted = $derived(
    [...activities].sort((a, b) => a.startDate < b.startDate ? -1 : 1)
  );
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="popup-overlay" onclick={onclose}>
  <div
    class="popup"
    style="left: {Math.min(x, window.innerWidth - 320)}px; top: {Math.min(y, window.innerHeight - 300)}px;"
    onclick={(e) => e.stopPropagation()}
  >
    <div class="popup-header" style="border-left: 4px solid {color};">
      <span class="popup-title">{tripName}</span>
      <span class="popup-count">{activities.length} {activities.length === 1 ? 'activity' : 'activities'}</span>
      {#if onedittrip && tripId}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <span class="popup-edit" onclick={() => { const id = tripId!; onclose(); onedittrip!(id); }}>Edit trip</span>
      {/if}
    </div>
    <div class="popup-list">
      {#each sorted as activity (activity.id)}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div class="popup-activity" onclick={() => { onclose(); onedit(activity); }}>
          <span class="popup-dot" style="background: {ACTIVITY_COLORS[activity.type]}"></span>
          <span class="popup-dates">{formatDates(activity)}</span>
          <span class="popup-name">{activity.title}</span>
          {#if activity.location}
            <span class="popup-location">{activity.location}</span>
          {/if}
        </div>
      {/each}
    </div>
  </div>
</div>

<style>
  .popup-overlay {
    position: fixed;
    inset: 0;
    z-index: 80;
  }

  .popup {
    position: fixed;
    background: white;
    border-radius: 10px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
    width: 300px;
    max-height: 280px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    z-index: 81;
  }

  .popup-header {
    padding: 0.6rem 0.75rem;
    border-bottom: 1px solid #eee;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .popup-title {
    font-weight: 700;
    font-size: 0.9rem;
    color: #333;
    flex: 1;
  }

  .popup-count {
    font-size: 0.7rem;
    color: #999;
  }

  .popup-edit {
    font-size: 0.7rem;
    color: #3b82f6;
    cursor: pointer;
    margin-left: auto;
  }

  .popup-edit:hover {
    text-decoration: underline;
  }

  .popup-list {
    overflow-y: auto;
    padding: 0.25rem 0;
  }

  .popup-activity {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.75rem;
    cursor: pointer;
    font-size: 0.8rem;
  }

  .popup-activity:hover {
    background: #f5f5f5;
  }

  .popup-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .popup-dates {
    color: #888;
    font-size: 0.7rem;
    min-width: 80px;
    white-space: nowrap;
  }

  .popup-name {
    flex: 1;
    font-weight: 500;
    color: #333;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .popup-location {
    color: #aaa;
    font-size: 0.7rem;
    flex-shrink: 0;
  }
</style>
