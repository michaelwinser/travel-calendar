# Frontend PRD

## Primary Use Case

Let the user quickly plan and review a coarse-grained view of their travel plans: **what, where, and when**.

## Core Principles

1. **Every day is visible.** The month, year, and day views are linear — they show every calendar day, whether or not there are activities. Empty days are just as important as busy ones (they show when you're home).

2. **Travel is the signal.** The visuals in all linear views are optimized to make it immediately obvious when travel is happening. Color, bars, and layout all serve this.

3. **Conflicts are obvious.** When activities put the user in multiple incompatible locations on the same day, it must be visually clear without being overwhelming.

4. **Quick entry, easy correction.** Adding an activity should be fast (quick add). Editing should be inline where possible. The UI should feel more like a spreadsheet than a form.

5. **Keyboard-first.** Navigation, editing, and creation should all work without a mouse. Tab, arrows, Enter, Escape.

## Views

### 1. Month View

The default view. Each row is a week (7 columns, Sun-Sat). The left edge shows the month and year.

- **Infinite scrolling** — no pagination, no month boundaries. Weeks flow continuously.
- **Default position** — current date centered in the viewport.
- **Cells** — each cell is a day. Activities are shown as colored bars spanning across days. Multi-day activities form continuous horizontal bands.
- **Month indicators** — alternating subtle background shading per month. Sticky month/year label on the left edge.
- **Interactions** — click a day to switch to day view at that date. Quick add targets the clicked date.

### 2. Year View

Overview for long-range planning. Each row is a month (12 rows visible = one year).

- **Compact** — much less space per day than month view. Days are small cells or thin columns.
- **Activities** shown as colored bars spanning days within the month row.
- **Purpose** — see the shape of 6-12 months at a glance. Where are the trips? Where are the gaps? Is there enough recovery time?
- **Interactions** — click a month to switch to month view at that month.

### 3. Day View

Every row is a single day. Spreadsheet-like editing.

- **Columns** — date, location, activity title, type, notes.
- **Inline editing** — click a cell to edit. Tab to move between cells. Enter to confirm, Escape to cancel.
- **Keyboard navigation** — arrow keys move between cells. Tab moves forward. Shift+Tab moves back.
- **Copy/paste** — select a range, paste into multiple cells (future, but the layout should support it).
- **Empty days** — still shown as rows. Location defaults to "Home".
- **Scrolling** — infinite, like month view. Defaults to current date.

### 4. Agenda View

Filtered list of activities only (like Google Calendar's agenda view).

- **Each row is an activity**, not a day. Sorted by start date.
- **Shows** — title, type, dates, location.
- **No empty days** — this is the only non-linear view.
- **Purpose** — quick reference for what's coming up. Compact overview of all planned activities.
- **Interactions** — click to edit. Delete button.

## Input Modes

### Quick Add (primary input)

Present in all views. A text input at the top of the screen.

- User types freeform text (e.g., "FOSDEM Jan 22 - Feb 3 in Brussels")
- **As the user types**, a form appears below and fills in fields in real-time from the parse result. The user sees the parser working live.
- The user can continue editing the text, or Tab into the form fields to refine them directly.
- Tab cycles between the quick add text input and the form fields — they're part of the same flow, not separate modes.
- Press Enter (or a Create button) to confirm. Escape to dismiss.
- The same form is used for editing existing activities (see below).

### Click to Add

In month and day views, clicking on an empty day opens the quick add with the date pre-filled.

- Minimal UI — the quick add bar activates with the date already set in the form.
- The user just needs to type a title and optionally a location.

### Click and Drag

In month view, clicking and dragging across multiple days creates a date range.

- The quick add form appears with start and end dates pre-filled.
- Same flow as click to add, but with a range.

### Edit Existing Activity

Double-clicking (or pressing Enter on) an existing activity opens it in the same form used by quick add.

- The form is pre-filled with the activity's current values.
- The quick add text field is empty (or could show a text representation of the activity — TBD).
- Save or cancel.

### Copy/Paste in Day View (deferred)

The day view's spreadsheet layout could support selecting a range and pasting into multiple cells, similar to a spreadsheet. This is remarkably convenient for the use case but complex to implement. Noted for completeness — not in initial phases.

## Conflict Visualization

Applies to month, year, and day views.

- A day with activities in multiple locations gets a **visual conflict indicator**: red border, warning icon, or background wash.
- The `GET /api/activities/check/{date}` endpoint provides `hasConflict` — the frontend calls this or computes it client-side from the activity list.

## Navigation

- **View switcher** — tabs or buttons to switch between Month / Year / Day / Agenda.
- **Today button** — returns to current date in any linear view.
- **Go to date** — keyboard shortcut (G) opens a date picker.
- **URL routing** — view and date in the URL (e.g., `/month/2026-10`, `/day/2026-10-05`).

## Color System

Activity types map to colors:

| Type | Color intent |
|------|-------------|
| travel | Blue — movement, transit |
| stay | Green — settled, present |
| conference | Purple — professional event |
| vacation | Orange/Yellow — leisure |
| commitment | Red/Pink — obligation, can conflict with travel |

Colors are used for:
- Activity bars in month and year views
- Row background tint in day view
- Type badges in agenda view

## Implementation Phases

### Phase 1: Agenda view + quick add + form modal
- Agenda view — scaffolding to exercise the core input/edit experience
- Quick add bar with live-parsing form
- Activity form modal — universal create/edit surface (4 entry points: quick add, click to add, click and drag, edit existing)
- View switching tabs (Month / Year / Day / Agenda)
- Color system for activity types
- Delete with confirmation

### Phase 2: Month view (the default view)
- Week-row layout with colored activity bars spanning days
- Infinite scroll with sticky month/year labels
- Conflict indicators on days with incompatible locations
- Click a day to open form with date pre-filled
- Click and drag across days to open form with date range pre-filled
- Today button to return to current date

### Phase 3: Day view
- Each row is a day, read-only display
- Click a day to edit via form modal
- Infinite scroll, defaults to current date
- Conflict indicators

### Phase 4: Year view
- Each row is a month, compact day cells
- Activity bars at reduced scale
- Click a month to switch to month view

## Out of Scope (for now)

- Spreadsheet-style inline editing and copy/paste in day view
- Drag-and-drop to move/resize activities
- Google Calendar import
- Sharing/export
- Mobile-specific layout
- Offline support
