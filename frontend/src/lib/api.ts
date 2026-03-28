// API client for Travel Calendar.
// Types are generated from openapi.yaml via openapi-typescript.

import type { components } from './api-types';

const API_BASE = '/api';

// --- Types (derived from generated schema) ---

export type Activity = components['schemas']['Activity'];
export type ActivityType = Activity['type'];
export type CreateActivityRequest = components['schemas']['CreateActivityRequest'];
export type UpdateActivityRequest = components['schemas']['UpdateActivityRequest'];
export type ParseResult = components['schemas']['ParseResult'];
export type ParsedActivity = components['schemas']['ParsedActivity'];
export type ParseConfidence = components['schemas']['ParseConfidence'];
export type DateCheck = components['schemas']['DateCheck'];
export type Trip = components['schemas']['Trip'];
export type TripSummary = components['schemas']['TripSummary'];
export type CreateTripRequest = components['schemas']['CreateTripRequest'];
export type UpdateTripRequest = components['schemas']['UpdateTripRequest'];
export type Confidence = 'high' | 'medium' | 'low';

export interface AuthStatus {
  loggedIn: boolean;
  email?: string;
}

// --- Auth ---

export async function getAuthStatus(): Promise<AuthStatus> {
  const res = await fetch(`${API_BASE}/auth/status`);
  return res.json();
}

export async function getLoginURL(): Promise<string> {
  const res = await fetch(`${API_BASE}/auth/login`);
  const data = await res.json();
  return data.url;
}

export async function logout(): Promise<void> {
  await fetch(`${API_BASE}/auth/logout`, { method: 'POST' });
}

// --- Activities ---

export async function listActivities(params?: { month?: string; from?: string; to?: string }): Promise<Activity[]> {
  const search = new URLSearchParams();
  if (params?.month) search.set('month', params.month);
  if (params?.from) search.set('from', params.from);
  if (params?.to) search.set('to', params.to);
  const qs = search.toString();
  const res = await fetch(`${API_BASE}/activities${qs ? '?' + qs : ''}`);
  if (!res.ok) throw new Error(`Failed to list activities: ${res.statusText}`);
  return res.json();
}

export async function getActivity(id: string): Promise<Activity> {
  const res = await fetch(`${API_BASE}/activities/${id}`);
  if (!res.ok) throw new Error(`Failed to get activity: ${res.statusText}`);
  return res.json();
}

export async function createActivity(req: CreateActivityRequest): Promise<Activity> {
  const res = await fetch(`${API_BASE}/activities`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export async function updateActivity(id: string, req: UpdateActivityRequest): Promise<Activity> {
  const res = await fetch(`${API_BASE}/activities/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export async function deleteActivity(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/activities/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(`Failed to delete activity: ${res.statusText}`);
}

export async function bulkDeleteActivities(ids: string[]): Promise<{ deleted: number }> {
  const res = await fetch(`${API_BASE}/activities/bulk-delete`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  });
  if (!res.ok) throw new Error(`Failed to delete activities: ${res.statusText}`);
  return res.json();
}

// --- Parse ---

export async function parseActivity(text: string): Promise<ParseResult> {
  const res = await fetch(`${API_BASE}/activities/parse`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  });
  if (!res.ok) throw new Error(`Failed to parse: ${res.statusText}`);
  return res.json();
}

// --- Check ---

export async function checkDate(date: string): Promise<DateCheck> {
  const res = await fetch(`${API_BASE}/activities/check/${date}`);
  if (!res.ok) throw new Error(`Failed to check date: ${res.statusText}`);
  return res.json();
}

// --- Trips ---

export async function listTrips(): Promise<TripSummary[]> {
  const res = await fetch(`${API_BASE}/trips`);
  if (!res.ok) throw new Error(`Failed to list trips: ${res.statusText}`);
  return res.json();
}

export async function createTrip(req: CreateTripRequest): Promise<Trip> {
  const res = await fetch(`${API_BASE}/trips`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export async function updateTrip(id: string, req: UpdateTripRequest): Promise<Trip> {
  const res = await fetch(`${API_BASE}/trips/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export async function deleteTrip(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/trips/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(`Failed to delete trip: ${res.statusText}`);
}

// --- Share Links ---

export type ShareLink = components['schemas']['ShareLink'];
export type CreateShareLinkRequest = components['schemas']['CreateShareLinkRequest'];

export async function listShareLinks(): Promise<ShareLink[]> {
  const res = await fetch(`${API_BASE}/share-links`);
  if (!res.ok) throw new Error(`Failed to list share links: ${res.statusText}`);
  return res.json();
}

export async function createShareLink(req: CreateShareLinkRequest): Promise<ShareLink> {
  const res = await fetch(`${API_BASE}/share-links`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export async function deleteShareLink(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/share-links/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(`Failed to delete share link: ${res.statusText}`);
}

// --- User-to-User Shares ---

export type Share = components['schemas']['Share'];
export type CreateShareRequest = components['schemas']['CreateShareRequest'];
export type SharedWithMeEntry = components['schemas']['SharedWithMeEntry'];

export async function listShares(): Promise<Share[]> {
  const res = await fetch(`${API_BASE}/shares`);
  if (!res.ok) throw new Error(`Failed to list shares: ${res.statusText}`);
  return res.json();
}

export async function createShare(req: CreateShareRequest): Promise<Share> {
  const res = await fetch(`${API_BASE}/shares`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export async function deleteShare(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/shares/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(`Failed to delete share: ${res.statusText}`);
}

export async function listSharedWithMe(): Promise<SharedWithMeEntry[]> {
  const res = await fetch(`${API_BASE}/shared-with-me`);
  if (!res.ok) throw new Error(`Failed to list shared calendars: ${res.statusText}`);
  return res.json();
}

// --- Places ---

export type Place = components['schemas']['Place'];
export type PlaceResolveResponse = components['schemas']['PlaceResolveResponse'];
export type PlaceSuggestion = components['schemas']['PlaceSuggestion'];
export type CreatePlaceRequest = components['schemas']['CreatePlaceRequest'];

export async function resolvePlaces(text: string): Promise<PlaceResolveResponse> {
  const res = await fetch(`${API_BASE}/places/resolve`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  });
  if (!res.ok) throw new Error(`Failed to resolve: ${res.statusText}`);
  return res.json();
}

export async function getPlace(id: string): Promise<Place> {
  const res = await fetch(`${API_BASE}/places/${id}`);
  if (!res.ok) throw new Error(`Failed to get place: ${res.statusText}`);
  return res.json();
}

export async function createPlace(req: CreatePlaceRequest): Promise<Place> {
  const res = await fetch(`${API_BASE}/places`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

// --- Public Profile ---

export type PublicProfile = components['schemas']['PublicProfile'];
export type UpdatePublicProfileRequest = components['schemas']['UpdatePublicProfileRequest'];

export async function getPublicProfile(): Promise<PublicProfile> {
  const res = await fetch(`${API_BASE}/public-profile`);
  if (!res.ok) throw new Error(`Failed to get public profile: ${res.statusText}`);
  return res.json();
}

export async function updatePublicProfile(req: UpdatePublicProfileRequest): Promise<PublicProfile> {
  const res = await fetch(`${API_BASE}/public-profile`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

// --- Shared Calendar (authenticated: view another user's calendar) ---

export async function fetchSharedWithMeCalendar(email: string): Promise<SharedCalendarResponse> {
  const res = await fetch(`${API_BASE}/shared-with-me/${encodeURIComponent(email)}/activities`);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export async function fetchSharedWithMeActivities(email: string): Promise<SharedCalendarResponse> {
  const res = await fetch(`${API_BASE}/shared-with-me/${encodeURIComponent(email)}/activities`);
  if (!res.ok) throw new Error(`Failed to fetch shared activities: ${res.statusText}`);
  return res.json();
}

// Overlay calendar: a shared calendar with its activities loaded
export interface OverlayCalendar {
  email: string;
  color: string;
  visible: boolean;
  activities: Activity[];
}

// --- Import Sources ---

export interface ImportSource {
  id: string;
  name: string;
  url: string;
  sourceType: string;
  filterConfig: string;
  lastSyncAt: string;
  status: string;
  newCount: number;
  importedCount: number;
  hiddenCount: number;
}

export interface StagedEvent {
  id: string;
  sourceId: string;
  sourceEventId: string;
  title: string;
  type: string;
  startDate: string;
  endDate: string;
  location: string;
  notes: string;
  state: 'new' | 'imported' | 'hidden';
  activityId: string;
}

export interface SyncResult {
  fetched: number;
  staged: number;
  updated: number;
  filtered: number;
}

export async function listSources(): Promise<ImportSource[]> {
  const res = await fetch(`${API_BASE}/sources`);
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

export async function createSource(name: string, url: string): Promise<{ source: ImportSource; syncResult: SyncResult }> {
  const res = await fetch(`${API_BASE}/sources`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, url }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export async function syncSource(id: string): Promise<SyncResult> {
  const res = await fetch(`${API_BASE}/sources/${id}/sync`, { method: 'POST' });
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

export async function deleteSource(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/sources/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(res.statusText);
}

export async function listStagedEvents(params?: { sourceId?: string; state?: string }): Promise<StagedEvent[]> {
  const search = new URLSearchParams();
  if (params?.sourceId) search.set('sourceId', params.sourceId);
  if (params?.state) search.set('state', params.state);
  const qs = search.toString();
  const res = await fetch(`${API_BASE}/staged${qs ? '?' + qs : ''}`);
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

export async function importStagedEvents(ids: string[]): Promise<{ imported: number }> {
  const res = await fetch(`${API_BASE}/staged/import`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  });
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

export async function hideStagedEvents(ids: string[]): Promise<{ hidden: number }> {
  const res = await fetch(`${API_BASE}/staged/hide`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  });
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

export async function unhideStagedEvents(ids: string[]): Promise<{ unhidden: number }> {
  const res = await fetch(`${API_BASE}/staged/unhide`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  });
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

// --- Source Filters ---

export interface SourceFilter {
  pattern: string;
  type: 'hide' | 'select' | 'include' | 'exclude'; // hide/select preferred, include/exclude legacy
  enabled: boolean;
  builtin: boolean;
}

export async function getGlobalFilters(): Promise<SourceFilter[]> {
  const res = await fetch(`${API_BASE}/filters`);
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

export async function updateGlobalFilters(filters: SourceFilter[]): Promise<SourceFilter[]> {
  const res = await fetch(`${API_BASE}/filters`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(filters),
  });
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

export async function applyGlobalFilters(): Promise<{ hidden: number; unhidden: number; selected: number }> {
  const res = await fetch(`${API_BASE}/filters/apply`, { method: 'POST' });
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

// --- Public Dashboard ---

export async function fetchPublicDashboard(handle: string): Promise<SharedCalendarResponse> {
  const res = await fetch(`/public/${handle}.json`);
  if (res.status === 404) throw new Error('Public profile not found');
  if (!res.ok) throw new Error(`Failed to load public dashboard: ${res.statusText}`);
  return res.json();
}

// --- Shared Calendar (public link view) ---

export interface SharedActivity {
  title?: string;
  type: ActivityType;
  startDate: string;
  endDate: string;
  location?: string;
  tripName?: string;
  tripColor?: string;
}

export interface SharedCalendarResponse {
  label: string;
  ownerEmail?: string;
  activities: SharedActivity[];
}

export async function fetchSharedCalendar(token: string): Promise<SharedCalendarResponse> {
  const res = await fetch(`/shared/${token}.json`);
  if (res.status === 410) throw new Error('This share link has expired');
  if (res.status === 404) throw new Error('Share link not found');
  if (!res.ok) throw new Error(`Failed to load shared calendar: ${res.statusText}`);
  return res.json();
}

// --- Helpers ---

export const ACTIVITY_COLORS: Record<ActivityType, string> = {
  travel: '#3b82f6',     // blue
  stay: '#22c55e',       // green
  conference: '#a855f7', // purple
  vacation: '#f59e0b',   // amber
  commitment: '#ef4444', // red
};

export const ACTIVITY_TYPES: ActivityType[] = ['travel', 'stay', 'conference', 'vacation', 'commitment'];
