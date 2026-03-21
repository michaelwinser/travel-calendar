<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import type { Trip, TripPurpose } from '@travel-calendar/shared';
	import { trips } from '$lib/stores';
	import { dayEntries } from '$lib/stores/dayEntries';
	import type { DayEntry } from '$lib/stores/dayEntries';
	import MonthRow from '$lib/components/calendar/MonthRow.svelte';
	import MonthGrid from '$lib/components/calendar/MonthGrid.svelte';
	import DayEntryPopup from '$lib/components/calendar/DayEntryPopup.svelte';
	import QuickEntry from '$lib/components/trip/QuickEntry.svelte';
	import type { ParsedTrip } from '$lib/utils/tripParser';

	let loading = true;
	let error: string | null = null;
	let scrollContainer: HTMLDivElement;
	let currentMonthElement: HTMLDivElement;

	// View mode: 'year' (scrolling rows) or 'month' (single MonthGrid)
	let viewMode: 'year' | 'month' = 'year';

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

	let quickEntryError: string | null = null;

	// Day entry popup state
	let popupDate: string | null = null;
	let popupX = 0;
	let popupY = 0;

	function handleDayClick(date: string, event: MouseEvent) {
		const rect = (event.target as HTMLElement).getBoundingClientRect();
		popupX = Math.min(rect.left, window.innerWidth - 300);
		popupY = Math.min(rect.bottom + 4, window.innerHeight - 250);
		popupDate = date;
	}

	function parseDayEntryText(text: string): { location: string; description?: string } {
		// Parse "dentist in Fairfield" → { location: "Fairfield", description: "dentist" }
		// Parse "NYC" → { location: "NYC" }
		// Parse "CTO club meeting in NYC" → { location: "NYC", description: "CTO club meeting" }
		const inMatch = text.match(/^(.+?)\s+in\s+(.+)$/i);
		if (inMatch) {
			return { description: inMatch[1].trim(), location: inMatch[2].trim() };
		}
		const atMatch = text.match(/^(.+?)\s+at\s+(.+)$/i);
		if (atMatch) {
			return { description: atMatch[1].trim(), location: atMatch[2].trim() };
		}
		return { location: text };
	}

	async function handleDayEntryCreate(e: CustomEvent<{ date: string; text: string }>) {
		const { date, text } = e.detail;
		const parsed = parseDayEntryText(text);
		try {
			await dayEntries.create({
				date,
				location: parsed.location,
				description: parsed.description
			});
		} catch (err) {
			quickEntryError = err instanceof Error ? err.message : 'Failed to create entry';
		}
	}

	async function handleDayEntryDelete(e: CustomEvent<{ id: string }>) {
		try {
			await dayEntries.delete(e.detail.id);
		} catch (err) {
			quickEntryError = err instanceof Error ? err.message : 'Failed to delete entry';
		}
	}

	$: popupEntries = popupDate ? ($dayEntries).filter((e: DayEntry) => e.date.startsWith(popupDate!)) : [];

	async function handleQuickEntry(parsed: ParsedTrip) {
		quickEntryError = null;
		try {
			await trips.create({
				name: parsed.name,
				purpose: parsed.purpose || 'other',
				startDate: parsed.startDate || undefined,
				endDate: parsed.endDate || undefined,
				notes: parsed.location ? `Location: ${parsed.location}` : undefined
			} as any);
		} catch (e) {
			quickEntryError = e instanceof Error ? e.message : 'Failed to create trip';
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
			// Load day entries for a wide range (±12 months)
			const now = new Date();
			const from = new Date(now.getFullYear() - 1, now.getMonth(), 1);
			const to = new Date(now.getFullYear() + 1, now.getMonth() + 1, 0);
			await dayEntries.load(from.toISOString().split('T')[0], to.toISOString().split('T')[0]);
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

		<!-- View mode toggle -->
		<div class="flex rounded overflow-hidden border">
			<button
				on:click={() => viewMode = 'year'}
				class="px-3 py-1 text-sm {viewMode === 'year' ? 'bg-blue-600 text-white' : 'bg-white text-gray-700 hover:bg-gray-100'}"
			>
				Year
			</button>
			<button
				on:click={() => viewMode = 'month'}
				class="px-3 py-1 text-sm {viewMode === 'month' ? 'bg-blue-600 text-white' : 'bg-white text-gray-700 hover:bg-gray-100'}"
			>
				Month
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
			href="/settings"
			class="px-2 py-1.5 text-sm rounded hover:bg-gray-100"
			title="Settings"
		>
			<svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
			</svg>
		</a>
	</div>
</header>

<main class="max-w-6xl mx-auto px-4 py-6 h-[calc(100vh-73px)] flex flex-col">
	<!-- Quick Entry -->
	<div class="mb-4 flex-shrink-0">
		<QuickEntry onSubmit={handleQuickEntry} />
		{#if quickEntryError}
			<div class="mt-2 p-2 bg-red-50 text-red-700 text-sm rounded">
				{quickEntryError}
				<button class="ml-2 text-red-500 hover:text-red-700" on:click={() => quickEntryError = null}>Dismiss</button>
			</div>
		{/if}
	</div>

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
	{:else if viewMode === 'year'}
		<!-- Scrollable months container (Year view) -->
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
	{:else}
		<!-- Scrollable month grids (Month view) -->
		<div
			bind:this={scrollContainer}
			on:scroll={handleScroll}
			class="flex-1 overflow-y-auto"
		>
			<div class="max-w-4xl mx-auto space-y-6 py-2">
				{#each months as { year, month, key } (key)}
					{#if isCurrentMonth(year, month)}
						<div bind:this={currentMonthElement}>
							<MonthGrid
								{year}
								{month}
								trips={$trips}
								dayEntries={$dayEntries}
								onTripClick={handleTripClick}
								onDayClick={handleDayClick}
							/>
						</div>
					{:else}
						<MonthGrid
							{year}
							{month}
							trips={$trips}
							dayEntries={$dayEntries}
							onTripClick={handleTripClick}
							onDayClick={handleDayClick}
						/>
					{/if}
				{/each}
			</div>
		</div>
	{/if}
</main>

{#if popupDate}
	<DayEntryPopup
		date={popupDate}
		entries={popupEntries}
		x={popupX}
		y={popupY}
		on:create={handleDayEntryCreate}
		on:delete={handleDayEntryDelete}
		on:close={() => popupDate = null}
	/>
{/if}
