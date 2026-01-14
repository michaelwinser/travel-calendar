<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import type { Trip, TripPurpose } from '@travel-calendar/shared';
	import { trips } from '$lib/stores';
	import Header from '$lib/components/ui/Header.svelte';
	import MonthGrid from '$lib/components/calendar/MonthGrid.svelte';

	let currentYear = new Date().getFullYear();
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

	function handleTripClick(trip: Trip) {
		goto(`/trips/${trip.id}`);
	}

	function previousYear() {
		currentYear--;
	}

	function nextYear() {
		currentYear++;
	}

	function goToToday() {
		currentYear = new Date().getFullYear();
	}

	// Get all 12 months
	$: months = Array.from({ length: 12 }, (_, i) => i);

	// Purpose legend items
	const purposes: { key: TripPurpose; label: string }[] = [
		{ key: 'conference', label: 'Conference' },
		{ key: 'business', label: 'Business' },
		{ key: 'vacation', label: 'Vacation' },
		{ key: 'family', label: 'Family' },
		{ key: 'other', label: 'Other' }
	];
</script>

<svelte:head>
	<title>Calendar - Travel Calendar</title>
</svelte:head>

<header class="bg-white border-b sticky top-0 z-20">
	<div class="max-w-6xl mx-auto px-4 py-3 flex items-center gap-4">
		<h1 class="text-xl font-semibold">Travel Calendar</h1>
		<div class="flex-1"></div>

		<!-- Year navigation -->
		<div class="flex items-center gap-2">
			<button
				on:click={previousYear}
				class="px-3 py-1.5 text-sm hover:bg-gray-100 rounded"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
				</svg>
			</button>
			<span class="font-medium px-2 min-w-[60px] text-center">{currentYear}</span>
			<button
				on:click={nextYear}
				class="px-3 py-1.5 text-sm hover:bg-gray-100 rounded"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		</div>

		<button
			on:click={goToToday}
			class="px-3 py-1.5 text-sm bg-gray-100 hover:bg-gray-200 rounded"
		>
			Today
		</button>

		<a href="/trips" class="px-3 py-1.5 text-sm hover:bg-gray-100 rounded">
			All Trips
		</a>

		<a
			href="/trips/new"
			class="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded"
		>
			+ New Trip
		</a>
	</div>
</header>

<main class="max-w-6xl mx-auto px-4 py-6">
	<!-- Legend -->
	<div class="flex gap-4 mb-6 text-xs">
		{#each purposes as { key, label }}
			<div class="flex items-center gap-1.5">
				<span class="w-3 h-3 rounded trip-bar-{key}"></span>
				<span class="text-gray-600">{label}</span>
			</div>
		{/each}
	</div>

	{#if loading}
		<div class="text-center py-12">
			<p class="text-gray-500">Loading calendar...</p>
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
	{:else}
		<!-- Months grid -->
		<div class="space-y-6">
			{#each months as month}
				<MonthGrid
					year={currentYear}
					{month}
					trips={$trips}
					onTripClick={handleTripClick}
				/>
			{/each}
		</div>
	{/if}
</main>
