/**
 * Trips Store
 *
 * Reactive store for trip data following MVC pattern.
 * This is the single source of truth for trip state.
 */

import { writable, derived } from 'svelte/store';
import type { Trip, CreateTripRequest, UpdateTripRequest } from '@travel-calendar/shared';
import { api, type TripFilters } from '$lib/api';

function createTripsStore() {
	const { subscribe, set, update } = writable<Trip[]>([]);

	return {
		subscribe,

		/**
		 * Load trips from API with optional filters
		 */
		async load(filters?: TripFilters): Promise<void> {
			const trips = await api.trips.list(filters);
			set(trips);
		},

		/**
		 * Search trips by query string
		 */
		async search(query: string): Promise<void> {
			if (!query.trim()) {
				// If empty query, load all trips
				return this.load();
			}
			const trips = await api.trips.search(query);
			set(trips);
		},

		/**
		 * Create a new trip
		 */
		async create(input: CreateTripRequest): Promise<Trip> {
			const trip = await api.trips.create(input);
			update((trips) => [...trips, trip]);
			return trip;
		},

		/**
		 * Update an existing trip
		 */
		async update(id: string, input: UpdateTripRequest): Promise<Trip> {
			const updated = await api.trips.update(id, input);
			update((trips) => trips.map((t) => (t.id === id ? updated : t)));
			return updated;
		},

		/**
		 * Delete a trip
		 */
		async delete(id: string): Promise<void> {
			await api.trips.delete(id);
			update((trips) => trips.filter((t) => t.id !== id));
		},

		/**
		 * Merge source trip into target trip
		 * Source trip is deleted, all items moved to target
		 */
		async merge(sourceId: string, targetId: string): Promise<Trip> {
			const merged = await api.trips.merge(sourceId, targetId);
			// Remove source trip and update target
			update((trips) =>
				trips.filter((t) => t.id !== sourceId).map((t) => (t.id === targetId ? merged : t))
			);
			return merged;
		},

		/**
		 * Clear the store
		 */
		clear(): void {
			set([]);
		}
	};
}

export const trips = createTripsStore();

/**
 * Derived store for upcoming trips (start date in future)
 */
export const upcomingTrips = derived(trips, ($trips) => {
	const today = new Date();
	today.setHours(0, 0, 0, 0);

	return $trips
		.filter((trip) => {
			if (!trip.startDate) return false;
			const startDate = new Date(trip.startDate);
			return startDate >= today;
		})
		.sort((a, b) => {
			const dateA = a.startDate ? new Date(a.startDate).getTime() : Infinity;
			const dateB = b.startDate ? new Date(b.startDate).getTime() : Infinity;
			return dateA - dateB;
		});
});

/**
 * Derived store for past trips (end date in past)
 */
export const pastTrips = derived(trips, ($trips) => {
	const today = new Date();
	today.setHours(0, 0, 0, 0);

	return $trips
		.filter((trip) => {
			if (!trip.endDate) return false;
			const endDate = new Date(trip.endDate);
			return endDate < today;
		})
		.sort((a, b) => {
			const dateA = a.endDate ? new Date(a.endDate).getTime() : 0;
			const dateB = b.endDate ? new Date(b.endDate).getTime() : 0;
			return dateB - dateA; // Most recent first
		});
});

/**
 * Single trip store for detail views
 */
function createCurrentTripStore() {
	const { subscribe, set } = writable<Trip | null>(null);

	return {
		subscribe,

		/**
		 * Load a specific trip by ID (includes items)
		 */
		async load(id: string): Promise<Trip> {
			const trip = await api.trips.get(id);
			set(trip);
			return trip;
		},

		/**
		 * Clear the current trip
		 */
		clear(): void {
			set(null);
		}
	};
}

export const currentTrip = createCurrentTripStore();
