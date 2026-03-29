<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { ACTIVITY_COLORS, type Activity, type TripSummary } from '../lib/api';
  import TripDetailPopup from './TripDetailPopup.svelte';
  import {
    today, addDays, getMonthsForRange, getTripBarsForMonth,
    minDate, maxDate, hasConflict,
    type YearMonth, type TripYearBar,
  } from '../lib/date-utils';

  interface Props {
    activities: Activity[];
    trips: TripSummary[];
    initialDate?: string;
    onedit: (activity: Activity) => void;
    ondayclick: (date: string) => void;
    ondragselect: (startDate: string, endDate: string) => void;
    onswitchtomonth: (date: string) => void;
    onedittrip?: (tripId: string) => void;
    onassigntotrip?: (activityId: string, tripId: string) => void;
    onfocusdate?: (date: string) => void;
  }

  let { activities, trips, initialDate, onedit, ondayclick, ondragselect, onswitchtomonth, onedittrip, onassigntotrip, onfocusdate }: Props = $props();

  const MAX_BAR_LANES = 3;

  // Build trip lookup
  let tripLookup = $derived.by(() => {
    const map = new Map<string, { name: string; color: string }>();
    for (const t of trips) map.set(t.id, { name: t.name, color: t.color });
    return map;
  });

  let rangeStart = $state(addDays(today(), -180));
  let rangeEnd = $state(addDays(today(), 365));

  let months = $derived(getMonthsForRange(rangeStart, rangeEnd));

  let monthData = $derived(
    months.map(m => ({
      month: m,
      bars: getTripBarsForMonth(activities, m, tripLookup, ACTIVITY_COLORS, MAX_BAR_LANES),
      conflicts: new Set(m.days.filter(d => hasConflict(d, activities))),
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

  // Trip detail popup
  let popupBar = $state<TripYearBar | null>(null);
  let popupX = $state(0);
  let popupY = $state(0);

  let popupTripActivities = $derived.by(() => {
    if (!popupBar) return [];
    return activities.filter(a => a.tripId === popupBar!.tripId);
  });

  let popupUnassigned = $derived.by(() => {
    if (!popupBar) return [];
    const tripActs = popupTripActivities;
    if (tripActs.length === 0) return [];
    const tripStart = tripActs.reduce((min, a) => a.startDate < min ? a.startDate : min, tripActs[0].startDate);
    const tripEnd = tripActs.reduce((max, a) => (a.endDate || a.startDate) > max ? (a.endDate || a.startDate) : max, tripActs[0].endDate);
    return activities.filter(a =>
      !a.tripId &&
      a.startDate <= tripEnd &&
      (a.endDate || a.startDate) >= tripStart
    );
  });

  // Scroll
  let scrollEl: HTMLElement;
  let scrollDebounce: ReturnType<typeof setTimeout> | undefined;

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

  export function scrollToToday() { scrollToDate(today()); }

  export function scrollAction(action: 'pageDown' | 'pageUp' | 'nextActivity' | 'prevActivity') {
    if (!scrollEl) return;
    const { clientHeight } = scrollEl;
    if (action === 'pageDown') {
      scrollEl.scrollBy({ top: clientHeight * 0.8, behavior: 'smooth' });
    } else if (action === 'pageUp') {
      scrollEl.scrollBy({ top: -clientHeight * 0.8, behavior: 'smooth' });
    } else {
      const bars = scrollEl.querySelectorAll('.year-bar:not(.ghost)');
      if (!bars.length) return;
      const containerTop = scrollEl.getBoundingClientRect().top;
      const center = containerTop + clientHeight / 2;
      if (action === 'nextActivity') {
        for (const bar of bars) {
          if (bar.getBoundingClientRect().top > center + 10) {
            bar.scrollIntoView({ block: 'center', behavior: 'smooth' });
            return;
          }
        }
      } else {
        const arr = Array.from(bars);
        for (let i = arr.length - 1; i >= 0; i--) {
          if (arr[i].getBoundingClientRect().bottom < center - 10) {
            arr[i].scrollIntoView({ block: 'center', behavior: 'smooth' });
            return;
          }
        }
      }
    }
  }

  function handleScroll() {
    if (!scrollEl) return;
    const { scrollTop, scrollHeight, clientHeight } = scrollEl;
    if (scrollTop < 200) {
      const prevHeight = scrollEl.scrollHeight;
      rangeStart = addDays(rangeStart, -180);
      tick().then(() => { scrollEl.scrollTop += scrollEl.scrollHeight - prevHeight; });
    }
    if (scrollHeight - scrollTop - clientHeight < 200) {
      rangeEnd = addDays(rangeEnd, 180);
    }
    if (onfocusdate) {
      clearTimeout(scrollDebounce);
      scrollDebounce = setTimeout(() => reportTopDate(), 500);
    }
  }

  function reportTopDate() {
    if (!scrollEl || !onfocusdate) return;
    const containerTop = scrollEl.getBoundingClientRect().top;
    for (const el of scrollEl.querySelectorAll('[data-month-contains]')) {
      if (el.getBoundingClientRect().top >= containerTop) {
        const month = el.getAttribute('data-month-contains');
        if (month) { onfocusdate(month + '-01'); return; }
      }
    }
  }

  function handleDayMouseDown(dateStr: string, e: MouseEvent) {
    if (e.button !== 0) return;
    dragStartDate = dateStr;
    dragCurrentDate = dateStr;
    isDragging = true;
    e.preventDefault();
    e.stopPropagation();
  }

  function handleDayMouseEnter(dateStr: string) {
    if (isDragging) dragCurrentDate = dateStr;
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
    if (start === end) ondayclick(start);
    else ondragselect(start, end);
    isDragging = false;
    dragStartDate = null;
    dragCurrentDate = null;
  }

  function handleBarClick(bar: TripYearBar, e: MouseEvent) {
    e.stopPropagation();
    if (bar.tripId === null && bar.activities.length === 1) {
      // Standalone activity — open edit directly
      onedit(bar.activities[0]);
    } else {
      // Trip — show detail popup
      popupBar = bar;
      popupX = e.clientX;
      popupY = e.clientY;
    }
  }

  function isToday(dateStr: string): boolean { return dateStr === today(); }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="year-view"
  bind:this={scrollEl}
  onscroll={handleScroll}
  onmouseup={handleMouseUp}
  onmouseleave={handleMouseUp}
>
  {#each monthData as { month: m, bars, conflicts } (m.label)}
    <div
      class="month-row"
      data-month-contains="{m.year}-{String(m.month + 1).padStart(2, '0')}"
    >
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <div class="month-name" onclick={() => onswitchtomonth(m.days[0])}>
        <span class="month-name-month">{['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'][m.month]}</span>
        <span class="month-name-year">{m.year}</span>
      </div>

      <div class="day-grid" style="--cols: {m.days.length}">
        {#each m.days as dateStr, di}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div
            class="day-num"
            class:today={isToday(dateStr)}
            class:weekend={new Date(m.year, m.month, di + 1).getDay() === 0 || new Date(m.year, m.month, di + 1).getDay() === 6}
            class:drag-selected={isInDragRange(dateStr)}
            class:conflict={conflicts.has(dateStr)}
            style="grid-column: {di + 1}; grid-row: 1 / -1;"
            onmousedown={(e) => handleDayMouseDown(dateStr, e)}
            onmouseenter={() => handleDayMouseEnter(dateStr)}
          >
            {di + 1}
          </div>
        {/each}

        <!-- Ghost bar during drag -->
        {#if isDragging && dragRange}
          {@const ghostStart = m.days.indexOf(dragRange.start < m.days[0] ? m.days[0] : dragRange.start)}
          {@const ghostEnd = m.days.indexOf(dragRange.end > m.days[m.days.length - 1] ? m.days[m.days.length - 1] : dragRange.end)}
          {#if ghostStart >= 0 && ghostEnd >= 0}
            <div
              class="year-bar ghost"
              style="grid-column: {ghostStart + 1} / span {ghostEnd - ghostStart + 1}; grid-row: {(bars.length > 0 ? Math.max(...bars.map(b => b.lane)) + 2 : 2)};"
            ></div>
          {/if}
        {/if}

        <!-- Trip bars -->
        {#each bars as bar}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <div
            class="year-bar"
            class:is-trip={bar.tripId !== null}
            style="
              grid-column: {bar.startDay + 1} / span {bar.spanDays};
              grid-row: {bar.lane + 2};
              background: {bar.color};
            "
            onclick={(e) => handleBarClick(bar, e)}
          >
            <span class="year-bar-label">{bar.tripName}</span>
            {#if bar.activityCount > 1}
              <span class="year-bar-count">{bar.activityCount}</span>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {/each}
</div>

<!-- Trip detail popup -->
{#if popupBar}
  <TripDetailPopup
    tripId={popupBar.tripId}
    tripName={popupBar.tripName}
    color={popupBar.color}
    activities={popupTripActivities}
    unassignedActivities={popupUnassigned}
    x={popupX}
    y={popupY}
    {onedit}
    {onedittrip}
    onassign={onassigntotrip}
    onclose={() => popupBar = null}
  />
{/if}

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
    min-height: 50px;
  }

  .month-name {
    width: 56px;
    flex-shrink: 0;
    padding: 0 0.5rem;
    text-align: right;
    line-height: 1.2;
    cursor: pointer;
  }

  .month-name:hover { color: #3b82f6; }

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

  .day-num.weekend { color: #ccc; }

  .day-num.conflict {
    color: #dc2626;
    font-weight: 700;
    background: rgba(239, 68, 68, 0.1);
    border-radius: 2px;
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
    height: 18px;
    display: flex;
    align-items: center;
    padding: 0 5px;
    border-radius: 4px;
    margin: 1px 1px;
    overflow: hidden;
    opacity: 0.85;
    cursor: pointer;
    z-index: 1;
    position: relative;
    gap: 3px;
  }

  .year-bar.is-trip {
    height: 24px;
  }

  .year-bar.ghost {
    background: rgba(0, 0, 0, 0.1);
    border: 1px dashed rgba(0, 0, 0, 0.2);
    pointer-events: none;
  }

  .year-bar:hover {
    opacity: 1;
    filter: brightness(1.1);
  }

  .year-bar-label {
    font-size: 0.65rem;
    color: white;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex: 1;
  }

  .year-bar-count {
    font-size: 0.55rem;
    color: rgba(255, 255, 255, 0.8);
    background: rgba(0, 0, 0, 0.15);
    border-radius: 3px;
    padding: 0 3px;
    flex-shrink: 0;
  }
</style>
