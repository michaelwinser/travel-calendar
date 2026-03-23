<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { ACTIVITY_COLORS, type Activity } from '../lib/api';
  import {
    today, addDays, stringToDate,
    getWeeksForRange, getDaySegmentsForWeek,
    monthLabel, monthIndex, hasConflict,
    minDate, maxDate,
    type DayBarSegment,
  } from '../lib/date-utils';

  interface Props {
    activities: Activity[];
    onedit: (activity: Activity) => void;
    ondayclick: (date: string) => void;
    ondragselect: (startDate: string, endDate: string) => void;
  }

  let { activities, onedit, ondayclick, ondragselect }: Props = $props();

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
  }

  function handleDayMouseDown(dateStr: string, e: MouseEvent) {
    if (e.button !== 0) return;
    dragStartDate = dateStr;
    dragCurrentDate = dateStr;
    isDragging = true;
    e.preventDefault();
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

  function handleSegmentClick(activity: Activity, e: MouseEvent) {
    e.stopPropagation();
    onedit(activity);
  }

  function handleMoreClick(dateStr: string, e: MouseEvent) {
    e.stopPropagation();
    ondayclick(dateStr);
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
          class:drag-selected={isInDragRange(dateStr)}
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
              {#if seg}
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <div
                  class="bar-segment"
                  class:is-start={seg.isStart}
                  class:is-end={seg.isEnd}
                  style="background: {seg.color};"
                  title="{seg.activity.title}{seg.activity.location ? ' — ' + seg.activity.location : ''}"
                  onclick={(e) => handleSegmentClick(seg.activity, e)}
                >
                  {#if seg.isStart}
                    <span class="bar-label">{seg.activity.title}</span>
                  {/if}
                </div>
              {:else}
                <div class="bar-slot-empty"></div>
              {/if}
            {/each}

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
