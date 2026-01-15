/**
 * Calendar Store
 *
 * Manages Google Calendar authentication state and selected calendars.
 */

import { writable, derived } from 'svelte/store';
import type { GoogleAuthStatus, GoogleCalendar, UserCalendar } from '@travel-calendar/shared';
import { api } from '../api/client';

interface CalendarState {
	authStatus: GoogleAuthStatus | null;
	availableCalendars: GoogleCalendar[];
	selectedCalendars: UserCalendar[];
	loading: boolean;
	error: string | null;
}

function createCalendarStore() {
	const { subscribe, set, update } = writable<CalendarState>({
		authStatus: null,
		availableCalendars: [],
		selectedCalendars: [],
		loading: false,
		error: null
	});

	return {
		subscribe,

		async loadAuthStatus() {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const status = await api.calendar.getAuthStatus();
				update((s) => ({ ...s, authStatus: status, loading: false }));
				return status;
			} catch (err) {
				const message = err instanceof Error ? err.message : 'Failed to load auth status';
				update((s) => ({ ...s, loading: false, error: message }));
				return null;
			}
		},

		async startOAuth() {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const result = await api.calendar.getAuthUrl();
				// Redirect to Google OAuth
				window.location.href = result.url;
			} catch (err) {
				const message = err instanceof Error ? err.message : 'Failed to start OAuth';
				update((s) => ({ ...s, loading: false, error: message }));
			}
		},

		async disconnect() {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				await api.calendar.disconnect();
				update((s) => ({
					...s,
					authStatus: { connected: false },
					availableCalendars: [],
					selectedCalendars: [],
					loading: false
				}));
			} catch (err) {
				const message = err instanceof Error ? err.message : 'Failed to disconnect';
				update((s) => ({ ...s, loading: false, error: message }));
			}
		},

		async loadCalendars() {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const [available, selected] = await Promise.all([
					api.calendar.listCalendars(),
					api.calendar.getSelectedCalendars()
				]);
				update((s) => ({
					...s,
					availableCalendars: available,
					selectedCalendars: selected,
					loading: false
				}));
			} catch (err) {
				const message = err instanceof Error ? err.message : 'Failed to load calendars';
				update((s) => ({ ...s, loading: false, error: message }));
			}
		},

		async loadSelectedCalendars() {
			try {
				const selected = await api.calendar.getSelectedCalendars();
				update((s) => ({ ...s, selectedCalendars: selected }));
				return selected;
			} catch (err) {
				const message = err instanceof Error ? err.message : 'Failed to load selected calendars';
				update((s) => ({ ...s, error: message }));
				return [];
			}
		},

		async setSelectedCalendars(calendarIds: string[]) {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const selected = await api.calendar.setSelectedCalendars({ calendarIds });
				update((s) => ({ ...s, selectedCalendars: selected, loading: false }));
				return selected;
			} catch (err) {
				const message = err instanceof Error ? err.message : 'Failed to save calendar selection';
				update((s) => ({ ...s, loading: false, error: message }));
				return null;
			}
		},

		clearError() {
			update((s) => ({ ...s, error: null }));
		}
	};
}

export const calendarStore = createCalendarStore();

// Derived store for connection status
export const isCalendarConnected = derived(
	calendarStore,
	($store) => $store.authStatus?.connected ?? false
);

// Derived store for selected calendar IDs
export const selectedCalendarIds = derived(calendarStore, ($store) =>
	$store.selectedCalendars.filter((c) => c.enabled).map((c) => c.calendarId)
);
