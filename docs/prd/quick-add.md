# Quick Add — Mini PRD

## Vision

Parse a freeform text string into a proposed Activity resource. The parser does its best guess; the result is always reviewed by the user before creation. This is a **form filler**, not an auto-creator.

## Key Principle

The parse endpoint never writes to the database. It returns a `CreateActivityRequest` that the client can display, let the user edit, and then POST to `/api/activities` via the normal creation flow with full validation.

## Endpoint

```
POST /api/activities/parse
Content-Type: application/json

{ "text": "FOSDEM Jan 22 - Feb 3 in Brussels" }
```

Response: `ParseResult`

```json
{
  "activity": {
    "title": "FOSDEM",
    "type": "stay",
    "startDate": "2027-01-22",
    "endDate": "2027-02-03",
    "location": "Brussels"
  },
  "confidence": {
    "title": "high",
    "type": "low",
    "startDate": "high",
    "endDate": "high",
    "location": "high"
  },
  "unparsed": ""
}
```

- `activity` — A `CreateActivityRequest` with best-guess values. Fields that couldn't be parsed are omitted (null/empty).
- `confidence` — Per-field indicator: `high` (explicit in input), `medium` (inferred), `low` (defaulted). Clients can use this to highlight fields that need attention.
- `unparsed` — Any tokens the parser couldn't classify. Helps the user see what was ignored.

## Supported Patterns

### Dates

| Input | Parsed as | Rule |
|-------|-----------|------|
| `Jan 22` | 2027-01-22 | Month in the past → next year |
| `Mar 25` | 2026-03-25 | Month today or ahead → this year |
| `Jan 22 2026` | 2026-01-22 | Explicit year always wins |
| `Jan 22 - Feb 3` | start=Jan 22, end=Feb 3 | Dash/to between dates = range |
| `from Jan 22 to Feb 3` | start=Jan 22, end=Feb 3 | from/to as date delimiters |
| `on Mar 20` | start=Mar 20, end=Mar 20 | Single date (one day) |
| `Mar 20` | start=Mar 20, end=Mar 20 | Bare date = single day |

**Year inference rule:** If the parsed month-day is before today, assume next year. If it's today or later, assume this year. Explicit year always overrides.

**Date formats recognized:**
- `Jan 22`, `January 22`, `jan 22` (month name + day)
- `Jan 22 2026` (month name + day + year)
- No numeric-only dates (10/4 is too ambiguous)

### Location

| Input | Parsed as | Rule |
|-------|-----------|------|
| `in Brussels` | location=Brussels | `in` keyword |
| `at MXP` | location=MXP | `at` keyword |
| `Brussels` (at end, after dates) | location=Brussels | Trailing proper noun after dates |

### From/To Disambiguation

`from` and `to` can mean dates or locations. Rules:

1. If both `from` and `to` values look like dates → date range
2. If both look like place names → route (location = "from → to", type = travel)
3. If mixed (date + place) → treat `from` as date, `to` as place (or vice versa based on what parses)

Examples:

| Input | Parsed as |
|-------|-----------|
| `from Jan 22 to Feb 3` | startDate=Jan 22, endDate=Feb 3 |
| `from MXP to EWR` | location="MXP → EWR", type=travel |
| `Flight UA 19 on Mar 20 from MXP to EWR` | title="Flight UA 19", startDate=Mar 20, location="MXP → EWR", type=travel |
| `from Jan 22 in Brussels` | startDate=Jan 22, location=Brussels |

**Heuristic:** A token following `from`/`to` is a date if it starts with a month name or is a recognized date pattern. Otherwise it's a location.

### Title

Everything that isn't recognized as a date, location keyword, or connector becomes the title. The parser strips recognized tokens and joins the remainder.

| Input | Title |
|-------|-------|
| `FOSDEM Jan 22 - Feb 3 in Brussels` | FOSDEM |
| `Flight UA 19 on Mar 20 from MXP to EWR` | Flight UA 19 |
| `Dentist on Apr 5` | Dentist |
| `Team offsite in NYC from Mar 10 to Mar 14` | Team offsite |

### Activity Type

Inferred from keywords in the title or location pattern. Default: `stay`.

| Signal | Type |
|--------|------|
| `flight`, `fly`, airport codes (3 uppercase letters), `from X to Y` where X/Y are not dates | `travel` |
| `conference`, `summit`, `conf` | `conference` |
| `vacation`, `holiday`, `break` | `vacation` |
| `dentist`, `doctor`, `appointment`, `meeting` | `commitment` |
| None of the above | `stay` |

## CLI Flow

```
$ travel quick "FOSDEM Jan 22 - Feb 3 in Brussels"

  Title:     FOSDEM
  Type:      conference
  Dates:     2027-01-22 → 2027-02-03
  Location:  Brussels

  [C]reate  [E]dit  [A]bort? c

Created: FOSDEM (a1b2c3d4)
  2027-01-22 → 2027-02-03  [conference]  Brussels
```

If the user presses `E`, the CLI presents each field for inline editing (prefilled with the parsed value, user can accept with Enter or type a new value):

```
  [C]reate  [E]dit  [A]bort? e

  Title [FOSDEM]:
  Type [conference]:
  Start date [2027-01-22]:
  End date [2027-02-03]:
  Location [Brussels]:

  [C]reate  [A]bort? c
```

## Acceptance Tests

### UC-2001: Simple event with location
- Input: `FOSDEM Jan 22 - Feb 3 in Brussels`
- Expected: title=FOSDEM, start=2027-01-22 (next year), end=2027-02-03 (next year), location=Brussels

### UC-2002: Flight with route
- Input: `Flight UA 19 on Mar 20 from MXP to EWR`
- Expected: title=Flight UA 19, start=2027-03-20, type=travel, location=MXP → EWR

### UC-2003: Single day with location keyword
- Input: `Dentist on Apr 5 at Home`
- Expected: title=Dentist, start=2026-04-05, end=2026-04-05, type=commitment, location=Home

### UC-2004: Date range with from/to
- Input: `Team offsite from Mar 10 to Mar 14 in NYC`
- Expected: title=Team offsite, start=2027-03-10, end=2027-03-14, location=NYC

### UC-2005: Explicit year
- Input: `Christmas vacation Dec 20 2026 - Jan 3 2027`
- Expected: title=Christmas vacation, start=2026-12-20, end=2027-01-03, type=vacation

### UC-2006: Minimal input
- Input: `meeting tomorrow`
- Expected: title=meeting tomorrow, type=commitment, all other fields null/empty (unparsed contains "tomorrow" since relative dates are not supported in v1)

### UC-2007: Year inference — past month means next year
- Input: `Jan 15` (when today is March 22, 2026)
- Expected: start=2027-01-15

### UC-2008: Year inference — future month means this year
- Input: `Apr 10` (when today is March 22, 2026)
- Expected: start=2026-04-10

### UC-2009: Unparsable input returns best effort
- Input: `stuff and things`
- Expected: title=stuff and things, all other fields null, confidence all low

### UC-2010: From/to as locations not dates
- Input: `Train from London to Paris on May 5`
- Expected: title=Train, type=travel, start=2026-05-05, location=London → Paris

## Out of Scope (for now)

- Relative dates ("next Friday", "tomorrow", "in 2 weeks")
- Timezone handling
- Recurring events
- Multi-leg trips in a single string
- LLM-based parsing
- Learning from corrections
