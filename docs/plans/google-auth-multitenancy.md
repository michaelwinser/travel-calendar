# Plan: Google OAuth Login + Multi-Tenancy

## Overview

Replace the current unauthenticated, single-user app with Google OAuth login and multi-tenant data isolation. The existing Google OAuth flow (used for Calendar API access) becomes the login mechanism — users authenticate once and grant calendar permissions in the same consent screen.

## Current State

- **No login**: All routes are publicly accessible
- **Single user**: All entities are global; `DefaultUserID = "default"` is hardcoded
- **Separate Calendar OAuth**: Users manually connect Google Calendar via Settings after using the app
- **User-scoped entities**: Only `GoogleCredentials` and `UserCalendar` have a `UserID` field
- **Global entities**: Trip, Item, Document, TripLocation, CalendarLink, ProcessedCalendarEvent have no user ownership

## Goals

1. **Google OAuth as login** — single consent screen that authenticates AND grants calendar access
2. **Session management** — cookie-based sessions so the browser stays logged in
3. **Multi-tenancy** — all data scoped to the authenticated user
4. **Email allowlist** — only permitted users can log in (reuse `.allowed-users` concept)

## Design Decisions

### Auth flow
- OAuth login replaces the separate "Connect Google Calendar" flow in Settings
- Scopes requested at login: `openid email profile` + `calendar.readonly`
- If user denies calendar scope, they can still log in (calendar features disabled)
- Refresh tokens stored in `GoogleCredentials` (existing entity)
- Session cookie (`travel_session`) maps to a server-side session with user info

### Session storage
- Sessions stored in the active store (SQLite or Firestore)
- New `Session` entity: `{ ID, UserID, Email, ExpiresAt, CreatedAt }`
- Cookie: `travel_session=<session-id>`, HttpOnly, Secure, SameSite=Lax
- Session TTL: 30 days (refreshed on activity)

### Multi-tenancy approach
- Add `UserID` field to: `Trip`, `CalendarLink`, `ProcessedCalendarEvent`
- `Item`, `Document`, `TripLocation` are scoped transitively through `Trip`
- All store queries for these entities gain a `userID` parameter
- Handler middleware extracts user from session and injects into request context
- Unauthorized access to another user's data returns 404 (not 403, to avoid enumeration)

### Allowlist
- `ALLOWED_USERS` env var: comma-separated list of emails
- Checked at login time — if email not in list, login is rejected
- If `ALLOWED_USERS` is empty/unset, any Google account can log in (open registration)

---

## Implementation Phases

### Phase 1: Auth Middleware + Google OAuth Login

**Goal**: Users must log in with Google to use the app. No data model changes yet (still single-tenant with `DefaultUserID`).

#### 1A: Backend — Session entity and store methods

New entity `Session`:
```go
type Session struct {
    ID        string    `db:"id" firestore:"-"`
    UserID    string    `db:"user_id" firestore:"userId"`
    Email     string    `db:"email" firestore:"email"`
    ExpiresAt time.Time `db:"expires_at" firestore:"expiresAt"`
    CreatedAt time.Time `db:"created_at" firestore:"createdAt"`
}
```

New `StoreInterface` methods:
```go
CreateSession(session *Session) error
GetSession(id string) (*Session, error)
DeleteSession(id string) error
DeleteExpiredSessions() error
```

Add to both SQLite and Firestore stores.

**Files**: `entity/session.go`, `store/interface.go`, `store/sqlite.go`, `store/firestore.go`, `store/store_test.go`

#### 1B: Backend — Auth middleware

New middleware in `internal/auth/`:
- Reads `travel_session` cookie
- Looks up session in store
- If valid: sets `UserID` and `Email` in request context
- If invalid/missing: returns 401 for `/api/*` routes, redirects to `/login` for page requests
- Exempt paths: `/api/auth/google`, `/api/auth/google/callback`, `/health`, `/login`

Context helpers:
```go
func UserIDFromContext(ctx context.Context) string
func EmailFromContext(ctx context.Context) string
```

**Files**: `internal/auth/middleware.go`, `internal/auth/context.go`

#### 1C: Backend — Unified OAuth login flow

Modify the existing `/api/auth/google` and `/api/auth/google/callback` endpoints:

**`GET /api/auth/google`** (login redirect):
- Request scopes: `openid email profile https://www.googleapis.com/auth/calendar.readonly`
- Redirect to Google consent screen
- Store CSRF state token in a short-lived cookie

**`GET /api/auth/google/callback`** (token exchange + session creation):
1. Exchange code for tokens
2. Fetch user profile from `https://www.googleapis.com/oauth2/v2/userinfo`
3. Check email against `ALLOWED_USERS` allowlist (reject if not permitted)
4. Save tokens to `GoogleCredentials` (keyed by user's Google ID or email)
5. Create `Session`, set `travel_session` cookie
6. Redirect to `/`

**`DELETE /api/auth/google/disconnect`** → becomes **`POST /api/auth/logout`**:
- Delete session from store
- Clear cookie
- Optionally revoke Google token

**`GET /api/auth/status`** (new):
- Returns `{ logged_in: true, email: "..." }` or `{ logged_in: false }`
- Used by frontend to show login state

Remove the separate "Connect Calendar" flow from Settings — it's now part of login.

**Files**: `internal/handler/auth.go` (new, extracted from handler.go), `internal/service/calendar.go`, `api/openapi.yaml`

#### 1D: Frontend — Login page and auth-aware layout

**`/login` route**:
- Simple page with "Sign in with Google" button
- Calls `/api/auth/google` to get redirect URL
- Shows error if login was rejected (not in allowlist)

**Layout changes** (`+layout.svelte`, `+layout.ts`):
- Check `/api/auth/status` on load
- If not logged in, redirect to `/login`
- Show user email + logout button in header

**Settings page cleanup**:
- Remove "Connect Google Calendar" section (happens at login)
- Keep calendar selection UI (which calendars to monitor)
- Show connected Google account info (read-only)

**Remove `/oauth/google/callback`** frontend route — callback now handled entirely by backend (redirects to `/`).

**Files**: `routes/login/+page.svelte`, `routes/+layout.svelte`, `routes/+layout.ts`, `routes/settings/+page.svelte`

#### 1E: Wire it up

- Register auth middleware in `main.go`
- Update `CalendarService` to use `UserIDFromContext` instead of `DefaultUserID`
- Update OpenAPI spec with new/changed endpoints
- Regenerate shared types

**Test**: Deploy, verify login flow end-to-end, verify calendar features still work post-login.

---

### Phase 2: Multi-Tenancy — User-Scoped Data

**Goal**: Each user sees only their own trips and data.

#### 2A: Entity changes

Add `UserID` to:
```go
// Trip
UserID string `db:"user_id" firestore:"userId"`

// CalendarLink
UserID string `db:"user_id" firestore:"userId"`

// ProcessedCalendarEvent
UserID string `db:"user_id" firestore:"userId"`
```

`Item`, `Document`, `TripLocation` don't need `UserID` — they're accessed through their parent Trip, and handlers will verify Trip ownership.

#### 2B: Store interface changes

Update signatures to require `userID`:
```go
// Trip methods
ListTrips(userID string, upcoming, past *bool, purpose *string) ([]Trip, error)
GetTrip(userID string, id uuid.UUID) (*Trip, error)
CreateTrip(trip *Trip) error  // UserID set on entity
UpdateTrip(userID string, trip *Trip) error
DeleteTrip(userID string, id uuid.UUID) error
SearchTrips(userID string, q string) ([]Trip, error)
GetTripsForDateRange(userID string, from, to time.Time) ([]Trip, error)

// Item methods (verify trip ownership at handler level)
// Signatures unchanged, but handlers check trip.UserID first

// CalendarLink methods
CreateCalendarLink(link *CalendarLink) error  // UserID set on entity
// etc.
```

#### 2C: Store implementations

**SQLite**: Add `user_id` column to `trips`, `calendar_links`, `processed_calendar_events` tables. Migration adds the column with default value matching `DefaultUserID` so existing data is preserved.

**Firestore**: Add `userId` field to trip documents. Queries filter by `userId`.

Update `store_test.go` shared test suite to pass `userID` and test isolation between users.

#### 2D: Handler + service changes

- All handlers extract `userID` from auth context
- Pass `userID` to store/service methods
- `GetTrip` returns 404 if trip belongs to different user
- `CreateTrip` sets `UserID` from context
- Calendar service uses context user for all operations

#### 2E: No data migration needed

Starting fresh — no need to migrate existing unscoped data.

---

## Implementation Order

```
Phase 1A  →  1B  →  1C  →  1D  →  1E  →  deploy + verify
                                              ↓
Phase 2A  →  2B  →  2C  →  2D  →  2E  →  deploy + verify
```

Phase 1 is independently deployable — adds login without breaking the data model.
Phase 2 adds isolation and can be deployed separately.

## Components Affected

| Component | Changes |
|-----------|---------|
| `api` | New/modified auth endpoints, remove separate calendar connect |
| `backend` | New session entity, auth middleware, store changes, handler refactor |
| `frontend` | Login page, layout auth checks, settings page cleanup |
| `shared` | Regenerated types |

## Env Var Changes

| Variable | Purpose |
|----------|---------|
| `ALLOWED_USERS` | Comma-separated email allowlist (empty = open) |
| `SESSION_SECRET` | Secret for signing session cookies (generated if unset) |
| `GOOGLE_CLIENT_ID` | Existing — now required (was optional) |
| `GOOGLE_CLIENT_SECRET` | Existing — now required (was optional) |
| `GOOGLE_REDIRECT_URL` | Existing — callback URL |

## Risks & Considerations

- **Calendar scope denial**: User might approve login but deny calendar access. App should handle gracefully (show calendar features as disabled).
- **Token refresh**: Existing refresh logic in `CalendarService` should work unchanged since tokens are still stored in `GoogleCredentials`.
- **Existing data**: No migration needed — starting fresh. Existing unscoped data can be ignored/cleared.
- **Session cleanup**: Need a periodic job or lazy cleanup for expired sessions.
- **CSRF**: OAuth state parameter handles CSRF for login. API endpoints should be safe since they use cookie auth with SameSite=Lax (no cross-origin POST).
