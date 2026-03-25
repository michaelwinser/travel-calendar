<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { ACTIVITY_COLORS, type Activity, type ActivityType, type TripSummary } from '../lib/api';
  import Tooltip from './Tooltip.svelte';
  import TripDetailPopup from './TripDetailPopup.svelte';
  import {
    today, addDays, stringToDate,
    getWeeksForRange, computeTripLanes, getDayTripSegments,
    monthLabel, monthIndex, hasConflict,
    minDate, maxDate,
    type TripLane, type DayTripSegment,
  } from '../lib/date-utils';

  interface Props {
    activities: Activity[];
    trips: TripSummary[];
    ghostDates?: { startDate: string; endDate: string; type: ActivityType } | null;
    initialDate?: string;
    onedit: (activity: Activity) => void;
    ondayclick: (date: string) => void;
    ondragselect: (startDate: string, endDate: string) => void;
    onresize?: (activityId: string, startDate: string, endDate: string) => void;
    onmove?: (activityId: string, startDate: string, endDate: string) => void;
    onfocusdate?: (date: string) => void;
  }

  let { activities, trips, ghostDates, initialDate, onedit, ondayclick, ondragselect, onresize, onmove, onfocusdate }: Props = $props();

  const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
  const MAX_LANES = 4;

  // Trip lookup
  let tripLookup = $derived.by(() => {
    const map = new Map<string, { name: string; color: string }>();
    for (const t of trips) map.set(t.id, { name: t.name, color: t.color });
    return map;
  });

  // View range
  let rangeStart = $state(addDays(today(), -90));
  let rangeEnd = $state(addDays(today(), 180));

  let weeks = $derived(getWeeksForRange(rangeStart, rangeEnd));

  // Global trip lane assignment
  let tripLanes = $derived(computeTripLanes(activities, tripLookup, ACTIVITY_COLORS, MAX_LANES));

  // Month labels
  let weekMonthLabels = $derived(
    weeks.map((week, i) => {
      const thursdayMonth = monthLabel(week.days[4]);
      if (i === 0) return thursdayMonth;
      const prevThursdayMonth = monthLabel(weeks[i - 1].days[4]);
      return thursdayMonth !== prevThursdayMonth ? thursdayMonth : '';
    })
  );

  // Max lane used (for row height)
  let maxLane = $derived(tripLanes.reduce((max, tl) => Math.max(max, tl.lane), -1));

  // === DRAG STATE ===
  type DragMode = 'create' | null;
  let dragMode = $state<DragMode>(null);
  let createStart = $state<string | null>(null);
  let createCurrent = $state<string | null>(null);

  let createRange = $derived(
    createStart && createCurrent
      ? { start: minDate(createStart, createCurrent), end: maxDate(createStart, createCurrent) }
      : null
  );

  function isInCreateRange(dateStr: string): boolean {
    if (!createRange) return false;
    return dateStr >= createRange.start && dateStr <= createRange.end;
  }

  // Trip detail popup
  let popupTripLane = $state<TripLane | null>(null);
  let popupX = $state(0);
  let popupY = $state(0);

  // Tooltip
  let tooltipActivity = $state<Activity | null>(null);
  let tooltipX = $state(0);
  let tooltipY = $state(0);

  // Scroll
  let scrollEl: HTMLElement;
  let scrollDebounce: ReturnType<typeof setTimeout> | undefined;

  onMount(async () => {
    await tick();
    scrollToDate(initialDate ?? today());
  });

  export function scrollToDate(dateStr: string) {
    const el = scrollEl?.querySelector(`[data-date="${dateStr}"]`);
    if (el) {
      el.scrollIntoView({ block: 'center', behavior: 'smooth' });
      return;
    }
    rangeStart = addDays(dateStr, -90);
    rangeEnd = addDays(dateStr, 180);
    tick().then(() => {
      const el2 = scrollEl?.querySelector(`[data-date="${dateStr}"]`);
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
      const bars = scrollEl.querySelectorAll('.bar-segment:not(.ghost):not(.bar-slot-empty)');
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
      rangeStart = addDays(rangeStart, -90);
      tick().then(() => { scrollEl.scrollTop += scrollEl.scrollHeight - prevHeight; });
    }
    if (scrollHeight - scrollTop - clientHeight < 200) {
      rangeEnd = addDays(rangeEnd, 90);
    }
    if (onfocusdate) {
      clearTimeout(scrollDebounce);
      scrollDebounce = setTimeout(() => reportTopDate(), 500);
    }
  }

  function reportTopDate() {
    if (!scrollEl || !onfocusdate) return;
    const containerTop = scrollEl.getBoundingClientRect().top;
    for (const el of scrollEl.querySelectorAll('[data-date]')) {
      if (el.getBoundingClientRect().top >= containerTop) {
        const date = el.getAttribute('data-date');
        if (date) { onfocusdate(date); return; }
      }
    }
  }

  // Day cell interactions
  function handleDayMouseDown(dateStr: string, e: MouseEvent) {
    if (e.button !== 0) return;
    const target = e.target as HTMLElement;
    if (target.closest('.bar-segment')) return;
    e.preventDefault();
    dragMode = 'create';
    createStart = dateStr;
    createCurrent = dateStr;
  }

  function handleDayMouseEnter(dateStr: string) {
    if (dragMode === 'create') createCurrent = dateStr;
  }

  function handleMouseUp() {
    if (dragMode === 'create' && createStart && createCurrent) {
      const start = minDate(createStart, createCurrent);
      const end = maxDate(createStart, createCurrent);
      if (start === end) ondayclick(start);
      else ondragselect(start, end);
    }
    dragMode = null;
    createStart = null;
    createCurrent = null;
  }

  // Bar interactions
  function handleBarClick(tripLane: TripLane, e: MouseEvent) {
    e.stopPropagation();
    if (tripLane.tripId === null && tripLane.activities.length === 1) {
      // Standalone activity — edit directly
      onedit(tripLane.activities[0]);
    } else if (tripLane.tripId !== null) {
      // Trip — show popup
      popupTripLane = tripLane;
      popupX = e.clientX;
      popupY = e.clientY;
    }
  }

  function handleBarEnter(tripLane: TripLane, e: MouseEvent) {
    if (dragMode) return;
    if (tripLane.activities.length === 1) {
      tooltipActivity = tripLane.activities[0];
    } else {
      // For trips, show first activity in tooltip (or could show trip summary)
      tooltipActivity = tripLane.activities[0];
    }
    tooltipX = e.clientX;
    tooltipY = e.clientY;
  }

  function handleBarHover(e: MouseEvent) {
    if (dragMode) { tooltipActivity = null; return; }
    tooltipX = e.clientX;
    tooltipY = e.clientY;
  }

  function handleBarLeave() { tooltipActivity = null; }

  // Helpers
  function dayOfMonth(dateStr: string): number { return stringToDate(dateStr).getDate(); }
  function isToday(dateStr: string): boolean { return dateStr === today(); }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="month-view"
  bind:this={scrollEl}
  onscroll={handleScroll}
  onmouseup={handleMouseUp}
  onmouseleave={handleMouseUp}
>
  <div class="week-header">
    <div class="label-col"></div>
    {#each DAY_NAMES as name}
      <div class="day-header">{name}</div>
    {/each}
  </div>

  {#each weeks as week, wi (week.weekStart)}
    {@const label = weekMonthLabels[wi]}

    <div class="week-row">
      <div class="label-col">
        {#if label}<span class="month-label">{label}</span>{/if}
      </div>

      {#each week.days as dateStr, di}
        {@const altMonth = monthIndex(dateStr) % 2 === 1}
        {@const daySegs = getDayTripSegments(dateStr, tripLanes)}
        {@const conflict = hasConflict(dateStr, activities)}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="day-cell"
          class:alt-month={altMonth}
          class:today={isToday(dateStr)}
          class:drag-selected={dragMode === 'create' && isInCreateRange(dateStr)}
          class:conflict={conflict}
          data-date={dateStr}
          onmousedown={(e) => handleDayMouseDown(dateStr, e)}
          onmouseenter={() => handleDayMouseEnter(dateStr)}
        >
          <div class="day-number-row">
            <span class="day-number" class:today-number={isToday(dateStr)}>
              {dayOfMonth(dateStr)}
            </span>
          </div>

          <div class="bar-slots">
            {#each { length: Math.min(maxLane + 1, MAX_LANES) } as _, lane}
              {@const seg = daySegs.get(lane)}
              {#if seg}
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <div
                  class="bar-segment"
                  class:is-trip-start={seg.isTripStart}
                  class:is-trip-end={seg.isTripEnd}
                  class:is-trip={seg.tripLane.tripId !== null}
                  class:has-activity-start={seg.hasActivityStart && seg.tripLane.tripId !== null}
                  style="background: {seg.tripLane.color};"
                  onclick={(e) => handleBarClick(seg.tripLane, e)}
                  onmouseenter={(e) => handleBarEnter(seg.tripLane, e)}
                  onmousemove={handleBarHover}
                  onmouseleave={handleBarLeave}
                >
                  {#if seg.isTripStart && seg.tripLane.tripId}
                    <span class="bar-label trip-label">{seg.tripLane.tripName}</span>
                  {:else if seg.activityLabel}
                    <span class="bar-label">{seg.activityLabel}</span>
                  {/if}
                </div>
              {:else}
                <div class="bar-slot-empty"></div>
              {/if}
            {/each}

            <!-- Ghost bar during create drag -->
            {#if dragMode === 'create' && isInCreateRange(dateStr)}
              <div
                class="bar-segment ghost"
                class:is-start={createRange && dateStr === createRange.start}
                class:is-end={createRange && dateStr === createRange.end}
              ></div>
            {/if}

            <!-- Ghost bar from modal -->
            {#if !dragMode && ghostDates && dateStr >= ghostDates.startDate && dateStr <= ghostDates.endDate}
              <div
                class="bar-segment ghost modal-ghost"
                class:is-start={dateStr === ghostDates.startDate}
                class:is-end={dateStr === ghostDates.endDate}
                style="background: {ACTIVITY_COLORS[ghostDates.type]}; opacity: 0.35;"
              ></div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/each}
</div>

<Tooltip activity={tooltipActivity} x={tooltipX} y={tooltipY} />

{#if popupTripLane}
  <TripDetailPopup
    tripName={popupTripLane.tripName}
    color={popupTripLane.color}
    activities={popupTripLane.activities}
    x={popupX}
    y={popupY}
    {onedit}
    onclose={() => popupTripLane = null}
  />
{/if}

<style>
  .month-view {
    overflow-y: auto;
    max-height: calc(100vh - 180px);
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    background: white;
    user-select: none;
  }

  .week-header {
    display: grid;
    grid-template-columns: 56px repeat(7, 1fr);
    position: sticky;
    top: 0;
    z-index: 10;
    background: white;
    border-bottom: 2px solid #e5e7eb;
  }

  .day-header {
    text-align: center;
    font-size: 0.7rem;
    font-weight: 600;
    color: #999;
    padding: 0.35rem 0;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .week-row {
    display: grid;
    grid-template-columns: 56px repeat(7, 1fr);
    border-bottom: 1px solid #eee;
  }

  .label-col {
    display: flex;
    align-items: center;
    justify-content: center;
    border-right: 1px solid #eee;
    background: #fafafa;
  }

  .month-label {
    font-size: 0.85rem;
    font-weight: 800;
    color: #444;
    white-space: nowrap;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    writing-mode: vertical-lr;
    transform: rotate(180deg);
  }

  .day-cell {
    display: flex;
    flex-direction: column;
    cursor: crosshair;
    border-right: 1px solid #f0f0f0;
    min-height: 90px;
    padding: 3px 0;
    overflow: hidden;
  }

  .day-cell:nth-child(9) { border-right: none; }

  .day-cell.alt-month { background: rgba(0, 0, 0, 0.03); }
  .day-cell.today { background: rgba(59, 130, 246, 0.06); }
  .day-cell.drag-selected { background: rgba(59, 130, 246, 0.15) !important; }
  .day-cell.conflict { background-color: rgba(239, 68, 68, 0.08); }
  .day-cell.conflict.alt-month { background-color: rgba(239, 68, 68, 0.11); }
  .day-cell:hover { background: rgba(59, 130, 246, 0.07) !important; }

  .day-number-row {
    text-align: right;
    line-height: 1;
    padding: 0 4px 2px;
  }

  .day-number { font-size: 0.75rem; color: #888; }

  .today-number {
    background: #3b82f6;
    color: white;
    border-radius: 50%;
    width: 20px;
    height: 20px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    font-size: 0.7rem;
  }

  .bar-slots {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-height: 0;
  }

  .bar-segment {
    height: 18px;
    display: flex;
    align-items: center;
    padding: 0 4px;
    overflow: hidden;
    opacity: 0.9;
    margin: 0 -1px;
    position: relative;
    z-index: 1;
    cursor: pointer;
  }

  .bar-segment:hover {
    opacity: 1;
    filter: brightness(1.1);
    z-index: 2;
  }

  .bar-segment.is-trip-start {
    border-top-left-radius: 4px;
    border-bottom-left-radius: 4px;
    margin-left: 3px;
  }

  .bar-segment.is-trip-end {
    border-top-right-radius: 4px;
    border-bottom-right-radius: 4px;
    margin-right: 3px;
  }

  .bar-segment.is-trip {
    height: 26px;
  }

  .bar-segment.has-activity-start {
    border-left: 2px solid rgba(255, 255, 255, 0.4);
  }

  .bar-segment.is-trip-start.has-activity-start {
    border-left: none;
  }

  .bar-segment.ghost {
    background: rgba(0, 0, 0, 0.18);
    border: 1px dashed rgba(0, 0, 0, 0.3);
    cursor: crosshair;
  }

  .bar-segment.modal-ghost {
    border: 1px dashed rgba(255, 255, 255, 0.5);
  }

  .bar-slot-empty { height: 18px; }

  .bar-label {
    font-size: 0.6rem;
    color: white;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    padding-left: 2px;
  }

  .is-trip .bar-label {
    font-size: 0.55rem;
  }

  .trip-label {
    font-weight: 700;
    font-size: 0.6rem;
  }
</style>
