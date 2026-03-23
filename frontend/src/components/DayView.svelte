<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { ACTIVITY_COLORS, type Activity } from '../lib/api';
  import {
    today, addDays, stringToDate, hasConflict,
    monthLabel,
  } from '../lib/date-utils';

  interface Props {
    activities: Activity[];
    onedit: (activity: Activity) => void;
    ondayclick: (date: string) => void;
  }

  let { activities, onedit, ondayclick }: Props = $props();

  const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

  // View range
  let rangeStart = $state(addDays(today(), -60));
  let rangeEnd = $state(addDays(today(), 120));

  // Generate all dates in range
  let dates = $derived.by(() => {
    const result: string[] = [];
    let d = rangeStart;
    while (d <= rangeEnd) {
      result.push(d);
      d = addDays(d, 1);
    }
    return result;
  });

  // Build a lookup: date → activities overlapping that date
  let dateActivities = $derived.by(() => {
    const map = new Map<string, Activity[]>();
    // For each activity, mark all dates it spans
    for (const a of activities) {
      let d = a.startDate < rangeStart ? rangeStart : a.startDate;
      const end = a.endDate > rangeEnd ? rangeEnd : a.endDate;
      while (d <= end) {
        const list = map.get(d);
        if (list) list.push(a);
        else map.set(d, [a]);
        d = addDays(d, 1);
      }
    }
    return map;
  });

  // Month labels: show when month changes
  let dateMonthLabels = $derived.by(() => {
    const labels = new Map<string, string>();
    for (let i = 0; i < dates.length; i++) {
      const d = dates[i];
      const ml = monthLabel(d);
      if (i === 0 || ml !== monthLabel(dates[i - 1])) {
        labels.set(d, ml);
      }
    }
    return labels;
  });

  let scrollEl: HTMLElement;

  onMount(async () => {
    await tick();
    scrollToDate(today());
  });

  export function scrollToDate(dateStr: string) {
    const el = scrollEl?.querySelector(`[data-date="${dateStr}"]`);
    if (el) {
      el.scrollIntoView({ block: 'center', behavior: 'smooth' });
    }
  }

  export function scrollToToday() {
    scrollToDate(today());
  }

  function handleScroll() {
    if (!scrollEl) return;
    const { scrollTop, scrollHeight, clientHeight } = scrollEl;

    if (scrollTop < 300) {
      const prevHeight = scrollEl.scrollHeight;
      rangeStart = addDays(rangeStart, -60);
      tick().then(() => {
        const newHeight = scrollEl.scrollHeight;
        scrollEl.scrollTop += newHeight - prevHeight;
      });
    }

    if (scrollHeight - scrollTop - clientHeight < 300) {
      rangeEnd = addDays(rangeEnd, 60);
    }
  }

  function formatDate(dateStr: string): { day: string; dow: string; isWeekend: boolean } {
    const d = stringToDate(dateStr);
    const dayNum = d.getDate();
    const dow = DAY_NAMES[d.getDay()];
    const isWeekend = d.getDay() === 0 || d.getDay() === 6;
    return { day: String(dayNum), dow, isWeekend };
  }

  function isToday(dateStr: string): boolean {
    return dateStr === today();
  }
</script>

<div
  class="day-view"
  bind:this={scrollEl}
  onscroll={handleScroll}
>
  {#each dates as dateStr (dateStr)}
    {@const info = formatDate(dateStr)}
    {@const dayActs = dateActivities.get(dateStr) ?? []}
    {@const conflict = dayActs.length > 1 && hasConflict(dateStr, activities)}
    {@const label = dateMonthLabels.get(dateStr)}

    {#if label}
      <div class="month-divider">
        <span class="month-divider-label">{label}</span>
      </div>
    {/if}

    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
      class="day-row"
      class:today={isToday(dateStr)}
      class:weekend={info.isWeekend}
      class:conflict={conflict}
      class:has-activities={dayActs.length > 0}
      data-date={dateStr}
      onclick={() => { if (dayActs.length === 0) ondayclick(dateStr); }}
    >
      <div class="date-col">
        <span class="dow">{info.dow}</span>
        <span class="day-num" class:today-num={isToday(dateStr)}>{info.day}</span>
      </div>

      <div class="activities-col">
        {#if dayActs.length === 0}
          <span class="home-label">Home</span>
        {:else}
          {#each dayActs as activity (activity.id)}
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <div
              class="activity-chip"
              onclick={(e) => { e.stopPropagation(); onedit(activity); }}
            >
              <span class="chip-dot" style="background: {ACTIVITY_COLORS[activity.type]}"></span>
              <span class="chip-title">{activity.title}</span>
              {#if activity.location}
                <span class="chip-location">{activity.location}</span>
              {/if}
              <span class="chip-type">{activity.type}</span>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  {/each}
</div>

<style>
  .day-view {
    overflow-y: auto;
    max-height: calc(100vh - 180px);
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    background: white;
  }

  .month-divider {
    position: sticky;
    top: 0;
    z-index: 5;
    background: #f8f9fa;
    border-bottom: 2px solid #e5e7eb;
    padding: 0.5rem 1rem;
  }

  .month-divider-label {
    font-size: 0.85rem;
    font-weight: 800;
    color: #444;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .day-row {
    display: flex;
    align-items: flex-start;
    border-bottom: 1px solid #f0f0f0;
    padding: 0.4rem 0;
    cursor: pointer;
    min-height: 36px;
  }

  .day-row:hover {
    background: rgba(59, 130, 246, 0.04);
  }

  .day-row.today {
    background: rgba(59, 130, 246, 0.05);
  }

  .day-row.weekend:not(.has-activities) {
    opacity: 0.6;
  }

  .day-row.conflict {
    background: rgba(239, 68, 68, 0.06);
  }

  .date-col {
    width: 72px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0.15rem 0.75rem;
  }

  .dow {
    font-size: 0.7rem;
    color: #aaa;
    width: 24px;
    text-transform: uppercase;
  }

  .day-num {
    font-size: 0.85rem;
    color: #666;
    font-weight: 500;
  }

  .today-num {
    background: #3b82f6;
    color: white;
    border-radius: 50%;
    width: 24px;
    height: 24px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    font-size: 0.75rem;
  }

  .activities-col {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 0.1rem 0.5rem 0.1rem 0;
    min-height: 0;
  }

  .home-label {
    font-size: 0.8rem;
    color: #ccc;
    padding: 0.1rem 0;
  }

  .activity-chip {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
  }

  .activity-chip:hover {
    background: rgba(0, 0, 0, 0.04);
  }

  .chip-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .chip-title {
    font-weight: 500;
    color: #333;
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .chip-location {
    color: #888;
    font-size: 0.75rem;
    flex-shrink: 0;
  }

  .chip-type {
    color: #bbb;
    font-size: 0.65rem;
    text-transform: capitalize;
    flex-shrink: 0;
  }
</style>
