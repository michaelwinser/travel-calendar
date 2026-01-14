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
	LocationRangeSegment
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
	}
};

export { ApiError };
