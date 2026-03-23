<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { ACTIVITY_COLORS, type Activity } from '../lib/api';
  import {
    today, addDays, getMonthsForRange, getYearBarsForMonth,
    minDate, maxDate,
    type YearMonth, type YearBar,
  } from '../lib/date-utils';

  interface Props {
    activities: Activity[];
    initialDate?: string;
    onedit: (activity: Activity) => void;
    ondayclick: (date: string) => void;
    ondragselect: (startDate: string, endDate: string) => void;
    onswitchtomonth: (date: string) => void;
  }

  let { activities, initialDate, onedit, ondayclick, ondragselect, onswitchtomonth }: Props = $props();

  const MAX_BAR_LANES = 3;

  // Show 6 months back, 12 months forward
  let rangeStart = $state(addDays(today(), -180));
  let rangeEnd = $state(addDays(today(), 365));

  let months = $derived(getMonthsForRange(rangeStart, rangeEnd));

  let monthData = $derived(
    months.map(m => ({
      month: m,
      bars: getYearBarsForMonth(activities, m, ACTIVITY_COLORS, MAX_BAR_LANES),
    }))
  );

  // Drag state
  let dragStartDate = $state<string | null>(null);
  let dragCurrentDate = $state<string | null>(null);
  let isDragging = $state(false);

  let dragRange = $derived(
    dragStartDate && dragCurrentDate
      ? { start: minDate(dragStartDate, dragCurrentDate), end: maxDate(dragStartDate, dragCurrentDate) }
      : null
  );

  function isInDragRange(dateStr: string): boolean {
    if (!dragRange) return false;
    return dateStr >= dragRange.start && dateStr <= dragRange.end;
  }

  let scrollEl: HTMLElement;

  onMount(async () => {
    await tick();
    scrollToDate(initialDate ?? today());
  });

  export function scrollToDate(dateStr: string) {
    const monthKey = dateStr.slice(0, 7);
    const el = scrollEl?.querySelector(`[data-month-contains="${monthKey}"]`);
    if (el) {
      el.scrollIntoView({ block: 'center', behavior: 'smooth' });
      return;
    }
    rangeStart = addDays(dateStr, -180);
    rangeEnd = addDays(dateStr, 365);
    tick().then(() => {
      const el2 = scrollEl?.querySelector(`[data-month-contains="${monthKey}"]`);
      if (el2) el2.scrollIntoView({ block: 'center' });
    });
  }

  export function scrollToToday() {
    scrollToDate(today());
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

  function handleDayMouseDown(dateStr: string, e: MouseEvent) {
    if (e.button !== 0) return;
    dragStartDate = dateStr;
    dragCurrentDate = dateStr;
    isDragging = true;
    e.preventDefault();
    e.stopPropagation(); // prevent month-row click
  }

  function handleDayMouseEnter(dateStr: string) {
    if (isDragging) {
      dragCurrentDate = dateStr;
    }
  }

  function handleMouseUp() {
    if (!isDragging || !dragStartDate || !dragCurrentDate) {
      isDragging = false;
      dragStartDate = null;
      dragCurrentDate = null;
      return;
    }

    const start = minDate(dragStartDate, dragCurrentDate);
    const end = maxDate(dragStartDate, dragCurrentDate);

    if (start === end) {
      ondayclick(start);
    } else {
      ondragselect(start, end);
    }

    isDragging = false;
    dragStartDate = null;
    dragCurrentDate = null;
  }

  function handleBarClick(activity: Activity, e: MouseEvent) {
    e.stopPropagation();
    onedit(activity);
  }

  function isToday(dateStr: string): boolean {
    return dateStr === today();
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="year-view"
  bind:this={scrollEl}
  onscroll={handleScroll}
  onmouseup={handleMouseUp}
  onmouseleave={handleMouseUp}
>
  {#each monthData as { month: m, bars } (m.label)}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
      class="month-row"
      data-month-contains="{m.year}-{String(m.month + 1).padStart(2, '0')}"
    >
      <!-- Month label (click to switch to month view) -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <div class="month-name" onclick={() => onswitchtomonth(m.days[0])}>
        <span class="month-name-month">{['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'][m.month]}</span>
        <span class="month-name-year">{m.year}</span>
      </div>

      <!-- Day grid: numbers + bar lanes -->
      <div class="day-grid" style="--cols: {m.days.length}">
        <!-- Day numbers (interactive for click/drag) -->
        {#each m.days as dateStr, di}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div
            class="day-num"
            class:today={isToday(dateStr)}
            class:weekend={new Date(m.year, m.month, di + 1).getDay() === 0 || new Date(m.year, m.month, di + 1).getDay() === 6}
            class:drag-selected={isInDragRange(dateStr)}
            style="grid-column: {di + 1}; grid-row: 1 / -1;"
            onmousedown={(e) => handleDayMouseDown(dateStr, e)}
            onmouseenter={() => handleDayMouseEnter(dateStr)}
          >
            {di + 1}
          </div>
        {/each}

        <!-- Activity bars -->
        {#each bars as bar}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <div
            class="year-bar"
            style="
              grid-column: {bar.startDay + 1} / span {bar.spanDays};
              grid-row: {bar.lane + 2};
              background: {bar.color};
            "
            title="{bar.activity.title}{bar.activity.location ? ' — ' + bar.activity.location : ''}"
            onclick={(e) => handleBarClick(bar.activity, e)}
          >
            <span class="year-bar-label">{bar.activity.title}</span>
          </div>
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
    user-select: none;
  }

  .month-row {
    display: flex;
    align-items: flex-start;
    border-bottom: 1px solid #eee;
    padding: 0.5rem 0;
    min-height: 40px;
  }

  .month-name {
    width: 56px;
    flex-shrink: 0;
    padding: 0 0.5rem;
    text-align: right;
    line-height: 1.2;
    cursor: pointer;
  }

  .month-name:hover {
    color: #3b82f6;
  }

  .month-name-month {
    display: block;
    font-size: 0.8rem;
    font-weight: 800;
    color: #444;
  }

  .month-name-year {
    display: block;
    font-size: 0.65rem;
    font-weight: 500;
    color: #aaa;
  }

  .day-grid {
    flex: 1;
    display: grid;
    grid-template-columns: repeat(var(--cols), 1fr);
    grid-auto-rows: auto;
    gap: 0 0;
    padding-right: 0.25rem;
    min-height: 32px;
  }

  .day-num {
    text-align: center;
    font-size: 0.6rem;
    color: #999;
    padding-top: 0;
    line-height: 1.4;
    cursor: crosshair;
    z-index: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .day-num.today {
    color: white;
    font-weight: 700;
    background: #3b82f6;
    border-radius: 50%;
    width: 18px;
    height: 18px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin: 0 auto;
  }

  .day-num.weekend {
    color: #ccc;
  }

  .day-num.drag-selected {
    background: rgba(59, 130, 246, 0.2);
    border-radius: 2px;
  }

  .day-num:hover:not(.today) {
    background: rgba(59, 130, 246, 0.08);
    border-radius: 2px;
  }

  .year-bar {
    height: 14px;
    display: flex;
    align-items: center;
    padding: 0 3px;
    border-radius: 3px;
    margin: 1px 1px;
    overflow: hidden;
    opacity: 0.85;
    cursor: pointer;
    z-index: 1;
    position: relative;
  }

  .year-bar:hover {
    opacity: 1;
    filter: brightness(1.1);
  }

  .year-bar-label {
    font-size: 0.55rem;
    color: white;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
