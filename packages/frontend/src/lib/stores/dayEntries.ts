/**
 * Day Entries Store
 *
 * Manages day-level location entries. Day entries are the atomic unit —
 * they can exist standalone or be associated with a trip.
 */

import { writable } from 'svelte/store';
import { api } from '../api/client';

export interface DayEntry {
	id: string;
	date: string;
	location: string;
	description?: string;
	tripId?: string;
	createdAt: string;
}

function createDayEntriesStore() {
	const { subscribe, set, update } = writable<DayEntry[]>([]);

	return {
		subscribe,

		async load(from: string, to: string): Promise<void> {
			const entries = await api.days.list(from, to);
			set(entries);
		},

		async create(input: { date: string; location: string; description?: string; tripId?: string }): Promise<DayEntry> {
			const entry = await api.days.create(input);
			update((entries) => [...entries, entry].sort((a, b) => a.date.localeCompare(b.date)));
			return entry;
		},

		async update(id: string, input: { location?: string; description?: string; tripId?: string }): Promise<DayEntry> {
			const updated = await api.days.update(id, input);
			update((entries) => entries.map((e) => (e.id === id ? updated : e)));
			return updated;
		},

		async delete(id: string): Promise<void> {
			await api.days.delete(id);
			update((entries) => entries.filter((e) => e.id !== id));
		},

		clear(): void {
			set([]);
		}
	};
}

export const dayEntries = createDayEntriesStore();
