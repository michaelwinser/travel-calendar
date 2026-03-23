<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { ACTIVITY_COLORS, type Activity, type ActivityType } from '../lib/api';
  import Tooltip from './Tooltip.svelte';
  import {
    today, addDays, stringToDate,
    getWeeksForRange, getDaySegmentsForWeek,
    monthLabel, monthIndex, hasConflict,
    minDate, maxDate,
    type DayBarSegment,
  } from '../lib/date-utils';

  interface Props {
    activities: Activity[];
    ghostDates?: { startDate: string; endDate: string; type: ActivityType } | null;
    initialDate?: string;
    onedit: (activity: Activity) => void;
    ondayclick: (date: string) => void;
    ondragselect: (startDate: string, endDate: string) => void;
    onresize?: (activityId: string, startDate: string, endDate: string) => void;
    onfocusdate?: (date: string) => void;
  }

  let { activities, ghostDates, initialDate, onedit, ondayclick, ondragselect, onresize, onfocusdate }: Props = $props();

  const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
  const MAX_BARS = 3;

  // View range: 3 months back, 6 months forward
  let rangeStart = $state(addDays(today(), -90));
  let rangeEnd = $state(addDays(today(), 180));

  let weeks = $derived(getWeeksForRange(rangeStart, rangeEnd));

  // Precompute per-day data for each week
  let weekData = $derived(
    weeks.map(week => {
      const { segments, overflowPerDay } = getDaySegmentsForWeek(activities, week, ACTIVITY_COLORS, MAX_BARS);
      const conflicts = week.days.map(d => hasConflict(d, activities));
      return { week, segments, overflowPerDay, conflicts };
    })
  );

  // Month labels: show on first week of each month
  let weekMonthLabels = $derived(
    weeks.map((week, i) => {
      const thursdayMonth = monthLabel(week.days[4]);
      if (i === 0) return thursdayMonth;
      const prevThursdayMonth = monthLabel(weeks[i - 1].days[4]);
      return thursdayMonth !== prevThursdayMonth ? thursdayMonth : '';
    })
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

  // Scroll container
  let scrollEl: HTMLElement;

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
    // Date outside current range — recenter and retry
    rangeStart = addDays(dateStr, -90);
    rangeEnd = addDays(dateStr, 180);
    tick().then(() => {
      const el2 = scrollEl?.querySelector(`[data-date="${dateStr}"]`);
      if (el2) el2.scrollIntoView({ block: 'center' });
    });
  }

  export function scrollToToday() {
    scrollToDate(today());
  }

  let scrollDebounce: ReturnType<typeof setTimeout> | undefined;

  function handleScroll() {
    if (!scrollEl) return;
    const { scrollTop, scrollHeight, clientHeight } = scrollEl;

    if (scrollTop < 200) {
      const prevHeight = scrollEl.scrollHeight;
      rangeStart = addDays(rangeStart, -90);
      tick().then(() => {
        const newHeight = scrollEl.scrollHeight;
        scrollEl.scrollTop += newHeight - prevHeight;
      });
    }

    if (scrollHeight - scrollTop - clientHeight < 200) {
      rangeEnd = addDays(rangeEnd, 90);
    }

    // Report top-visible date (debounced)
    if (onfocusdate) {
      clearTimeout(scrollDebounce);
      scrollDebounce = setTimeout(() => reportTopDate(), 500);
    }
  }

  function reportTopDate() {
    if (!scrollEl || !onfocusdate) return;
    const containerTop = scrollEl.getBoundingClientRect().top;
    const dateCells = scrollEl.querySelectorAll('[data-date]');
    for (const el of dateCells) {
      if (el.getBoundingClientRect().top >= containerTop) {
        const date = el.getAttribute('data-date');
        if (date) { onfocusdate(date); return; }
      }
    }
  }

  function handleDayMouseDown(dateStr: string, e: MouseEvent) {
    if (e.button !== 0) return;
    dragStartDate = dateStr;
    dragCurrentDate = dateStr;
    isDragging = true;
    e.preventDefault();
  }

  function handleDayMouseEnter(dateStr: string) {
    if (resizing) {
      handleResizeMove(dateStr);
    } else if (isDragging) {
      dragCurrentDate = dateStr;
    }
  }

  function handleMouseUp() {
    if (resizing) {
      handleResizeUp();
      return;
    }

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

  // Resize state
  let resizing = $state<{ activity: Activity; edge: 'start' | 'end'; originalStart: string; originalEnd: string } | null>(null);
  let resizeCurrentDate = $state<string | null>(null);

  function handleSegmentMouseDown(activity: Activity, seg: DayBarSegment, dateStr: string, e: MouseEvent) {
    if (e.button !== 0 || !onresize) return;
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const edgeZone = 6;

    if (seg.isStart && e.clientX - rect.left < edgeZone) {
      e.stopPropagation();
      e.preventDefault();
      resizing = { activity, edge: 'start', originalStart: activity.startDate, originalEnd: activity.endDate };
      resizeCurrentDate = dateStr;
    } else if (seg.isEnd && rect.right - e.clientX < edgeZone) {
      e.stopPropagation();
      e.preventDefault();
      resizing = { activity, edge: 'end', originalStart: activity.startDate, originalEnd: activity.endDate };
      resizeCurrentDate = dateStr;
    }
  }

  // Computed resize preview range
  let resizePreview = $derived.by(() => {
    if (!resizing || !resizeCurrentDate) return null;
    let start = resizing.originalStart;
    let end = resizing.originalEnd;
    if (resizing.edge === 'start') {
      start = resizeCurrentDate <= end ? resizeCurrentDate : end;
    } else {
      end = resizeCurrentDate >= start ? resizeCurrentDate : start;
    }
    return { start, end, type: resizing.activity.type, activityId: resizing.activity.id };
  });

  function isInResizeRange(dateStr: string): boolean {
    if (!resizePreview) return false;
    return dateStr >= resizePreview.start && dateStr <= resizePreview.end;
  }

  function handleResizeMove(dateStr: string) {
    if (resizing) {
      resizeCurrentDate = dateStr;
    }
  }

  function handleResizeUp() {
    if (!resizing || !resizeCurrentDate || !onresize) {
      resizing = null;
      resizeCurrentDate = null;
      return;
    }

    let newStart = resizing.originalStart;
    let newEnd = resizing.originalEnd;

    if (resizing.edge === 'start') {
      newStart = resizeCurrentDate <= resizing.originalEnd ? resizeCurrentDate : resizing.originalEnd;
    } else {
      newEnd = resizeCurrentDate >= resizing.originalStart ? resizeCurrentDate : resizing.originalStart;
    }

    if (newStart !== resizing.originalStart || newEnd !== resizing.originalEnd) {
      onresize(resizing.activity.id, newStart, newEnd);
    }

    resizing = null;
    resizeCurrentDate = null;
  }

  function getSegmentCursor(seg: DayBarSegment): string {
    if (!onresize) return 'pointer';
    if (seg.isStart || seg.isEnd) return 'col-resize';
    return 'pointer';
  }

  function handleSegmentClick(activity: Activity, e: MouseEvent) {
    if (resizing) return; // Don't open edit if we just finished resizing
    e.stopPropagation();
    tooltipActivity = null;
    onedit(activity);
  }

  function handleMoreClick(dateStr: string, e: MouseEvent) {
    e.stopPropagation();
    ondayclick(dateStr);
  }

  // Tooltip state
  let tooltipActivity = $state<Activity | null>(null);
  let tooltipX = $state(0);
  let tooltipY = $state(0);

  function handleBarEnter(activity: Activity, e: MouseEvent) {
    tooltipActivity = activity;
    tooltipX = e.clientX;
    tooltipY = e.clientY;
  }

  function handleBarMove(e: MouseEvent) {
    tooltipX = e.clientX;
    tooltipY = e.clientY;
  }

  function handleBarLeave() {
    tooltipActivity = null;
  }

  function dayOfMonth(dateStr: string): number {
    return stringToDate(dateStr).getDate();
  }

  function isToday(dateStr: string): boolean {
    return dateStr === today();
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="month-view"
  bind:this={scrollEl}
  onscroll={handleScroll}
  onmouseup={handleMouseUp}
  onmouseleave={handleMouseUp}
>
  <!-- Day-of-week header -->
  <div class="week-header">
    <div class="label-col"></div>
    {#each DAY_NAMES as name}
      <div class="day-header">{name}</div>
    {/each}
  </div>

  <!-- Week rows -->
  {#each weekData as { week, segments, overflowPerDay, conflicts }, wi (week.weekStart)}
    {@const label = weekMonthLabels[wi]}

    <div class="week-row">
      <!-- Month label -->
      <div class="label-col">
        {#if label}
          <span class="month-label">{label}</span>
        {/if}
      </div>

      <!-- Day cells -->
      {#each week.days as dateStr, di}
        {@const altMonth = monthIndex(dateStr) % 2 === 1}
        {@const daySegments = segments.get(di) ?? []}
        {@const overflow = overflowPerDay[di]}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="day-cell"
          class:alt-month={altMonth}
          class:today={isToday(dateStr)}
          class:drag-selected={!resizing && isInDragRange(dateStr)}
          class:conflict={conflicts[di]}
          data-date={dateStr}
          onmousedown={(e) => handleDayMouseDown(dateStr, e)}
          onmouseenter={() => handleDayMouseEnter(dateStr)}
        >
          <div class="day-number-row">
            <span class="day-number" class:today-number={isToday(dateStr)}>
              {dayOfMonth(dateStr)}
            </span>
          </div>

          <!-- Activity bar segments -->
          <div class="bar-slots">
            {#each { length: MAX_BARS } as _, lane}
              {@const seg = daySegments.find(s => s.lane === lane)}
              {@const isResizingThis = seg && resizePreview && seg.activity.id === resizePreview.activityId}
              {#if seg && !isResizingThis}
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <div
                  class="bar-segment"
                  class:is-start={seg.isStart}
                  class:is-end={seg.isEnd}
                  style="background: {seg.color}; cursor: {getSegmentCursor(seg)};"
                  onclick={(e) => handleSegmentClick(seg.activity, e)}
                  onmousedown={(e) => handleSegmentMouseDown(seg.activity, seg, dateStr, e)}
                  onmouseenter={(e) => handleBarEnter(seg.activity, e)}
                  onmousemove={handleBarMove}
                  onmouseleave={handleBarLeave}
                >
                  {#if seg.isStart}
                    <span class="bar-label">{seg.activity.title}</span>
                  {/if}
                </div>
              {:else if isResizingThis && isInResizeRange(dateStr)}
                <div
                  class="bar-segment resize-preview"
                  class:is-start={dateStr === resizePreview?.start}
                  class:is-end={dateStr === resizePreview?.end}
                  style="background: {seg?.color};"
                >
                  {#if dateStr === resizePreview?.start}
                    <span class="bar-label">{seg?.activity.title}</span>
                  {/if}
                </div>
              {:else if isResizingThis}
                <div class="bar-slot-empty"></div>
              {:else}
                <div class="bar-slot-empty"></div>
              {/if}
            {/each}

            <!-- Resize preview for expanded days -->
            {#if resizePreview && isInResizeRange(dateStr) && !daySegments.some(s => s.activity.id === resizePreview.activityId)}
              <div
                class="bar-segment resize-preview"
                class:is-start={dateStr === resizePreview.start}
                class:is-end={dateStr === resizePreview.end}
                style="background: {ACTIVITY_COLORS[resizePreview.type]};"
              >
                {#if dateStr === resizePreview.start}
                  <span class="bar-label">{resizing?.activity.title}</span>
                {/if}
              </div>
            {/if}

            <!-- Ghost bar during drag or modal edit -->
            {#if !resizing && isDragging && isInDragRange(dateStr) && daySegments.length < MAX_BARS}
              {@const isGhostStart = dragRange && dateStr === dragRange.start}
              {@const isGhostEnd = dragRange && dateStr === dragRange.end}
              <div
                class="bar-segment ghost"
                class:is-start={isGhostStart}
                class:is-end={isGhostEnd}
              ></div>
            {:else if ghostDates && dateStr >= ghostDates.startDate && dateStr <= ghostDates.endDate && daySegments.length < MAX_BARS}
              {@const isGhostStart = dateStr === ghostDates.startDate}
              {@const isGhostEnd = dateStr === ghostDates.endDate}
              <div
                class="bar-segment ghost modal-ghost"
                class:is-start={isGhostStart}
                class:is-end={isGhostEnd}
                style="background: {ACTIVITY_COLORS[ghostDates.type]}; opacity: 0.35;"
              ></div>
            {/if}

            {#if overflow > 0}
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <span class="more-link" onclick={(e) => handleMoreClick(dateStr, e)}>
                +{overflow} more
              </span>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/each}
</div>

<Tooltip activity={tooltipActivity} x={tooltipX} y={tooltipY} />

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
    cursor: pointer;
    border-right: 1px solid #f0f0f0;
    min-height: 90px;
    padding: 3px 0;
    overflow: hidden;
  }

  .day-cell:nth-child(9) {
    border-right: none;
  }

  .day-cell.alt-month {
    background: rgba(0, 0, 0, 0.03);
  }

  .day-cell.today {
    background: rgba(59, 130, 246, 0.06);
  }

  .day-cell.drag-selected {
    background: rgba(59, 130, 246, 0.15) !important;
  }

  .day-cell.conflict {
    background-color: rgba(239, 68, 68, 0.08);
  }

  .day-cell.conflict.alt-month {
    background-color: rgba(239, 68, 68, 0.11);
  }

  .day-cell:hover {
    background: rgba(59, 130, 246, 0.07) !important;
  }

  .day-number-row {
    text-align: right;
    line-height: 1;
    padding: 0 4px 2px;
  }

  .day-number {
    font-size: 0.75rem;
    color: #888;
  }

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
    cursor: pointer;
    overflow: hidden;
    opacity: 0.9;
    /* Extend into neighboring cells to hide the border gap */
    margin: 0 -1px;
    position: relative;
    z-index: 1;
  }

  .bar-segment:hover {
    opacity: 1;
    filter: brightness(1.1);
    z-index: 2;
  }

  .bar-segment.is-start {
    border-top-left-radius: 4px;
    border-bottom-left-radius: 4px;
    margin-left: 3px;
  }

  .bar-segment.is-end {
    border-top-right-radius: 4px;
    border-bottom-right-radius: 4px;
    margin-right: 3px;
  }

  .bar-segment.ghost {
    background: rgba(0, 0, 0, 0.18);
    border: 1px dashed rgba(0, 0, 0, 0.3);
  }

  .bar-segment.modal-ghost {
    border: 1px dashed rgba(255, 255, 255, 0.5);
  }

  .bar-segment.resize-preview {
    opacity: 0.6;
    border: 1px dashed rgba(255, 255, 255, 0.5);
  }

  .bar-slot-empty {
    height: 18px;
  }

  .bar-label {
    font-size: 0.65rem;
    color: white;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .more-link {
    font-size: 0.6rem;
    color: #888;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1.2;
  }

  .more-link:hover {
    color: #333;
  }
</style>
