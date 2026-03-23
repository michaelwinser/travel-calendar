<script lang="ts">
  import { ACTIVITY_COLORS, type Activity } from '../lib/api';

  interface Props {
    activity: Activity | null;
    x: number;
    y: number;
  }

  let { activity, x, y }: Props = $props();

  function formatDates(a: Activity): string {
    if (a.startDate === a.endDate) return a.startDate;
    return `${a.startDate} \u2192 ${a.endDate}`;
  }
</script>

{#if activity}
  <div
    class="tooltip"
    style="left: {x}px; top: {y}px;"
  >
    <div class="tooltip-header">
      <span class="tooltip-dot" style="background: {ACTIVITY_COLORS[activity.type]}"></span>
      <span class="tooltip-title">{activity.title}</span>
    </div>
    <div class="tooltip-dates">{formatDates(activity)}</div>
    {#if activity.location}
      <div class="tooltip-location">{activity.location}</div>
    {/if}
    <div class="tooltip-type">{activity.type}</div>
  </div>
{/if}

<style>
  .tooltip {
    position: fixed;
    z-index: 200;
    background: #1f2937;
    color: white;
    border-radius: 8px;
    padding: 0.5rem 0.65rem;
    font-size: 0.75rem;
    pointer-events: none;
    max-width: 250px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
    transform: translate(-50%, -100%) translateY(-8px);
    line-height: 1.4;
  }

  .tooltip-header {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    margin-bottom: 0.2rem;
  }

  .tooltip-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .tooltip-title {
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tooltip-dates {
    color: #e5e7eb;
  }

  .tooltip-location {
    color: #d1d5db;
  }

  .tooltip-type {
    color: #9ca3af;
    text-transform: capitalize;
    font-size: 0.65rem;
  }
</style>
