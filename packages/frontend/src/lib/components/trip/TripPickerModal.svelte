<script lang="ts">
	import type { Trip } from '@travel-calendar/shared';
	import { trips } from '$lib/stores';
	import { onMount } from 'svelte';

	export let excludeTripId: string | undefined = undefined;
	export let title = 'Select trip';
	export let onSelect: (trip: Trip) => void;
	export let onCancel: () => void;
	export let allowCreateNew = false;
	export let onCreateNew: (() => void) | undefined = undefined;

	let loading = true;
	let tripsToShow: Trip[] = [];

	onMount(async () => {
		await trips.load();
		loading = false;
	});

	// Filter and sort trips reactively
	$: {
		tripsToShow = $trips
			.filter((t) => t.id !== excludeTripId)
			.sort((a, b) => {
				// Sort by start date (future trips first), nulls last
				const dateA = a.startDate ? new Date(a.startDate).getTime() : Infinity;
				const dateB = b.startDate ? new Date(b.startDate).getTime() : Infinity;
				return dateA - dateB;
			});
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onCancel();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onCancel();
		}
	}

	function formatDateRange(trip: Trip): string {
		if (!trip.startDate && !trip.endDate) return '';
		if (trip.startDate && trip.endDate) {
			return `${trip.startDate} - ${trip.endDate}`;
		}
		return trip.startDate || trip.endDate || '';
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
	on:click={handleBackdropClick}
>
	<div class="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 max-h-[80vh] overflow-hidden">
		<div class="p-4 border-b">
			<h2 class="text-lg font-semibold text-gray-900">{title}</h2>
		</div>

		<div class="overflow-y-auto max-h-96 p-2">
			{#if loading}
				<p class="text-center py-8 text-gray-500">Loading trips...</p>
			{:else if tripsToShow.length === 0}
				<p class="text-center py-8 text-gray-500">No other trips available</p>
			{:else}
				<div class="space-y-1">
					{#each tripsToShow as trip (trip.id)}
						<button
							type="button"
							class="w-full text-left p-3 hover:bg-gray-100 rounded-lg flex items-center gap-3 transition-colors"
							on:click={() => onSelect(trip)}
						>
							<div class="flex-1 min-w-0">
								<div class="font-medium text-gray-900 truncate">{trip.name}</div>
								{#if formatDateRange(trip)}
									<div class="text-sm text-gray-500">{formatDateRange(trip)}</div>
								{/if}
							</div>
							<div class="flex-shrink-0">
								<span
									class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium
									{trip.purpose === 'business'
										? 'bg-blue-100 text-blue-800'
										: trip.purpose === 'vacation'
											? 'bg-green-100 text-green-800'
											: trip.purpose === 'conference'
												? 'bg-purple-100 text-purple-800'
												: trip.purpose === 'family'
													? 'bg-pink-100 text-pink-800'
													: 'bg-gray-100 text-gray-800'}"
								>
									{trip.purpose}
								</span>
							</div>
						</button>
					{/each}
				</div>
			{/if}

			{#if allowCreateNew && onCreateNew}
				<button
					type="button"
					class="w-full text-left p-3 hover:bg-gray-100 rounded-lg flex items-center gap-3 border-t mt-2 pt-3 transition-colors"
					on:click={onCreateNew}
				>
					<div
						class="flex-shrink-0 w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center"
					>
						<svg
							class="w-5 h-5 text-blue-600"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 4v16m8-8H4"
							/>
						</svg>
					</div>
					<span class="text-blue-600 font-medium">Create new trip</span>
				</button>
			{/if}
		</div>

		<div class="p-4 border-t flex justify-end">
			<button
				type="button"
				class="px-4 py-2 text-gray-600 hover:text-gray-800 font-medium transition-colors"
				on:click={onCancel}
			>
				Cancel
			</button>
		</div>
	</div>
</div>
