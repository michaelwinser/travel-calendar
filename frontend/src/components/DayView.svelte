<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { ACTIVITY_COLORS, type Activity, type TripSummary, type OverlayCalendar } from '../lib/api';
  import {
    today, addDays, stringToDate, hasConflict,
    monthLabel, minDate, maxDate,
  } from '../lib/date-utils';

  interface Props {
    activities: Activity[];
    trips: TripSummary[];
    overlayActivities?: Activity[];
    overlayCalendars?: OverlayCalendar[];
    initialDate?: string;
    onedit: (activity: Activity) => void;
    ondayclick: (date: string) => void;
    ondragselect: (startDate: string, endDate: string) => void;
    onfocusdate?: (date: string) => void;
  }

  let { activities, trips, overlayActivities, overlayCalendars, initialDate, onedit, ondayclick, ondragselect, onfocusdate }: Props = $props();

  // Build per-day overlay lookup
  let overlayByDay = $derived.by(() => {
    const map = new Map<string, { location: string; color: string; ownerEmail: string }[]>();
    if (!overlayActivities?.length || !overlayCalendars?.length) return map;
    const colorMap = new Map<string, string>();
    for (const c of overlayCalendars) if (c.visible) colorMap.set(c.email, c.color);
    for (const a of overlayActivities) {
      const color = colorMap.get(a.userId) ?? '#999';
      const start = stringToDate(a.startDate);
      const end = stringToDate(a.endDate);
      for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
        const ds = d.toISOString().slice(0, 10);
        const item = { location: a.location ?? '', color, ownerEmail: a.userId };
        const list = map.get(ds);
        if (list) list.push(item); else map.set(ds, [item]);
      }
    }
    return map;
  });

  // Trip lookup
  function tripFor(activity: Activity): TripSummary | undefined {
    if (!activity.tripId) return undefined;
    return trips.find(t => t.id === activity.tripId);
  }

  // Find the trip(s) active on a given date
  function tripsOnDate(dateStr: string, dayActs: Activity[]): TripSummary[] {
    const tripIds = new Set<string>();
    const result: TripSummary[] = [];
    for (const a of dayActs) {
      if (a.tripId && !tripIds.has(a.tripId)) {
        tripIds.add(a.tripId);
        const t = trips.find(tr => tr.id === a.tripId);
        if (t) result.push(t);
      }
    }
    return result;
  }

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
    scrollToDate(initialDate ?? today());
  });

  export function scrollToDate(dateStr: string) {
    const el = scrollEl?.querySelector(`[data-date="${dateStr}"]`);
    if (el) {
      el.scrollIntoView({ block: 'center', behavior: 'smooth' });
      return;
    }
    rangeStart = addDays(dateStr, -60);
    rangeEnd = addDays(dateStr, 120);
    tick().then(() => {
      const el2 = scrollEl?.querySelector(`[data-date="${dateStr}"]`);
      if (el2) el2.scrollIntoView({ block: 'center' });
    });
  }

  export function scrollToToday() {
    scrollToDate(today());
  }

  export function scrollAction(action: 'pageDown' | 'pageUp' | 'nextActivity' | 'prevActivity') {
    if (!scrollEl) return;
    const { clientHeight } = scrollEl;

    if (action === 'pageDown') {
      scrollEl.scrollBy({ top: clientHeight * 0.8, behavior: 'smooth' });
    } else if (action === 'pageUp') {
      scrollEl.scrollBy({ top: -clientHeight * 0.8, behavior: 'smooth' });
    } else {
      const chips = scrollEl.querySelectorAll('.activity-chip');
      if (!chips.length) return;
      const containerTop = scrollEl.getBoundingClientRect().top;
      const center = containerTop + clientHeight / 2;

      if (action === 'nextActivity') {
        for (const chip of chips) {
          if (chip.getBoundingClientRect().top > center + 10) {
            chip.scrollIntoView({ block: 'center', behavior: 'smooth' });
            return;
          }
        }
      } else {
        const arr = Array.from(chips);
        for (let i = arr.length - 1; i >= 0; i--) {
          if (arr[i].getBoundingClientRect().bottom < center - 10) {
            arr[i].scrollIntoView({ block: 'center', behavior: 'smooth' });
            return;
          }
        }
      }
    }
  }

  let scrollDebounce: ReturnType<typeof setTimeout> | undefined;

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
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="day-view"
  bind:this={scrollEl}
  onscroll={handleScroll}
  onmouseup={handleMouseUp}
  onmouseleave={handleMouseUp}
>
  {#each dates as dateStr (dateStr)}
    {@const info = formatDate(dateStr)}
    {@const dayActs = dateActivities.get(dateStr) ?? []}
    {@const conflict = dayActs.length > 1 && hasConflict(dateStr, activities)}
    {@const label = dateMonthLabels.get(dateStr)}
    {@const dayTrips = tripsOnDate(dateStr, dayActs)}

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
      class:has-trip={dayTrips.length > 0}
      class:drag-selected={isInDragRange(dateStr)}
      style={dayTrips.length === 1 ? `border-left: 4px solid ${dayTrips[0].color};` : ''}
      data-date={dateStr}
      onmousedown={(e) => handleDayMouseDown(dateStr, e)}
      onmouseenter={() => handleDayMouseEnter(dateStr)}
    >
      <div class="date-col">
        <span class="dow">{info.dow}</span>
        <span class="day-num" class:today-num={isToday(dateStr)}>{info.day}</span>
      </div>

      <div class="activities-col">
        {#if dayTrips.length > 0}
          <div class="trip-labels">
            {#each dayTrips as trip}
              <span class="trip-badge" style="color: {trip.color};">{trip.name}</span>
            {/each}
          </div>
        {/if}

        {#if dayActs.length === 0}
          <span class="home-label">Home</span>
        {:else}
          {#each dayActs as activity (activity.id)}
            {@const trip = tripFor(activity)}
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <div
              class="activity-chip"
              style={trip ? `border-left: 3px solid ${trip.color};` : ''}
              onclick={(e) => { e.stopPropagation(); onedit(activity); }}
            >
              <span class="chip-dot" style="background: {trip ? trip.color : ACTIVITY_COLORS[activity.type]}"></span>
              <span class="chip-title">{activity.title}</span>
              {#if activity.location}
                <span class="chip-location">{activity.location}</span>
              {/if}
              <span class="chip-type">{activity.type}</span>
            </div>
          {/each}
        {/if}

        {#if overlayByDay.has(dateStr)}
          {#each overlayByDay.get(dateStr)! as item}
            <div class="overlay-chip" style="border-left-color: {item.color};" title={item.ownerEmail}>
              <span class="overlay-loc">{item.location || item.ownerEmail}</span>
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

  .day-row.drag-selected {
    background: rgba(59, 130, 246, 0.1) !important;
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

  .day-row.has-trip {
    padding-left: 0;
  }

  .trip-labels {
    display: flex;
    gap: 0.4rem;
    padding: 0 0 2px;
  }

  .trip-badge {
    font-size: 0.65rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.03em;
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

  .overlay-chip {
    display: inline-flex;
    align-items: center;
    border-left: 3px solid;
    padding: 0.1rem 0.4rem;
    font-size: 0.7rem;
    color: #888;
    opacity: 0.75;
  }

  .overlay-loc {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
