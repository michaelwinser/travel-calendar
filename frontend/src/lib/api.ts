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

// --- Helpers ---

export const ACTIVITY_COLORS: Record<ActivityType, string> = {
  travel: '#3b82f6',     // blue
  stay: '#22c55e',       // green
  conference: '#a855f7', // purple
  vacation: '#f59e0b',   // amber
  commitment: '#ef4444', // red
};

export const ACTIVITY_TYPES: ActivityType[] = ['travel', 'stay', 'conference', 'vacation', 'commitment'];
