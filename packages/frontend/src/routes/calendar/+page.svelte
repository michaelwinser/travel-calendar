<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import type { Trip, TripPurpose } from '@travel-calendar/shared';
	import { trips } from '$lib/stores';
	import MonthRow from '$lib/components/calendar/MonthRow.svelte';

	let loading = true;
	let error: string | null = null;
	let scrollContainer: HTMLDivElement;
	let currentMonthElement: HTMLDivElement;

	// The actual list of months - grows as user scrolls, never shrinks
	let months: { year: number; month: number; key: string }[] = [];

	// Guard to prevent concurrent operations
	let isExpanding = false;

	// Helper to create a month key
	function monthKey(year: number, month: number): string {
		return `${year}-${month}`;
	}

	// Helper to add/subtract months from a date
	function addMonths(year: number, month: number, delta: number): { year: number; month: number } {
		let newMonth = month + delta;
		let newYear = year;
		while (newMonth < 0) {
			newMonth += 12;
			newYear--;
		}
		while (newMonth > 11) {
			newMonth -= 12;
			newYear++;
		}
		return { year: newYear, month: newMonth };
	}

	// Initialize months list centered on today
	function initializeMonths() {
		const today = new Date();
		const centerYear = today.getFullYear();
		const centerMonth = today.getMonth();

		months = [];
		// Start 12 months before today, go 12 months after (25 total)
		for (let i = -12; i <= 12; i++) {
			const { year, month } = addMonths(centerYear, centerMonth, i);
			months.push({ year, month, key: monthKey(year, month) });
		}
	}

	// Check if a month is the current month (today)
	function isCurrentMonth(year: number, month: number): boolean {
		const today = new Date();
		return year === today.getFullYear() && month === today.getMonth();
	}

	// Handle scroll - check if we need more months
	function handleScroll() {
		if (!scrollContainer || isExpanding) return;

		const { scrollTop, scrollHeight, clientHeight } = scrollContainer;
		const scrollBottom = scrollHeight - scrollTop - clientHeight;

		// If near top, prepend more months
		if (scrollTop < 500) {
			prependMonths();
		}

		// If near bottom, append more months
		if (scrollBottom < 500) {
			appendMonths();
		}
	}

	async function prependMonths() {
		if (isExpanding || months.length === 0) return;
		isExpanding = true;

		try {
			// Get the earliest month in our list
			const first = months[0];
			const newMonths: { year: number; month: number; key: string }[] = [];

			// Add 6 months before the first
			for (let i = 6; i >= 1; i--) {
				const { year, month } = addMonths(first.year, first.month, -i);
				newMonths.push({ year, month, key: monthKey(year, month) });
			}

			// Save scroll state before DOM update
			const oldScrollHeight = scrollContainer.scrollHeight;
			const oldScrollTop = scrollContainer.scrollTop;

			// Prepend to the array
			months = [...newMonths, ...months];

			// Wait for DOM to update
			await tick();

			// Adjust scroll position to compensate for added content above
			const newScrollHeight = scrollContainer.scrollHeight;
			const addedHeight = newScrollHeight - oldScrollHeight;
			scrollContainer.scrollTop = oldScrollTop + addedHeight;

			// Wait for scroll to settle
			await new Promise(resolve => setTimeout(resolve, 100));
		} finally {
			isExpanding = false;
		}
	}

	async function appendMonths() {
		if (isExpanding || months.length === 0) return;
		isExpanding = true;

		try {
			// Get the latest month in our list
			const last = months[months.length - 1];
			const newMonths: { year: number; month: number; key: string }[] = [];

			// Add 6 months after the last
			for (let i = 1; i <= 6; i++) {
				const { year, month } = addMonths(last.year, last.month, i);
				newMonths.push({ year, month, key: monthKey(year, month) });
			}

			// Append to the array
			months = [...months, ...newMonths];

			await tick();
			await new Promise(resolve => setTimeout(resolve, 100));
		} finally {
			isExpanding = false;
		}
	}

	function handleTripClick(trip: Trip) {
		goto(`/trips/${trip.id}`);
	}

	async function goToToday() {
		// Scroll to the current month element
		if (currentMonthElement) {
			currentMonthElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
		}
	}

	// Purpose legend items
	const purposes: { key: TripPurpose; label: string }[] = [
		{ key: 'conference', label: 'Conference' },
		{ key: 'business', label: 'Business' },
		{ key: 'vacation', label: 'Vacation' },
		{ key: 'family', label: 'Family' },
		{ key: 'other', label: 'Other' }
	];

	onMount(async () => {
		// Initialize the months list
		initializeMonths();

		try {
			await trips.load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load trips';
		} finally {
			loading = false;
		}

		// After initial render, ensure enough content to enable scrolling
		await tick();

		// On large monitors, initial 25 months may not fill the container.
		// Keep loading until content exceeds container height (scrollbar appears).
		if (scrollContainer) {
			while (scrollContainer.scrollHeight <= scrollContainer.clientHeight) {
				const prevLength = months.length;
				await appendMonths();
				await prependMonths();
				// Safety: prevent infinite loop if months stop growing
				if (months.length === prevLength) break;
			}
		}

		// Scroll to current month
		if (currentMonthElement) {
			currentMonthElement.scrollIntoView({ block: 'center' });
		}
	});
</script>

<svelte:head>
	<title>Calendar - Travel Calendar</title>
</svelte:head>

<header class="bg-white border-b sticky top-0 z-20">
	<div class="max-w-6xl mx-auto px-4 py-3 flex items-center gap-4">
		<h1 class="text-xl font-semibold">Travel Calendar</h1>
		<div class="flex-1"></div>

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

<main class="max-w-6xl mx-auto px-4 py-6 h-[calc(100vh-73px)] flex flex-col">
	<!-- Legend -->
	<div class="flex gap-4 mb-4 text-xs flex-shrink-0">
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
		<!-- Scrollable months container -->
		<div
			bind:this={scrollContainer}
			on:scroll={handleScroll}
			class="flex-1 overflow-y-auto border rounded-lg"
		>
			{#each months as { year, month, key } (key)}
				{#if isCurrentMonth(year, month)}
					<div bind:this={currentMonthElement}>
						<MonthRow
							{year}
							{month}
							trips={$trips}
							onTripClick={handleTripClick}
						/>
					</div>
				{:else}
					<MonthRow
						{year}
						{month}
						trips={$trips}
						onTripClick={handleTripClick}
					/>
				{/if}
			{/each}
		</div>
	{/if}
</main>
