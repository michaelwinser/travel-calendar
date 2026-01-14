<script lang="ts">
	import type { Trip } from '@travel-calendar/shared';

	export let year: number;
	export let month: number; // 0-indexed (0 = January)
	export let trips: Trip[] = [];
	export let onTripClick: (trip: Trip) => void = () => {};

	const monthNames = [
		'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
		'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'
	];

	// Get number of days in month
	function getDaysInMonth(year: number, month: number): number {
		return new Date(year, month + 1, 0).getDate();
	}

	// Check if a day is weekend (0 = Sunday, 6 = Saturday)
	function isWeekend(year: number, month: number, day: number): boolean {
		const dow = new Date(year, month, day).getDay();
		return dow === 0 || dow === 6;
	}

	// Check if a day is today
	function isToday(year: number, month: number, day: number): boolean {
		const today = new Date();
		return (
			today.getFullYear() === year &&
			today.getMonth() === month &&
			today.getDate() === day
		);
	}

	// Calculate trip bars for this month
	function calculateTripBars(trips: Trip[], year: number, month: number): TripBar[] {
		const daysInMonth = getDaysInMonth(year, month);
		const monthStart = new Date(year, month, 1);
		const monthEnd = new Date(year, month, daysInMonth);

		const bars: TripBar[] = [];

		for (const trip of trips) {
			if (!trip.startDate || !trip.endDate) continue;

			const tripStart = new Date(trip.startDate);
			const tripEnd = new Date(trip.endDate);

			// Skip if trip doesn't overlap with this month
			if (tripEnd < monthStart || tripStart > monthEnd) continue;

			// Calculate start and end day within this month
			const startDay = tripStart < monthStart ? 1 : tripStart.getDate();
			const endDay = tripEnd > monthEnd ? daysInMonth : tripEnd.getDate();

			// Calculate if trip continues from previous/to next month
			const continuesFromPrev = tripStart < monthStart;
			const continuesToNext = tripEnd > monthEnd;

			bars.push({
				trip,
				startDay,
				endDay,
				continuesFromPrev,
				continuesToNext
			});
		}

		// Sort bars by start day, then by duration (longer trips first)
		bars.sort((a, b) => {
			if (a.startDay !== b.startDay) return a.startDay - b.startDay;
			return (b.endDay - b.startDay) - (a.endDay - a.startDay);
		});

		// Assign rows to avoid overlap
		const rows: number[] = []; // tracks end day for each row
		for (const bar of bars) {
			let row = 0;
			while (rows[row] !== undefined && rows[row] >= bar.startDay) {
				row++;
			}
			bar.row = row;
			rows[row] = bar.endDay;
		}

		return bars;
	}

	interface TripBar {
		trip: Trip;
		startDay: number;
		endDay: number;
		continuesFromPrev: boolean;
		continuesToNext: boolean;
		row?: number;
	}

	$: daysInMonth = getDaysInMonth(year, month);
	$: days = Array.from({ length: daysInMonth }, (_, i) => i + 1);
	$: tripBars = calculateTripBars(trips, year, month);
	$: maxRow = Math.max(-1, ...tripBars.map(b => b.row ?? 0));
	$: barAreaHeight = maxRow >= 0 ? (maxRow + 1) * 24 + 4 : 0; // 24px per row + 4px padding
</script>

<div class="flex border-b border-gray-200 bg-white">
	<!-- Month/Year label column -->
	<div class="w-16 flex-shrink-0 py-2 px-2 border-r border-gray-100 flex flex-col justify-center">
		<div class="text-sm font-medium text-gray-700">{monthNames[month]}</div>
		<div class="text-xs text-gray-400">{year}</div>
	</div>

	<!-- Days and trip bars -->
	<div class="flex-1 py-2 px-2 min-w-0">
		<!-- Day numbers -->
		<div class="grid gap-px" style="grid-template-columns: repeat({daysInMonth}, minmax(0, 1fr));">
			{#each days as day}
				<div
					class="text-center text-xs {isWeekend(year, month, day) ? 'text-gray-300' : 'text-gray-400'}"
				>
					{#if isToday(year, month, day)}
						<span class="inline-flex items-center justify-center w-5 h-5 bg-blue-500 text-white rounded-full text-[10px]">
							{day}
						</span>
					{:else}
						{day}
					{/if}
				</div>
			{/each}
		</div>

		<!-- Trip bars area -->
		{#if barAreaHeight > 0}
			<div class="relative mt-1" style="height: {barAreaHeight}px;">
				{#each tripBars as bar}
					{@const widthPercent = ((bar.endDay - bar.startDay + 1) / daysInMonth) * 100}
					{@const leftPercent = ((bar.startDay - 1) / daysInMonth) * 100}
					<button
						type="button"
						class="trip-bar trip-bar-{bar.trip.purpose} absolute text-xs"
						style="left: {leftPercent}%; width: {widthPercent}%; top: {(bar.row || 0) * 24}px; height: 20px; line-height: 20px;"
						on:click={() => onTripClick(bar.trip)}
					>
						{#if bar.continuesFromPrev}
							<span class="mr-0.5">&larr;</span>
						{/if}
						<span class="truncate">{bar.trip.name}</span>
						{#if bar.continuesToNext}
							<span class="ml-0.5">&rarr;</span>
						{/if}
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
