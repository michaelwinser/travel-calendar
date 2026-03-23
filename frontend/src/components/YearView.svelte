<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { ACTIVITY_COLORS, type Activity } from '../lib/api';
  import {
    today, addDays, getMonthsForRange, hasConflict,
    type YearMonth,
  } from '../lib/date-utils';

  interface Props {
    activities: Activity[];
    onswitchtomonth: (date: string) => void;
  }

  let { activities, onswitchtomonth }: Props = $props();

  // Show 6 months back, 12 months forward
  let rangeStart = $state(addDays(today(), -180));
  let rangeEnd = $state(addDays(today(), 365));

  let months = $derived(getMonthsForRange(rangeStart, rangeEnd));

  // Build activity lookup: date → activities
  let dateActivities = $derived.by(() => {
    const map = new Map<string, Activity[]>();
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

  let scrollEl: HTMLElement;

  onMount(async () => {
    await tick();
    scrollToToday();
  });

  export function scrollToToday() {
    const todayStr = today();
    // Find the month containing today
    const el = scrollEl?.querySelector(`[data-month-contains="${todayStr.slice(0, 7)}"]`);
    if (el) {
      el.scrollIntoView({ block: 'center', behavior: 'smooth' });
    }
  }

  function handleScroll() {
    if (!scrollEl) return;
    const { scrollTop, scrollHeight, clientHeight } = scrollEl;

    if (scrollTop < 200) {
      const prevHeight = scrollEl.scrollHeight;
      rangeStart = addDays(rangeStart, -180);
      tick().then(() => {
        const newHeight = scrollEl.scrollHeight;
        scrollEl.scrollTop += newHeight - prevHeight;
      });
    }

    if (scrollHeight - scrollTop - clientHeight < 200) {
      rangeEnd = addDays(rangeEnd, 180);
    }
  }

  function isToday(dateStr: string): boolean {
    return dateStr === today();
  }

  function dayColor(dateStr: string): string | null {
    const acts = dateActivities.get(dateStr);
    if (!acts || acts.length === 0) return null;
    // Use the first activity's color (or blend if multiple — for now, first wins)
    return ACTIVITY_COLORS[acts[0].type] ?? null;
  }

  function dayHasConflict(dateStr: string): boolean {
    const acts = dateActivities.get(dateStr);
    if (!acts || acts.length < 2) return false;
    return hasConflict(dateStr, activities);
  }
</script>

<div
  class="year-view"
  bind:this={scrollEl}
  onscroll={handleScroll}
>
  {#each months as m (m.label)}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
      class="month-row"
      data-month-contains="{m.year}-{String(m.month + 1).padStart(2, '0')}"
      onclick={() => onswitchtomonth(m.days[0])}
    >
      <div class="month-name">
        <span class="month-name-text">{m.label}</span>
      </div>
      <div class="day-grid">
        {#each m.days as dateStr, di}
          {@const color = dayColor(dateStr)}
          {@const conflict = dayHasConflict(dateStr)}
          <div
            class="year-day"
            class:has-activity={color !== null}
            class:today={isToday(dateStr)}
            class:conflict={conflict}
            style={color ? `background: ${color};` : ''}
            title={dateStr}
          ></div>
        {/each}
      </div>
    </div>
  {/each}
</div>

<style>
  .year-view {
    overflow-y: auto;
    max-height: calc(100vh - 180px);
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    background: white;
  }

  .month-row {
    display: flex;
    align-items: center;
    border-bottom: 1px solid #f0f0f0;
    padding: 0.5rem 0;
    cursor: pointer;
  }

  .month-row:hover {
    background: rgba(59, 130, 246, 0.03);
  }

  .month-name {
    width: 90px;
    flex-shrink: 0;
    padding: 0 0.75rem;
  }

  .month-name-text {
    font-size: 0.75rem;
    font-weight: 700;
    color: #555;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .day-grid {
    flex: 1;
    display: flex;
    gap: 1px;
    padding-right: 0.5rem;
  }

  .year-day {
    flex: 1;
    height: 22px;
    border-radius: 2px;
    background: #f5f5f5;
    transition: transform 0.1s;
  }

  .year-day.has-activity {
    opacity: 0.85;
  }

  .year-day.has-activity:hover {
    opacity: 1;
    transform: scaleY(1.3);
  }

  .year-day.today {
    outline: 2px solid #3b82f6;
    outline-offset: -1px;
    z-index: 1;
  }

  .year-day.conflict {
    background: #ef4444 !important;
    opacity: 0.9;
  }
</style>
