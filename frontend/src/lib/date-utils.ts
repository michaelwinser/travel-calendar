// Pure date computation helpers for calendar views.
// All dates are YYYY-MM-DD strings to match the API.

import type { Activity } from './api';

export interface Week {
  days: string[];     // 7 YYYY-MM-DD strings, Sun-Sat
  weekStart: string;  // ISO date of the Sunday
}

export interface ActivityBar {
  activity: Activity;
  startCol: number;   // 0-6 column index within the week
  spanCols: number;   // how many columns wide
  lane: number;       // vertical stacking position
  startsHere: boolean; // activity starts in this week (round left)
  endsHere: boolean;   // activity ends in this week (round right)
}

export function dateToString(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

export function stringToDate(s: string): Date {
  const [y, m, d] = s.split('-').map(Number);
  return new Date(y, m - 1, d);
}

export function today(): string {
  return dateToString(new Date());
}

export function addDays(dateStr: string, n: number): string {
  const d = stringToDate(dateStr);
  d.setDate(d.getDate() + n);
  return dateToString(d);
}

/** Get the Sunday that starts the week containing the given date. */
function weekSunday(dateStr: string): Date {
  const d = stringToDate(dateStr);
  d.setDate(d.getDate() - d.getDay());
  return d;
}

/** Generate weeks covering a date range. Each week runs Sun-Sat. */
export function getWeeksForRange(startDate: string, endDate: string): Week[] {
  const weeks: Week[] = [];
  const sun = weekSunday(startDate);
  const end = stringToDate(endDate);

  while (sun <= end) {
    const days: string[] = [];
    for (let i = 0; i < 7; i++) {
      const d = new Date(sun);
      d.setDate(d.getDate() + i);
      days.push(dateToString(d));
    }
    weeks.push({ days, weekStart: days[0] });
    sun.setDate(sun.getDate() + 7);
  }
  return weeks;
}

/** Get the month label for a date (e.g., "Mar 2026"). */
export function monthLabel(dateStr: string): string {
  const d = stringToDate(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
}

/** Get month index (0-11) for alternating shading. */
export function monthIndex(dateStr: string): number {
  return stringToDate(dateStr).getMonth();
}

/** Check if an activity overlaps a given date. */
function activityOverlapsDate(a: Activity, dateStr: string): boolean {
  return a.startDate <= dateStr && a.endDate >= dateStr;
}

/** Get the columns an activity occupies in a given week. Returns null if no overlap. */
export function getActivityDaysInWeek(
  activity: Activity,
  week: Week,
): { startCol: number; spanCols: number; startsHere: boolean; endsHere: boolean } | null {
  const weekEnd = week.days[6];
  const weekStart = week.days[0];

  // No overlap at all
  if (activity.endDate < weekStart || activity.startDate > weekEnd) return null;

  // Clip to week boundaries
  const effectiveStart = activity.startDate > weekStart ? activity.startDate : weekStart;
  const effectiveEnd = activity.endDate < weekEnd ? activity.endDate : weekEnd;

  const startCol = week.days.indexOf(effectiveStart);
  const endCol = week.days.indexOf(effectiveEnd);
  if (startCol === -1 || endCol === -1) return null;

  return {
    startCol,
    spanCols: endCol - startCol + 1,
    startsHere: activity.startDate >= weekStart,
    endsHere: activity.endDate <= weekEnd,
  };
}

/** Compute activity bars for a week with lane assignment (greedy first-fit). */
export function getActivityBarsForWeek(
  activities: Activity[],
  week: Week,
  colorMap: Record<string, string>,
): ActivityBar[] {
  const bars: ActivityBar[] = [];

  // Get all activities that overlap this week, sorted by start date then duration (longer first)
  const overlapping = activities
    .map(a => ({ activity: a, span: getActivityDaysInWeek(a, week) }))
    .filter((x): x is { activity: Activity; span: NonNullable<ReturnType<typeof getActivityDaysInWeek>> } => x.span !== null)
    .sort((a, b) => {
      if (a.activity.startDate !== b.activity.startDate) return a.activity.startDate < b.activity.startDate ? -1 : 1;
      // Longer activities first for better visual stacking
      const aDur = a.span.spanCols;
      const bDur = b.span.spanCols;
      return bDur - aDur;
    });

  // Greedy lane assignment
  const lanes: boolean[][] = []; // lanes[lane][col] = occupied

  for (const { activity, span } of overlapping) {
    let lane = 0;
    while (true) {
      if (!lanes[lane]) lanes[lane] = Array(7).fill(false);
      const conflict = lanes[lane].slice(span.startCol, span.startCol + span.spanCols).some(Boolean);
      if (!conflict) break;
      lane++;
    }

    // Mark columns as occupied
    for (let c = span.startCol; c < span.startCol + span.spanCols; c++) {
      lanes[lane][c] = true;
    }

    bars.push({
      activity,
      startCol: span.startCol,
      spanCols: span.spanCols,
      lane,
      startsHere: span.startsHere,
      endsHere: span.endsHere,
    });
  }

  return bars;
}

/** Check if a date has conflicting activities (multiple locations).
 *  Activities within the same trip do not conflict with each other. */
export function hasConflict(dateStr: string, activities: Activity[]): boolean {
  const overlapping = activities.filter(a => activityOverlapsDate(a, dateStr) && a.location);
  if (overlapping.length < 2) return false;

  // Collect locations per source (trip or standalone).
  // All activities in the same trip are one source.
  // Each standalone activity is its own source.
  const sourceLocations: string[] = [];
  const tripsSeen = new Set<string>();
  let standaloneIdx = 0;

  for (const a of overlapping) {
    if (a.tripId) {
      if (!tripsSeen.has(a.tripId)) {
        tripsSeen.add(a.tripId);
        // Use the trip's first activity's location as representative
        sourceLocations.push(a.location!);
      }
      // Skip other activities in the same trip
    } else {
      sourceLocations.push(a.location!);
    }
  }

  // Conflict if sources have different locations
  const uniqueLocations = new Set(sourceLocations);
  return uniqueLocations.size > 1;
}

/** Per-day bar segment for rendering inside day cells. */
export interface DayBarSegment {
  activity: Activity;
  lane: number;
  isStart: boolean;
  isEnd: boolean;
  color: string;
}

/**
 * Compute per-day bar segments for a week.
 * Lane assignment is done at the week level so multi-day activities
 * maintain the same vertical position across all their day cells.
 * Returns a map: dayIndex (0-6) → array of segments.
 */
/** Get the lane assigned to a specific activity in a week. */
export function getActivityLane(
  activities: Activity[],
  week: Week,
  activityId: string,
): number {
  const bars = getActivityBarsForWeek(activities, week, {} as Record<string, string>);
  const bar = bars.find(b => b.activity.id === activityId);
  return bar?.lane ?? 0;
}

export function getDaySegmentsForWeek(
  activities: Activity[],
  week: Week,
  colorMap: Record<string, string>,
  maxLanes: number,
): { segments: Map<number, DayBarSegment[]>; overflowPerDay: number[] } {
  // First, do week-level lane assignment (same as getActivityBarsForWeek)
  const bars = getActivityBarsForWeek(activities, week, colorMap);

  // Build per-day segment lists
  const segments = new Map<number, DayBarSegment[]>();
  for (let d = 0; d < 7; d++) segments.set(d, []);

  const overflowPerDay = Array(7).fill(0);

  for (const bar of bars) {
    for (let col = bar.startCol; col < bar.startCol + bar.spanCols; col++) {
      const seg: DayBarSegment = {
        activity: bar.activity,
        lane: bar.lane,
        isStart: col === bar.startCol && bar.startsHere,
        isEnd: col === bar.startCol + bar.spanCols - 1 && bar.endsHere,
        color: colorMap[bar.activity.type] ?? '#999',
      };

      if (bar.lane < maxLanes) {
        segments.get(col)!.push(seg);
      } else {
        overflowPerDay[col]++;
      }
    }
  }

  // Sort segments by lane within each day
  for (const segs of segments.values()) {
    segs.sort((a, b) => a.lane - b.lane);
  }

  return { segments, overflowPerDay };
}

/** Generate month data for the year view. Each month has its days and metadata. */
export interface YearMonth {
  year: number;
  month: number;          // 0-11
  label: string;          // "Jan 2026"
  days: string[];         // YYYY-MM-DD for each day in the month
  firstDayOffset: number; // 0=Sun, 1=Mon, etc. — day-of-week of the 1st
}

export function getMonthsForRange(startDate: string, endDate: string): YearMonth[] {
  const months: YearMonth[] = [];
  const start = stringToDate(startDate);
  const end = stringToDate(endDate);

  let year = start.getFullYear();
  let month = start.getMonth();

  while (true) {
    const firstDay = new Date(year, month, 1);
    if (firstDay > end) break;

    const daysInMonth = new Date(year, month + 1, 0).getDate();
    const days: string[] = [];
    for (let d = 1; d <= daysInMonth; d++) {
      days.push(dateToString(new Date(year, month, d)));
    }

    months.push({
      year,
      month,
      label: firstDay.toLocaleDateString('en-US', { month: 'short', year: 'numeric' }),
      days,
      firstDayOffset: firstDay.getDay(),
    });

    month++;
    if (month > 11) { month = 0; year++; }
  }
  return months;
}

/** Activity bar for the year view — spans day columns within a month row. */
export interface YearBar {
  activity: Activity;
  startDay: number;   // 0-based day index within the month
  spanDays: number;
  lane: number;
  color: string;
}

/** Compute activity bars for a month in the year view with lane assignment. */
export function getYearBarsForMonth(
  activities: Activity[],
  month: YearMonth,
  colorMap: Record<string, string>,
  maxLanes: number,
): YearBar[] {
  const monthStart = month.days[0];
  const monthEnd = month.days[month.days.length - 1];

  const overlapping = activities
    .filter(a => a.endDate >= monthStart && a.startDate <= monthEnd)
    .sort((a, b) => {
      if (a.startDate !== b.startDate) return a.startDate < b.startDate ? -1 : 1;
      // Longer first
      return (b.endDate > a.endDate ? 1 : b.endDate < a.endDate ? -1 : 0);
    });

  const bars: YearBar[] = [];
  const lanes: boolean[][] = [];

  for (const activity of overlapping) {
    const effectiveStart = activity.startDate > monthStart ? activity.startDate : monthStart;
    const effectiveEnd = activity.endDate < monthEnd ? activity.endDate : monthEnd;

    const startDay = month.days.indexOf(effectiveStart);
    const endDay = month.days.indexOf(effectiveEnd);
    if (startDay === -1 || endDay === -1) continue;

    const spanDays = endDay - startDay + 1;

    // Lane assignment
    let lane = 0;
    while (lane < maxLanes) {
      if (!lanes[lane]) lanes[lane] = Array(month.days.length).fill(false);
      const conflict = lanes[lane].slice(startDay, startDay + spanDays).some(Boolean);
      if (!conflict) break;
      lane++;
    }
    if (lane >= maxLanes) continue; // skip if no room

    for (let d = startDay; d < startDay + spanDays; d++) {
      lanes[lane][d] = true;
    }

    bars.push({
      activity,
      startDay,
      spanDays,
      lane,
      color: colorMap[activity.type] ?? '#999',
    });
  }

  return bars;
}

/** Trip-level bar for the year view. */
export interface TripYearBar {
  tripId: string | null;  // null for standalone activities
  tripName: string;
  color: string;
  startDay: number;
  spanDays: number;
  lane: number;
  activityCount: number;
  activities: Activity[];
}

/** Compute trip-level bars for a month in the year view.
 *  Each trip becomes one bar spanning its activities' date range.
 *  Standalone activities become individual bars. */
export function getTripBarsForMonth(
  activities: Activity[],
  month: YearMonth,
  tripColors: Map<string, { name: string; color: string }>,
  activityColors: Record<string, string>,
  maxLanes: number,
): TripYearBar[] {
  const monthStart = month.days[0];
  const monthEnd = month.days[month.days.length - 1];

  // Group activities by trip
  const tripGroups = new Map<string, Activity[]>();
  const standalone: Activity[] = [];

  for (const a of activities) {
    if (a.endDate < monthStart || a.startDate > monthEnd) continue;
    if (a.tripId) {
      const list = tripGroups.get(a.tripId);
      if (list) list.push(a);
      else tripGroups.set(a.tripId, [a]);
    } else {
      standalone.push(a);
    }
  }

  // Build bar entries (trips + standalone), each with their month span
  interface BarEntry {
    tripId: string | null;
    name: string;
    color: string;
    startDay: number;
    spanDays: number;
    activityCount: number;
    activities: Activity[];
  }

  const entries: BarEntry[] = [];

  for (const [tripId, acts] of tripGroups) {
    const trip = tripColors.get(tripId);
    let earliest = acts[0].startDate;
    let latest = acts[0].endDate;
    for (const a of acts) {
      if (a.startDate < earliest) earliest = a.startDate;
      if (a.endDate > latest) latest = a.endDate;
    }

    const effectiveStart = earliest > monthStart ? earliest : monthStart;
    const effectiveEnd = latest < monthEnd ? latest : monthEnd;
    const startDay = month.days.indexOf(effectiveStart);
    const endDay = month.days.indexOf(effectiveEnd);
    if (startDay === -1 || endDay === -1) continue;

    entries.push({
      tripId,
      name: trip?.name ?? 'Trip',
      color: trip?.color ?? '#999',
      startDay,
      spanDays: endDay - startDay + 1,
      activityCount: acts.length,
      activities: acts,
    });
  }

  for (const a of standalone) {
    const effectiveStart = a.startDate > monthStart ? a.startDate : monthStart;
    const effectiveEnd = a.endDate < monthEnd ? a.endDate : monthEnd;
    const startDay = month.days.indexOf(effectiveStart);
    const endDay = month.days.indexOf(effectiveEnd);
    if (startDay === -1 || endDay === -1) continue;

    entries.push({
      tripId: null,
      name: a.title,
      color: activityColors[a.type] ?? '#999',
      startDay,
      spanDays: endDay - startDay + 1,
      activityCount: 1,
      activities: [a],
    });
  }

  // Sort by start day, then longer first
  entries.sort((a, b) => {
    if (a.startDay !== b.startDay) return a.startDay - b.startDay;
    return b.spanDays - a.spanDays;
  });

  // Lane assignment
  const lanes: boolean[][] = [];
  const bars: TripYearBar[] = [];

  for (const entry of entries) {
    let lane = 0;
    while (lane < maxLanes) {
      if (!lanes[lane]) lanes[lane] = Array(month.days.length).fill(false);
      const conflict = lanes[lane].slice(entry.startDay, entry.startDay + entry.spanDays).some(Boolean);
      if (!conflict) break;
      lane++;
    }
    if (lane >= maxLanes) continue;

    for (let d = entry.startDay; d < entry.startDay + entry.spanDays; d++) {
      lanes[lane][d] = true;
    }

    bars.push({
      tripId: entry.tripId,
      tripName: entry.name,
      color: entry.color,
      startDay: entry.startDay,
      spanDays: entry.spanDays,
      lane,
      activityCount: entry.activityCount,
      activities: entry.activities,
    });
  }

  return bars;
}

/** A trip (or standalone activity) with its global lane assignment for month view. */
export interface TripLane {
  tripId: string | null;
  tripName: string;
  color: string;
  startDate: string;
  endDate: string;
  lane: number;
  activities: Activity[];
}

/** Compute global trip lane assignments for a date range.
 *  Each trip gets one lane spanning its full duration.
 *  Standalone activities each get their own lane.
 *  Returns lanes sorted by start date. */
export function computeTripLanes(
  activities: Activity[],
  tripColors: Map<string, { name: string; color: string }>,
  activityColors: Record<string, string>,
  maxLanes: number,
): TripLane[] {
  // Group activities by trip
  const tripGroups = new Map<string, Activity[]>();
  const standalone: Activity[] = [];

  for (const a of activities) {
    if (a.tripId) {
      const list = tripGroups.get(a.tripId);
      if (list) list.push(a);
      else tripGroups.set(a.tripId, [a]);
    } else {
      standalone.push(a);
    }
  }

  // Build lane entries
  interface LaneEntry {
    tripId: string | null;
    name: string;
    color: string;
    startDate: string;
    endDate: string;
    activities: Activity[];
  }

  const entries: LaneEntry[] = [];

  for (const [tripId, acts] of tripGroups) {
    const trip = tripColors.get(tripId);
    let earliest = acts[0].startDate;
    let latest = acts[0].endDate;
    for (const a of acts) {
      if (a.startDate < earliest) earliest = a.startDate;
      if (a.endDate > latest) latest = a.endDate;
    }
    entries.push({
      tripId,
      name: trip?.name ?? 'Trip',
      color: trip?.color ?? '#999',
      startDate: earliest,
      endDate: latest,
      activities: acts.sort((a, b) => a.startDate < b.startDate ? -1 : 1),
    });
  }

  for (const a of standalone) {
    entries.push({
      tripId: null,
      name: a.title,
      color: activityColors[a.type] ?? '#999',
      startDate: a.startDate,
      endDate: a.endDate,
      activities: [a],
    });
  }

  // Sort by start date, then longer first
  entries.sort((a, b) => {
    if (a.startDate !== b.startDate) return a.startDate < b.startDate ? -1 : 1;
    // Longer spans first for better packing
    const aDur = a.endDate > b.endDate ? 1 : a.endDate < b.endDate ? -1 : 0;
    return -aDur;
  });

  // Greedy lane assignment using date ranges
  const laneEnds: string[] = []; // laneEnds[i] = the end date of the last entry in lane i
  const result: TripLane[] = [];

  for (const entry of entries) {
    let lane = 0;
    while (lane < maxLanes) {
      if (lane >= laneEnds.length || laneEnds[lane] < entry.startDate) break;
      lane++;
    }
    if (lane >= maxLanes) continue; // overflow, skip

    if (lane >= laneEnds.length) laneEnds.push(entry.endDate);
    else laneEnds[lane] = entry.endDate;

    result.push({
      tripId: entry.tripId,
      tripName: entry.name,
      color: entry.color,
      startDate: entry.startDate,
      endDate: entry.endDate,
      lane,
      activities: entry.activities,
    });
  }

  return result;
}

/** For a given date, find which TripLane entries are active and what activity text to show. */
export interface DayTripSegment {
  tripLane: TripLane;
  isStart: boolean;
  isEnd: boolean;
  activityLabel: string; // the activity text for this day within the trip
}

export function getDayTripSegments(dateStr: string, tripLanes: TripLane[]): Map<number, DayTripSegment> {
  const segments = new Map<number, DayTripSegment>();

  for (const tl of tripLanes) {
    if (dateStr < tl.startDate || dateStr > tl.endDate) continue;

    // Find which activity(ies) span this day
    const dayActivities = tl.activities.filter(
      a => a.startDate <= dateStr && a.endDate >= dateStr
    );

    let label = '';
    if (dayActivities.length === 1) {
      const a = dayActivities[0];
      // Show label only on the activity's start day (or first visible day)
      if (a.startDate === dateStr) {
        label = a.title;
      }
    } else if (dayActivities.length > 1) {
      // Multiple activities on this day — show on first occurrence
      const starting = dayActivities.filter(a => a.startDate === dateStr);
      if (starting.length === 1) {
        label = starting[0].title;
      } else if (starting.length > 1) {
        label = `${starting.length} activities`;
      }
    }

    segments.set(tl.lane, {
      tripLane: tl,
      isStart: dateStr === tl.startDate,
      isEnd: dateStr === tl.endDate,
      activityLabel: label,
    });
  }

  return segments;
}

/** Get min/max date strings, or null if either is null. */
export function minDate(a: string, b: string): string {
  return a < b ? a : b;
}

export function maxDate(a: string, b: string): string {
  return a > b ? a : b;
}
