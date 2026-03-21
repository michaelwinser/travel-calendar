<script lang="ts">
	import { onMount } from 'svelte';
	import type { CalendarEvent } from '@travel-calendar/shared';
	import { api } from '$lib/api/client';
	import { dayEntries } from '$lib/stores/dayEntries';
	import type { DayEntry } from '$lib/stores/dayEntries';

	export let tripId: string;
	export let startDate: string;
	export let endDate: string;

	let events: CalendarEvent[] = [];
	let loading = true;
	let error: string | null = null;
	let tripDayEntries: DayEntry[] = [];

	// Conflict: event with a location that doesn't match the trip's day entry location
	interface EventWithConflict {
		event: CalendarEvent;
		conflict: boolean;
		conflictReason?: string;
	}

	onMount(async () => {
		try {
			// Load calendar events for the trip's date range
			events = await api.calendar.listEvents(startDate, endDate);

			// Load day entries for this trip's range
			const allEntries = await api.days.list(startDate, endDate);
			tripDayEntries = allEntries.filter((e: DayEntry) => e.tripId === tripId || !e.tripId);
		} catch (e) {
			if (e instanceof Error && e.message.includes('not configured')) {
				// Calendar not connected — not an error
				events = [];
			} else {
				error = e instanceof Error ? e.message : 'Failed to load calendar events';
			}
		} finally {
			loading = false;
		}
	});

	function getEventDate(event: CalendarEvent): string {
		return event.start.split('T')[0];
	}

	function getTripLocationOnDate(dateStr: string): string | null {
		const entry = tripDayEntries.find((e) => e.date.split('T')[0] === dateStr);
		return entry?.location || null;
	}

	function checkConflict(event: CalendarEvent): EventWithConflict {
		if (!event.location) {
			return { event, conflict: false };
		}

		const eventDate = getEventDate(event);
		const tripLocation = getTripLocationOnDate(eventDate);

		if (!tripLocation) {
			return { event, conflict: false };
		}

		// Simple heuristic: if event location doesn't contain the trip location
		// (or vice versa), it's a potential conflict
		const eventLoc = event.location.toLowerCase();
		const tripLoc = tripLocation.toLowerCase();

		if (!eventLoc.includes(tripLoc) && !tripLoc.includes(eventLoc)) {
			return {
				event,
				conflict: true,
				conflictReason: `This event is in "${event.location}" but you'll be in "${tripLocation}"`
			};
		}

		return { event, conflict: false };
	}

	function formatEventTime(dateStr: string): string {
		const d = new Date(dateStr);
		return d.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
	}

	function formatEventDate(dateStr: string): string {
		const d = new Date(dateStr);
		return d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
	}

	$: eventsWithConflicts = events.map(checkConflict);
	$: conflicts = eventsWithConflicts.filter((e) => e.conflict);
	$: nonConflicts = eventsWithConflicts.filter((e) => !e.conflict);
</script>

{#if loading}
	<p class="text-sm text-gray-500 py-2">Loading calendar events...</p>
{:else if error}
	<p class="text-sm text-red-500 py-2">{error}</p>
{:else if events.length === 0}
	<p class="text-sm text-gray-500 py-2">No calendar events during this trip. Connect Google Calendar in Settings to see events here.</p>
{:else}
	<!-- Conflicts first -->
	{#if conflicts.length > 0}
		<div class="mb-4">
			<h4 class="text-sm font-medium text-red-700 mb-2 flex items-center gap-1.5">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
				</svg>
				Location Conflicts ({conflicts.length})
			</h4>
			<div class="space-y-2">
				{#each conflicts as { event, conflictReason }}
					<div class="p-3 bg-red-50 border border-red-100 rounded-lg">
						<div class="flex items-start justify-between">
							<div>
								<div class="text-sm font-medium text-gray-900">{event.summary}</div>
								<div class="text-xs text-gray-500 mt-0.5">
									{formatEventDate(event.start)} at {formatEventTime(event.start)}
								</div>
								{#if event.location}
									<div class="text-xs text-gray-600 mt-0.5">{event.location}</div>
								{/if}
							</div>
						</div>
						{#if conflictReason}
							<div class="mt-2 text-xs text-red-600 flex items-center gap-1">
								<svg class="w-3 h-3 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
									<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
								</svg>
								{conflictReason}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Other events -->
	{#if nonConflicts.length > 0}
		<div class="space-y-1.5">
			{#each nonConflicts as { event }}
				<div class="flex items-center gap-3 px-3 py-2 bg-gray-50 rounded text-sm">
					<div class="flex-1">
						<span class="text-gray-900">{event.summary}</span>
						{#if event.location}
							<span class="text-gray-400"> — {event.location}</span>
						{/if}
					</div>
					<span class="text-xs text-gray-500 flex-shrink-0">
						{formatEventDate(event.start)} {formatEventTime(event.start)}
					</span>
				</div>
			{/each}
		</div>
	{/if}
{/if}
