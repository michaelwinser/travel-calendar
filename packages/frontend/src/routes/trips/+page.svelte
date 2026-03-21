<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import type { Trip, TripPurpose } from '@travel-calendar/shared';
	import { trips, upcomingTrips, pastTrips } from '$lib/stores';
	import Header from '$lib/components/ui/Header.svelte';
	import TripCard from '$lib/components/trip/TripCard.svelte';

	let searchQuery = '';
	let purposeFilter: TripPurpose | '' = '';
	let timeFilter: 'upcoming' | 'past' | 'all' = 'all';
	let loading = true;
	let error: string | null = null;

	onMount(async () => {
		try {
			await trips.load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load trips';
		} finally {
			loading = false;
		}
	});

	async function handleSearch() {
		loading = true;
		error = null;
		try {
			if (searchQuery.trim()) {
				await trips.search(searchQuery);
			} else {
				await trips.load();
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Search failed';
		} finally {
			loading = false;
		}
	}

	function handleSelect(trip: Trip) {
		goto(`/trips/${trip.id}`);
	}

	// Filter trips based on current filters
	$: filteredTrips = (() => {
		let result: Trip[];

		// Apply time filter
		if (timeFilter === 'upcoming') {
			result = $upcomingTrips;
		} else if (timeFilter === 'past') {
			result = $pastTrips;
		} else {
			result = $trips;
		}

		// Apply purpose filter
		if (purposeFilter) {
			result = result.filter((t) => t.purpose === purposeFilter);
		}

		return result;
	})();
</script>

<svelte:head>
	<title>Trips - Travel Calendar</title>
</svelte:head>

<Header title="Trips" />

<main class="max-w-4xl mx-auto px-4 py-6">
	<!-- Filters -->
	<div class="flex gap-3 mb-6">
		<div class="flex-1 relative">
			<svg
				class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
				/>
			</svg>
			<input
				type="text"
				bind:value={searchQuery}
				on:keydown={(e) => e.key === 'Enter' && handleSearch()}
				placeholder="Search trips..."
				class="w-full pl-10 pr-4 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
			/>
		</div>

		<select
			bind:value={purposeFilter}
			class="px-3 py-2 border rounded-lg text-sm bg-white"
		>
			<option value="">All types</option>
			<option value="conference">Conference</option>
			<option value="business">Business</option>
			<option value="vacation">Vacation</option>
			<option value="family">Family</option>
			<option value="other">Other</option>
		</select>

		<select
			bind:value={timeFilter}
			class="px-3 py-2 border rounded-lg text-sm bg-white"
		>
			<option value="upcoming">Upcoming</option>
			<option value="past">Past</option>
			<option value="all">All</option>
		</select>
	</div>

	<!-- Loading state -->
	{#if loading}
		<div class="text-center py-12">
			<p class="text-gray-500">Loading trips...</p>
		</div>
	{:else if error}
		<div class="text-center py-12">
			<p class="text-red-500">{error}</p>
			<button
				on:click={() => trips.load()}
				class="mt-4 px-4 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-700"
			>
				Retry
			</button>
		</div>
	{:else if filteredTrips.length === 0}
		<div class="text-center py-12">
			<svg
				class="w-12 h-12 mx-auto mb-3 text-gray-300"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="1.5"
					d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
				/>
			</svg>
			<p class="text-gray-500">No trips found</p>
			<a
				href="/trips/new"
				class="inline-block mt-4 px-4 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-700"
			>
				Create your first trip
			</a>
		</div>
	{:else}
		<!-- Trip List -->
		<div class="space-y-3">
			{#if timeFilter === 'all' || timeFilter === 'upcoming'}
				{#if $upcomingTrips.length > 0 && (timeFilter === 'all' || timeFilter === 'upcoming')}
					<div class="text-xs font-medium text-gray-500 uppercase tracking-wide px-1 pt-4">
						Upcoming
					</div>
				{/if}
			{/if}

			{#each filteredTrips as trip (trip.id)}
				<TripCard {trip} onSelect={handleSelect} />
			{/each}
		</div>
	{/if}
</main>
