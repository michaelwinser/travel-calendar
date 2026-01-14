<script lang="ts">
	import { onMount } from 'svelte';
	import type { TripDayLocation } from '@travel-calendar/shared';
	import { api } from '$lib/api/client';
	import DayLocationRow from './DayLocationRow.svelte';
	import TagInput from '../ui/TagInput.svelte';

	export let tripId: string;
	export let startDate: string;
	export let endDate: string;
	export let onSave: () => void = () => {};

	// Map of date string (YYYY-MM-DD) to locations array
	let locationsByDate: Record<string, string[]> = {};
	let suggestions: string[] = [];
	let loading = true;
	let saving = false;
	let error: string | null = null;

	// Generate array of dates between start and end (inclusive)
	function generateDates(start: string, end: string): Date[] {
		const dates: Date[] = [];
		const startD = new Date(start + 'T00:00:00');
		const endD = new Date(end + 'T00:00:00');

		const current = new Date(startD);
		while (current <= endD) {
			dates.push(new Date(current));
			current.setDate(current.getDate() + 1);
		}
		return dates;
	}

	function formatDateKey(date: Date): string {
		return date.toISOString().split('T')[0];
	}

	// Load existing locations
	async function loadLocations() {
		loading = true;
		error = null;
		try {
			const tripLocations = await api.locations.getTripLocations(tripId);

			// Convert to our internal format
			locationsByDate = {};
			for (const loc of tripLocations) {
				if (loc.date && loc.locations) {
					locationsByDate[loc.date] = loc.locations;
				}
			}

			// Build suggestions from existing locations across all days
			const allLocations = new Set<string>();
			for (const locs of Object.values(locationsByDate)) {
				for (const loc of locs) {
					allLocations.add(loc);
				}
			}
			suggestions = Array.from(allLocations).sort();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load locations';
		} finally {
			loading = false;
		}
	}

	// Save locations to API
	async function saveLocations() {
		saving = true;
		error = null;
		try {
			// Convert to API format
			const locations: TripDayLocation[] = [];
			for (const [date, locs] of Object.entries(locationsByDate)) {
				if (locs.length > 0) {
					locations.push({ date, locations: locs });
				}
			}

			await api.locations.setTripLocations(tripId, { locations });
			onSave();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save locations';
		} finally {
			saving = false;
		}
	}

	function handleDayUpdate(dateKey: string, newLocations: string[]) {
		locationsByDate[dateKey] = newLocations;
		locationsByDate = locationsByDate; // Trigger reactivity

		// Update suggestions with any new locations
		const allLocations = new Set(suggestions);
		for (const loc of newLocations) {
			allLocations.add(loc);
		}
		suggestions = Array.from(allLocations).sort();
	}

	// Bulk set all days
	let bulkLocations: string[] = [];

	function handleBulkAdd(tag: string) {
		bulkLocations = [...bulkLocations, tag];
	}

	function handleBulkRemove(tag: string) {
		bulkLocations = bulkLocations.filter((l) => l !== tag);
	}

	function applyBulkLocations() {
		if (bulkLocations.length === 0) return;

		for (const date of dates) {
			const dateKey = formatDateKey(date);
			locationsByDate[dateKey] = [...bulkLocations];
		}
		locationsByDate = locationsByDate; // Trigger reactivity
		bulkLocations = [];
	}

	$: dates = generateDates(startDate, endDate);

	onMount(() => {
		loadLocations();
	});
</script>

<div class="location-editor">
	<div class="flex items-center justify-between mb-4">
		<h3 class="text-lg font-medium">Trip Locations</h3>
		<button
			type="button"
			class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
			on:click={saveLocations}
			disabled={saving || loading}
		>
			{saving ? 'Saving...' : 'Save Locations'}
		</button>
	</div>

	{#if error}
		<div class="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 rounded-md text-sm">
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="text-gray-500 py-8 text-center">Loading locations...</div>
	{:else}
		<!-- Bulk location setter -->
		<div class="mb-6 p-4 bg-gray-50 rounded-lg">
			<div class="flex items-center gap-2 mb-2">
				<span class="text-sm font-medium text-gray-700">Set all days:</span>
			</div>
			<div class="flex gap-2">
				<div class="flex-1">
					<TagInput
						tags={bulkLocations}
						{suggestions}
						placeholder="Enter locations for all days..."
						onAdd={handleBulkAdd}
						onRemove={handleBulkRemove}
					/>
				</div>
				<button
					type="button"
					class="px-3 py-2 text-sm bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 disabled:opacity-50"
					on:click={applyBulkLocations}
					disabled={bulkLocations.length === 0}
				>
					Apply
				</button>
			</div>
		</div>

		<!-- Day-by-day editor -->
		<div class="space-y-1 border rounded-lg p-4">
			{#each dates as date (formatDateKey(date))}
				<DayLocationRow
					{date}
					locations={locationsByDate[formatDateKey(date)] || []}
					{suggestions}
					onUpdate={(newLocs) => handleDayUpdate(formatDateKey(date), newLocs)}
				/>
			{/each}
		</div>
	{/if}
</div>
