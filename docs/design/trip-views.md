# Trip-Centric Views: Design Document

## Scope

Rework Year and Month views to be trip-centric. Activities without a trip are treated as unnamed single-activity trips.

## Trip Colors

Auto-assign colors from a palette based on trip name hash. The palette should have 8-10 distinct, pleasant colors. Standalone activities (no tripName) keep their activity-type color.

```typescript
const TRIP_COLORS = [
  '#4f86c6', '#e07b53', '#6bb86a', '#c75ca2',
  '#d4a843', '#5cbcb6', '#8b6cc1', '#c95454',
  '#5a8f5a', '#c4853d',
];

function tripColor(tripName: string): string {
  let hash = 0;
  for (const ch of tripName) hash = ((hash << 5) - hash + ch.charCodeAt(0)) | 0;
  return TRIP_COLORS[Math.abs(hash) % TRIP_COLORS.length];
}
```

## Year View

### Current
Each row is a month. Activity bars span their date ranges with activity-type colors.

### New
Each row is a month. **Trip bars** span the trip's full date range, colored by trip color. Standalone activities render as individual bars (type-colored).

- Trip bar shows the trip name as label
- Click a trip bar → popup showing the trip's activities (could be a mini day-view scoped to the trip's dates)
- Hover → tooltip with trip summary (name, dates, locations, activity count)
- Activities within the trip are NOT shown individually — the trip bar is the visual unit

### Detail popup
When clicking a trip bar, show a popover/modal with:
- Trip name and date range
- List of activities (similar to agenda view, scoped to the trip)
- Click an activity to edit it

## Month View

### Current
Per-day cell rendering. Activity bars as segments in each day cell. Lane assignment per week.

### New
Trips own lanes that persist across the entire visible range (not just per-week). A trip's lane stays the same from its start to its end.

Within a trip's lane, the trip bar spans the full date range as a colored background. Activities are rendered as inline text with connecting lines:

```
|------ Flight UA 1234 ------|---- Hotel Brussels ----|--- FOSDEM ---|
```

When activities overlap within a lane (same days), summarize:
- 2 activities: "UA 1234 & Hotel Brussels"
- 3+ activities: "3 activities" (or smart summary)

### Lane assignment
- Each trip gets one lane for its entire duration
- Lanes are assigned globally (not per-week) based on start date
- Standalone activities get their own mini-lane (type-colored, one row)
- Maximum visible lanes: still capped (e.g., 4), with "+N more" for overflow

### Visual
- Trip lane: colored background bar spanning start→end date
- Activity text within the bar: white text, separated by thin vertical marks
- Trip name shown at the start of the bar (or floating label)
- Lane height may be slightly taller than current bars to accommodate text

## Day View

Unchanged for now.

## Implementation Steps

1. Add `tripColor()` utility to date-utils
2. Rework YearView: compute trip bars, render trip-level bars, add detail popup
3. Rework MonthView: global lane assignment by trip, render trip bands with inline activities
4. Update tooltips to show trip info
