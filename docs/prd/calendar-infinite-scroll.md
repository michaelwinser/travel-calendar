# Feature: Calendar Infinite Scroll

## Overview

Replace the current year-based pagination in the calendar view with an infinite scroll interface. Users scroll vertically through months without interruption, and the view defaults to the current month on initial load.

## User Value

Eliminates friction when viewing trips across year boundaries. Users no longer need to click pagination controls to see December and January trips together - they simply scroll. This is the expected modern calendar behavior.

---

## Use Cases

### [UC-CAL-001] Initial Load Shows Current Month

**Actor**: User

**Preconditions**:
- Calendar page loads

**Steps**:
1. User navigates to /calendar

**Expected Result**:
- Current month is visible in the viewport
- Months before and after current month are rendered (for immediate scroll)
- No loading indicator blocks the current month view

---

### [UC-CAL-002] Scroll Down to Future Months

**Actor**: User

**Preconditions**:
- Calendar page is loaded

**Steps**:
1. User scrolls down

**Expected Result**:
- Future months appear seamlessly
- No pagination buttons or "load more" clicks required
- Month/year labels remain clear as user scrolls

---

### [UC-CAL-003] Scroll Up to Past Months

**Actor**: User

**Preconditions**:
- Calendar page is loaded

**Steps**:
1. User scrolls up

**Expected Result**:
- Past months appear seamlessly
- Scroll position remains stable (no jumping)
- Performance remains smooth

---

### [UC-CAL-004] Jump to Today

**Actor**: User

**Preconditions**:
- User has scrolled away from current month

**Steps**:
1. User clicks "Today" button

**Expected Result**:
- View scrolls to current month
- Current month is centered/visible in viewport
- Today's date highlighted as before

---

### [UC-CAL-005] Trip Bars Span Year Boundaries

**Actor**: User

**Preconditions**:
- Trip exists spanning December to January (e.g., Dec 28 - Jan 3)

**Steps**:
1. User scrolls through December/January transition

**Expected Result**:
- Trip bar visible in December with continuation indicator
- Trip bar visible in January with continuation indicator
- Visual continuity maintained (same color, clickable in both months)

---

## UI Components

| Component | Change | Use Cases |
|-----------|--------|-----------|
| `calendar/+page.svelte` | Replace year pagination with infinite scroll container | UC-CAL-001 through UC-CAL-004 |
| `calendar/MonthGrid.svelte` | No changes needed (already receives year/month as props) | UC-CAL-005 |
| New: `calendar/InfiniteCalendar.svelte` (optional) | Encapsulate scroll virtualization logic | UC-CAL-001 through UC-CAL-003 |

---

## Acceptance Criteria

- [ ] Current month visible on initial page load without user action
- [ ] Scrolling up reveals past months indefinitely
- [ ] Scrolling down reveals future months indefinitely
- [ ] "Today" button scrolls view to current month
- [ ] Scroll position stable when prepending months (no jump)
- [ ] Year labels visible as user scrolls across year boundaries
- [ ] Existing trip bar functionality preserved (click to navigate)
- [ ] Existing purpose legend preserved
- [ ] Performance acceptable with 36+ months rendered (3 years)
- [ ] Mobile touch scrolling works correctly

---

## Technical Considerations

### Virtualization Strategy

Two viable approaches:

**Option A: Windowed Rendering (Recommended for MVP)**
- Render ~24 months around current position (12 before, 12 after)
- As user scrolls, add/remove months at edges
- Simpler implementation, sufficient for calendar use case

**Option B: Full Virtualization**
- Only render months visible in viewport + small buffer
- Use virtual scroll library (svelte-virtual-list or similar)
- Better for extreme scroll ranges, more complex

### Scroll Position Stability

When prepending months (scrolling up), the DOM grows above the current position. Without correction, the viewport will appear to jump down. Solutions:

1. **ScrollTop adjustment**: After prepending, adjust `scrollTop` by height of added content
2. **CSS anchor**: Use `overflow-anchor: auto` (modern browsers)
3. **IntersectionObserver**: Detect when sentinel element enters viewport, load more content

### Year Boundary Display

As user scrolls across year boundaries, ensure year context is clear:
- Month headers should include year when it changes: "January 2026"
- Alternatively, sticky year indicator that updates as user scrolls

### Initial Scroll Position

On mount:
1. Render months centered on current month
2. Calculate pixel offset to current month
3. Set `scrollTop` to that offset (before first paint if possible)

---

## Out of Scope

- Horizontal scrolling (months remain vertical)
- Keyboard navigation between months
- URL state for current scroll position (e.g., /calendar?month=2025-06)
- Pinch-to-zoom on mobile
- Backend changes (this is frontend-only)
- Month collapse/expand functionality
- Alternative calendar layouts (week view, day view)

---

## MVP Scope

### Included in MVP
- [UC-CAL-001] Initial load at current month
- [UC-CAL-002] Scroll down to future
- [UC-CAL-003] Scroll up to past
- [UC-CAL-004] Today button
- [UC-CAL-005] Year-spanning trips display correctly
- Windowed rendering (Option A)
- Basic year labels in month headers

### Deferred to Later
- URL state for scroll position
- Full virtualization (if performance issues arise)
- Keyboard navigation
- Accessibility improvements for screen readers
