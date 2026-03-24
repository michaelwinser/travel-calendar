<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { ACTIVITY_COLORS, type Activity, type ActivityType } from '../lib/api';
  import Tooltip from './Tooltip.svelte';
  import {
    today, addDays, stringToDate, addDays as addD,
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
    onmove?: (activityId: string, startDate: string, endDate: string) => void;
    onfocusdate?: (date: string) => void;
  }

  let { activities, ghostDates, initialDate, onedit, ondayclick, ondragselect, onresize, onmove, onfocusdate }: Props = $props();

  const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
  const MAX_BARS = 3;
  const EDGE_ZONE = 6; // pixels from bar edge for resize cursor

  // View range
  let rangeStart = $state(addDays(today(), -90));
  let rangeEnd = $state(addDays(today(), 180));

  let weeks = $derived(getWeeksForRange(rangeStart, rangeEnd));

  let weekData = $derived(
    weeks.map(week => {
      const { segments, overflowPerDay } = getDaySegmentsForWeek(activities, week, ACTIVITY_COLORS, MAX_BARS);
      const conflicts = week.days.map(d => hasConflict(d, activities));
      return { week, segments, overflowPerDay, conflicts };
    })
  );

  let weekMonthLabels = $derived(
    weeks.map((week, i) => {
      const thursdayMonth = monthLabel(week.days[4]);
      if (i === 0) return thursdayMonth;
      const prevThursdayMonth = monthLabel(weeks[i - 1].days[4]);
      return thursdayMonth !== prevThursdayMonth ? thursdayMonth : '';
    })
  );

  // === DRAG STATE ===
  type DragMode = 'create' | 'move' | 'resize' | null;
  let dragMode = $state<DragMode>(null);
  let didMove = $state(false); // tracks if mouse moved during drag (click vs drag)
  let dragActivity = $state<Activity | null>(null); // the activity being moved/resized

  // Create drag
  let createStart = $state<string | null>(null);
  let createCurrent = $state<string | null>(null);

  let createRange = $derived(
    createStart && createCurrent
      ? { start: minDate(createStart, createCurrent), end: maxDate(createStart, createCurrent) }
      : null
  );

  // Move drag
  let moveAnchorDate = $state<string | null>(null);
  let moveCurrentDate = $state<string | null>(null);
  let dragLane = $state(0); // shared lane for move and resize

  let movePreview = $derived.by(() => {
    if (dragMode !== 'move' || !dragActivity || !moveAnchorDate || !moveCurrentDate) return null;
    const daysDelta = dateDiff(moveAnchorDate, moveCurrentDate);
    const newStart = addDays(dragActivity.startDate, daysDelta);
    const newEnd = addDays(dragActivity.endDate, daysDelta);
    return { start: newStart, end: newEnd, activityId: dragActivity.id, type: dragActivity.type, title: dragActivity.title };
  });

  // Resize drag
  let resizeEdge = $state<'start' | 'end'>('end');
  let resizeCurrent = $state<string | null>(null);

  let resizePreview = $derived.by(() => {
    if (dragMode !== 'resize' || !dragActivity || !resizeCurrent) return null;
    let start = dragActivity.startDate;
    let end = dragActivity.endDate;
    if (resizeEdge === 'start') {
      start = resizeCurrent <= end ? resizeCurrent : end;
    } else {
      end = resizeCurrent >= start ? resizeCurrent : start;
    }
    return { start, end, activityId: dragActivity.id, type: dragActivity.type, title: dragActivity.title };
  });

  // The active drag preview (whichever mode)
  let activePreview = $derived(dragMode === 'move' ? movePreview : dragMode === 'resize' ? resizePreview : null);
  let activeDragId = $derived(activePreview?.activityId ?? null);

  function dateDiff(from: string, to: string): number {
    const f = stringToDate(from);
    const t = stringToDate(to);
    return Math.round((t.getTime() - f.getTime()) / (86400000));
  }

  function isInCreateRange(dateStr: string): boolean {
    if (!createRange) return false;
    return dateStr >= createRange.start && dateStr <= createRange.end;
  }

  function isInActivePreview(dateStr: string): boolean {
    if (!activePreview) return false;
    return dateStr >= activePreview.start && dateStr <= activePreview.end;
  }

  // === SCROLL ===
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

  // === DAY CELL MOUSEDOWN (create mode only) ===
  function handleDayMouseDown(dateStr: string, e: MouseEvent) {
    if (e.button !== 0) return;
    // Only start create if clicking on the day-number area or empty space
    const target = e.target as HTMLElement;
    if (target.closest('.bar-segment')) return; // click was on a bar
    e.preventDefault();
    dragMode = 'create';
    createStart = dateStr;
    createCurrent = dateStr;
  }

  // === DAY CELL MOUSEENTER ===
  function handleDayMouseEnter(dateStr: string) {
    if (!dragMode) return;
    if (dragMode === 'create') {
      if (dateStr !== createStart) didMove = true;
      createCurrent = dateStr;
    } else if (dragMode === 'move') {
      if (dateStr !== moveAnchorDate) didMove = true;
      moveCurrentDate = dateStr;
    } else if (dragMode === 'resize') {
      didMove = true;
      resizeCurrent = dateStr;
    }
  }

  // === GLOBAL MOUSEUP ===
  function handleMouseUp() {
    const mode = dragMode;
    const moved = didMove;

    if (mode === 'create') {
      if (createStart && createCurrent) {
        const start = minDate(createStart, createCurrent);
        const end = maxDate(createStart, createCurrent);
        if (start === end) ondayclick(start);
        else ondragselect(start, end);
      }
    } else if (mode === 'move') {
      if (!moved && dragActivity) {
        // No movement — treat as click → edit
        tooltipActivity = null;
        onedit(dragActivity);
      } else if (moved && movePreview && dragActivity && onmove) {
        onmove(dragActivity.id, movePreview.start, movePreview.end);
      }
    } else if (mode === 'resize') {
      if (!moved && dragActivity) {
        // No movement — treat as click → edit
        tooltipActivity = null;
        onedit(dragActivity);
      } else if (moved && resizePreview && dragActivity && onresize) {
        if (resizePreview.start !== dragActivity.startDate || resizePreview.end !== dragActivity.endDate) {
          onresize(dragActivity.id, resizePreview.start, resizePreview.end);
        }
      }
    }
    clearDrag();
  }

  function clearDrag() {
    dragMode = null;
    didMove = false;
    dragActivity = null;
    createStart = null; createCurrent = null;
    moveAnchorDate = null; moveCurrentDate = null;
    resizeCurrent = null;
  }

  // === BAR INTERACTIONS ===
  function handleBarMouseDown(activity: Activity, seg: DayBarSegment, dateStr: string, lane: number, e: MouseEvent) {
    if (e.button !== 0) return;
    e.stopPropagation();
    e.preventDefault();

    dragActivity = activity;
    dragLane = lane;
    didMove = false;

    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const fromLeft = e.clientX - rect.left;
    const fromRight = rect.right - e.clientX;

    if (seg.isStart && fromLeft < EDGE_ZONE) {
      dragMode = 'resize';
      resizeEdge = 'start';
      resizeCurrent = dateStr;
    } else if (seg.isEnd && fromRight < EDGE_ZONE) {
      dragMode = 'resize';
      resizeEdge = 'end';
      resizeCurrent = dateStr;
    } else {
      dragMode = 'move';
      moveAnchorDate = dateStr;
      moveCurrentDate = dateStr;
    }
  }

  function handleBarMouseMove(seg: DayBarSegment, e: MouseEvent) {
    if (dragMode) return;
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const fromLeft = e.clientX - rect.left;
    const fromRight = rect.right - e.clientX;
    const el = e.currentTarget as HTMLElement;

    if ((seg.isStart && fromLeft < EDGE_ZONE) || (seg.isEnd && fromRight < EDGE_ZONE)) {
      el.style.cursor = 'col-resize';
    } else {
      el.style.cursor = 'grab';
    }
  }

  function handleMoreClick(dateStr: string, e: MouseEvent) {
    e.stopPropagation();
    ondayclick(dateStr);
  }

  // === TOOLTIP ===
  let tooltipActivity = $state<Activity | null>(null);
  let tooltipX = $state(0);
  let tooltipY = $state(0);

  function handleBarEnter(activity: Activity, e: MouseEvent) {
    if (dragMode) return;
    tooltipActivity = activity;
    tooltipX = e.clientX;
    tooltipY = e.clientY;
  }

  function handleBarHover(e: MouseEvent) {
    if (dragMode) { tooltipActivity = null; return; }
    tooltipX = e.clientX;
    tooltipY = e.clientY;
  }

  function handleBarLeave() { tooltipActivity = null; }

  // === HELPERS ===
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

  {#each weekData as { week, segments, overflowPerDay, conflicts }, wi (week.weekStart)}
    {@const label = weekMonthLabels[wi]}

    <div class="week-row">
      <div class="label-col">
        {#if label}<span class="month-label">{label}</span>{/if}
      </div>

      {#each week.days as dateStr, di}
        {@const altMonth = monthIndex(dateStr) % 2 === 1}
        {@const daySegments = segments.get(di) ?? []}
        {@const overflow = overflowPerDay[di]}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="day-cell"
          class:alt-month={altMonth}
          class:today={isToday(dateStr)}
          class:drag-selected={dragMode === 'create' && isInCreateRange(dateStr)}
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

          <div class="bar-slots">
            {#each { length: MAX_BARS } as _, lane}
              {@const seg = daySegments.find(s => s.lane === lane)}
              {@const isActiveDrag = seg && seg.activity.id === activeDragId}
              {@const showPreviewHere = activePreview && lane === dragLane && isInActivePreview(dateStr)}
              {#if showPreviewHere}
                <!-- Drag preview in the original lane -->
                <div
                  class="bar-segment active-drag"
                  class:is-start={dateStr === activePreview?.start}
                  class:is-end={dateStr === activePreview?.end}
                  style="background: {ACTIVITY_COLORS[activePreview?.type ?? 'stay']};"
                >
                  {#if dateStr === activePreview?.start}
                    <span class="bar-label">{activePreview?.title}</span>
                  {/if}
                </div>
              {:else if isActiveDrag}
                <!-- Vacated slot (original position of dragged activity) -->
                <div class="bar-slot-empty"></div>
              {:else if seg}
                <!-- Normal bar segment -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <div
                  class="bar-segment"
                  class:is-start={seg.isStart}
                  class:is-end={seg.isEnd}
                  style="background: {seg.color};"
                  onmousedown={(e) => handleBarMouseDown(seg.activity, seg, dateStr, lane, e)}
                  onmouseenter={(e) => handleBarEnter(seg.activity, e)}
                  onmousemove={(e) => { handleBarMouseMove(seg, e); handleBarHover(e); }}
                  onmouseleave={handleBarLeave}
                >
                  {#if seg.isStart}
                    <span class="bar-label">{seg.activity.title}</span>
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
            {#if !dragMode && ghostDates && dateStr >= ghostDates.startDate && dateStr <= ghostDates.endDate && daySegments.length < MAX_BARS}
              <div
                class="bar-segment ghost modal-ghost"
                class:is-start={dateStr === ghostDates.startDate}
                class:is-end={dateStr === ghostDates.endDate}
                style="background: {ACTIVITY_COLORS[ghostDates.type]}; opacity: 0.35;"
              ></div>
            {/if}

            {#if overflow > 1}
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
    cursor: grab;
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

  .bar-segment.active-drag {
    opacity: 0.8;
    z-index: 5;
    outline: 2px dashed rgba(255, 255, 255, 0.9);
    outline-offset: -1px;
    cursor: grabbing;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
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

  .more-link:hover { color: #333; }
</style>
