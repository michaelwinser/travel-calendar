/**
 * API Client for Travel Calendar
 *
 * Typed fetch wrapper for the backend REST API.
 * Uses native fetch - works in both browser and SSR.
 */

import type {
	Trip,
	TripPurpose,
	CreateTripRequest,
	UpdateTripRequest,
	Item,
	CreateItemRequest,
	Document,
	ErrorResponse,
	BaseLocations,
	SetBaseLocationsRequest,
	TripDayLocation,
	SetTripLocationsRequest,
	LocationOnDateResponse,
	LocationRangeSegment,
	OAuthUrlResponse,
	GoogleAuthStatus,
	GoogleCalendar,
	UserCalendar,
	SetSelectedCalendarsRequest,
	CalendarEvent,
	TripSuggestion,
	MergeTripsRequest,
	MoveItemRequest,
	MoveItemResponse
} from '@travel-calendar/shared';

// In development, Vite proxies /api to localhost:3000
// In production, this would be configured via environment variable
const API_BASE = '';

class ApiError extends Error {
	constructor(
		public status: number,
		public statusText: string,
		public details?: ErrorResponse
	) {
		super(details?.error || statusText);
		this.name = 'ApiError';
	}
}

async function handleResponse<T>(response: Response): Promise<T> {
	if (!response.ok) {
		let details: ErrorResponse | undefined;
		try {
			details = await response.json();
		} catch {
			// Response body wasn't JSON
		}
		throw new ApiError(response.status, response.statusText, details);
	}

	// Handle 204 No Content
	if (response.status === 204) {
		return undefined as T;
	}

	return response.json();
}

function buildQueryString(params: Record<string, string | boolean | undefined>): string {
	const searchParams = new URLSearchParams();
	for (const [key, value] of Object.entries(params)) {
		if (value !== undefined) {
			searchParams.set(key, String(value));
		}
	}
	const qs = searchParams.toString();
	return qs ? `?${qs}` : '';
}

export interface TripFilters {
	[key: string]: string | boolean | undefined;
	upcoming?: boolean;
	past?: boolean;
	purpose?: TripPurpose;
}

export interface DocumentFilters {
	[key: string]: string | boolean | undefined;
	tripId?: string;
	unassociated?: boolean;
}

export const api = {
	auth: {
		async status(): Promise<{ loggedIn: boolean; email?: string; connected?: boolean }> {
			const response = await fetch(`${API_BASE}/api/auth/google/status`);
			return handleResponse(response);
		},

		async logout(): Promise<void> {
			const response = await fetch(`${API_BASE}/api/auth/logout`, { method: 'POST' });
			return handleResponse<void>(response);
		}
	},

	trips: {
		async list(filters?: TripFilters): Promise<Trip[]> {
			const qs = buildQueryString(filters || {});
			const response = await fetch(`${API_BASE}/api/trips${qs}`);
			return handleResponse<Trip[]>(response);
		},

		async search(q: string): Promise<Trip[]> {
			const qs = buildQueryString({ q });
			const response = await fetch(`${API_BASE}/api/trips/search${qs}`);
			return handleResponse<Trip[]>(response);
		},

		async get(id: string): Promise<Trip> {
			const response = await fetch(`${API_BASE}/api/trips/${id}`);
			return handleResponse<Trip>(response);
		},

		async create(input: CreateTripRequest): Promise<Trip> {
			const response = await fetch(`${API_BASE}/api/trips`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(input)
			});
			return handleResponse<Trip>(response);
		},

		async update(id: string, input: UpdateTripRequest): Promise<Trip> {
			const response = await fetch(`${API_BASE}/api/trips/${id}`, {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(input)
			});
			return handleResponse<Trip>(response);
		},

		async delete(id: string): Promise<void> {
			const response = await fetch(`${API_BASE}/api/trips/${id}`, {
				method: 'DELETE'
			});
			return handleResponse<void>(response);
		},

		async merge(sourceId: string, targetId: string, options?: MergeTripsRequest): Promise<Trip> {
			const response = await fetch(`${API_BASE}/api/trips/${sourceId}/merge/${targetId}`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(options || {})
			});
			return handleResponse<Trip>(response);
		}
	},

	items: {
		async list(tripId: string): Promise<Item[]> {
			const response = await fetch(`${API_BASE}/api/trips/${tripId}/items`);
			return handleResponse<Item[]>(response);
		},

		async create(tripId: string, input: CreateItemRequest): Promise<Item> {
			const response = await fetch(`${API_BASE}/api/trips/${tripId}/items`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(input)
			});
			return handleResponse<Item>(response);
		},

		async delete(id: string): Promise<void> {
			const response = await fetch(`${API_BASE}/api/items/${id}`, {
				method: 'DELETE'
			});
			return handleResponse<void>(response);
		},

		async move(itemId: string, request: MoveItemRequest): Promise<MoveItemResponse> {
			const response = await fetch(`${API_BASE}/api/items/${itemId}/move`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(request)
			});
			return handleResponse<MoveItemResponse>(response);
		}
	},

	documents: {
		async list(filters?: DocumentFilters): Promise<Document[]> {
			const qs = buildQueryString(filters || {});
			const response = await fetch(`${API_BASE}/api/documents${qs}`);
			return handleResponse<Document[]>(response);
		}
	},

	locations: {
		async getBaseLocations(): Promise<BaseLocations> {
			const response = await fetch(`${API_BASE}/api/config/locations`);
			return handleResponse<BaseLocations>(response);
		},

		async setBaseLocations(input: SetBaseLocationsRequest): Promise<BaseLocations> {
			const response = await fetch(`${API_BASE}/api/config/locations`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(input)
			});
			return handleResponse<BaseLocations>(response);
		},

		async getTripLocations(tripId: string): Promise<TripDayLocation[]> {
			const response = await fetch(`${API_BASE}/api/trips/${tripId}/locations`);
			return handleResponse<TripDayLocation[]>(response);
		},

		async setTripLocations(tripId: string, input: SetTripLocationsRequest): Promise<TripDayLocation[]> {
			const response = await fetch(`${API_BASE}/api/trips/${tripId}/locations`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(input)
			});
			return handleResponse<TripDayLocation[]>(response);
		},

		async getLocationOnDate(date: string): Promise<LocationOnDateResponse> {
			const response = await fetch(`${API_BASE}/api/location/on/${date}`);
			return handleResponse<LocationOnDateResponse>(response);
		},

		async getLocationRange(from: string, to: string): Promise<LocationRangeSegment[]> {
			const qs = buildQueryString({ from, to });
			const response = await fetch(`${API_BASE}/api/location/range${qs}`);
			return handleResponse<LocationRangeSegment[]>(response);
		}
	},

	calendar: {
		async getAuthUrl(scopes?: string): Promise<OAuthUrlResponse> {
			const qs = scopes ? buildQueryString({ scopes }) : '';
			const response = await fetch(`${API_BASE}/api/auth/google${qs}`);
			return handleResponse<OAuthUrlResponse>(response);
		},

		async getAuthStatus(): Promise<GoogleAuthStatus> {
			const response = await fetch(`${API_BASE}/api/auth/google/status`);
			return handleResponse<GoogleAuthStatus>(response);
		},

		async disconnect(): Promise<void> {
			const response = await fetch(`${API_BASE}/api/auth/google/disconnect`, {
				method: 'POST'
			});
			return handleResponse<void>(response);
		},

		async listCalendars(): Promise<GoogleCalendar[]> {
			const response = await fetch(`${API_BASE}/api/calendars`);
			return handleResponse<GoogleCalendar[]>(response);
		},

		async getSelectedCalendars(): Promise<UserCalendar[]> {
			const response = await fetch(`${API_BASE}/api/calendars/selected`);
			return handleResponse<UserCalendar[]>(response);
		},

		async setSelectedCalendars(input: SetSelectedCalendarsRequest): Promise<UserCalendar[]> {
			const response = await fetch(`${API_BASE}/api/calendars/selected`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(input)
			});
			return handleResponse<UserCalendar[]>(response);
		},

		async listEvents(from: string, to: string, calendarId?: string): Promise<CalendarEvent[]> {
			const params: Record<string, string | undefined> = { from, to };
			if (calendarId) params.calendarId = calendarId;
			const qs = buildQueryString(params);
			const response = await fetch(`${API_BASE}/api/calendar/events${qs}`);
			return handleResponse<CalendarEvent[]>(response);
		},

		async getTripSuggestions(from?: string, to?: string): Promise<TripSuggestion[]> {
			const params: Record<string, string | undefined> = {};
			if (from) params.from = from;
			if (to) params.to = to;
			const qs = buildQueryString(params);
			const response = await fetch(`${API_BASE}/api/calendar/trip-suggestions${qs}`);
			return handleResponse<TripSuggestion[]>(response);
		},

		async importSuggestion(suggestionId: string): Promise<Trip> {
			const response = await fetch(`${API_BASE}/api/calendar/trip-suggestions/${suggestionId}/import`, {
				method: 'POST'
			});
			return handleResponse<Trip>(response);
		},

		async dismissSuggestion(suggestionId: string): Promise<void> {
			const response = await fetch(`${API_BASE}/api/calendar/trip-suggestions/${suggestionId}/dismiss`, {
				method: 'POST'
			});
			return handleResponse<void>(response);
		},

		async mergeSuggestion(suggestionId: string, tripId: string): Promise<Trip> {
			const response = await fetch(`${API_BASE}/api/calendar/trip-suggestions/${suggestionId}/merge/${tripId}`, {
				method: 'POST'
			});
			return handleResponse<Trip>(response);
		},

		async resetProcessedEvents(): Promise<void> {
			const response = await fetch(`${API_BASE}/api/calendar/processed-events`, {
				method: 'DELETE'
			});
			return handleResponse<void>(response);
		}
	}
};

export { ApiError };
