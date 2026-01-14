<script lang="ts">
	import type { Trip } from '@travel-calendar/shared';
	import TripBadge from './TripBadge.svelte';

	export let trip: Trip;
	export let onSelect: (trip: Trip) => void = () => {};

	// Format date for display
	function formatDate(dateStr: string | undefined): { day: string; month: string } {
		if (!dateStr) return { day: '--', month: '---' };
		const date = new Date(dateStr);
		return {
			day: date.getDate().toString(),
			month: date.toLocaleDateString('en-US', { month: 'short' })
		};
	}

	// Calculate days until/since trip
	function getDaysAway(dateStr: string | undefined): string {
		if (!dateStr) return '';
		const date = new Date(dateStr);
		const today = new Date();
		today.setHours(0, 0, 0, 0);
		date.setHours(0, 0, 0, 0);

		const diffTime = date.getTime() - today.getTime();
		const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

		if (diffDays === 0) return 'Today';
		if (diffDays === 1) return 'Tomorrow';
		if (diffDays > 0) return `In ${diffDays} days`;
		if (diffDays === -1) return 'Yesterday';
		return `${Math.abs(diffDays)} days ago`;
	}

	// Calculate trip duration
	function getDuration(startDate: string | undefined, endDate: string | undefined): string {
		if (!startDate || !endDate) return '';
		const start = new Date(startDate);
		const end = new Date(endDate);
		const diffTime = end.getTime() - start.getTime();
		const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24)) + 1;
		return `${diffDays} day${diffDays === 1 ? '' : 's'}`;
	}

	// Get status badge styling
	function getStatusStyle(status: string): string {
		switch (status) {
			case 'confirmed':
				return 'bg-green-100 text-green-700';
			case 'planning':
				return 'bg-yellow-100 text-yellow-700';
			case 'in_progress':
				return 'bg-blue-100 text-blue-700';
			case 'completed':
				return 'bg-gray-100 text-gray-500';
			case 'cancelled':
				return 'bg-red-100 text-red-700';
			default:
				return 'bg-gray-100 text-gray-600';
		}
	}

	function getStatusLabel(status: string): string {
		switch (status) {
			case 'in_progress':
				return 'In Progress';
			default:
				return status.charAt(0).toUpperCase() + status.slice(1);
		}
	}

	$: startDateParts = formatDate(trip.startDate);
	$: isPast = trip.endDate && new Date(trip.endDate) < new Date();
	$: itemCount = trip.items?.length ?? 0;
</script>

<button
	type="button"
	on:click={() => onSelect(trip)}
	class="block w-full text-left bg-white rounded-lg shadow-sm border p-4 hover:shadow-md transition-shadow {isPast
		? 'opacity-75'
		: ''}"
>
	<div class="flex items-start gap-4">
		<!-- Date display -->
		<div class="text-center min-w-[50px]">
			<div class="text-2xl font-bold {isPast ? 'text-gray-500' : 'text-gray-800'}">
				{startDateParts.day}
			</div>
			<div class="text-xs {isPast ? 'text-gray-400' : 'text-gray-500'}">
				{startDateParts.month}
			</div>
		</div>

		<!-- Trip info -->
		<div class="flex-1 min-w-0">
			<div class="flex items-center gap-2 mb-1">
				<h3 class="font-semibold {isPast ? 'text-gray-600' : ''}">
					{trip.name}
				</h3>
				<TripBadge purpose={trip.purpose} />
			</div>

			<p class="text-sm text-gray-600 mb-2">
				{#if trip.notes}
					{trip.notes.length > 50 ? trip.notes.slice(0, 50) + '...' : trip.notes}
				{:else if trip.startDate && trip.endDate}
					{getDuration(trip.startDate, trip.endDate)}
				{/if}
			</p>

			<!-- Item counts -->
			<div class="flex gap-4 text-xs {isPast ? 'text-gray-400' : 'text-gray-500'}">
				{#if itemCount > 0}
					<span class="flex items-center gap-1">
						<svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
							<path
								d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11z"
							/>
						</svg>
						{itemCount} item{itemCount === 1 ? '' : 's'}
					</span>
				{:else}
					<span class="text-gray-400 italic">No items yet</span>
				{/if}
			</div>
		</div>

		<!-- Status and countdown -->
		<div class="text-right">
			{#if trip.startDate}
				<div class="text-xs {isPast ? 'text-gray-400' : 'text-gray-500'}">
					{getDaysAway(trip.startDate)}
				</div>
			{/if}
			<div class="mt-1 px-2 py-0.5 text-xs rounded {getStatusStyle(trip.status)}">
				{getStatusLabel(trip.status)}
			</div>
		</div>
	</div>
</button>
