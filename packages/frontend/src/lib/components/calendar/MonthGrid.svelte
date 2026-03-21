<script lang="ts">
	import type { Trip } from '@travel-calendar/shared';
	import type { DayEntry } from '$lib/stores/dayEntries';

	export let year: number;
	export let month: number; // 0-indexed (0 = January)
	export let trips: Trip[] = [];
	export let dayEntries: DayEntry[] = [];
	export let onTripClick: (trip: Trip) => void = () => {};
	export let onDayClick: (date: string, event: MouseEvent) => void = () => {};

	const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
	const monthNames = [
		'January', 'February', 'March', 'April', 'May', 'June',
		'July', 'August', 'September', 'October', 'November', 'December'
	];

	function getDaysInMonth(y: number, m: number): number {
		return new Date(y, m + 1, 0).getDate();
	}

	function isToday(y: number, m: number, d: number): boolean {
		const today = new Date();
		return today.getFullYear() === y && today.getMonth() === m && today.getDate() === d;
	}

	function formatDateKey(y: number, m: number, d: number): string {
		return `${y}-${String(m + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
	}

	interface CalendarDay {
		day: number;
		inMonth: boolean;
		year: number;
		month: number;
		dateKey: string;
	}

	interface WeekTripBar {
		trip: Trip;
		startCol: number;
		span: number;
		row: number;
	}

	interface CalendarWeek {
		days: CalendarDay[];
		tripBars: WeekTripBar[];
		maxRow: number;
	}

	// Build lookup of day entries by date
	function buildEntryMap(entries: DayEntry[]): Map<string, DayEntry[]> {
		const map = new Map<string, DayEntry[]>();
		for (const entry of entries) {
			const key = entry.date.split('T')[0]; // Handle ISO strings
			const list = map.get(key) || [];
			list.push(entry);
			map.set(key, list);
		}
		return map;
	}

	function buildCalendar(y: number, m: number): CalendarWeek[] {
		const daysInMonth = getDaysInMonth(y, m);
		const firstDow = new Date(y, m, 1).getDay();

		const allDays: CalendarDay[] = [];

		// Padding from previous month
		if (firstDow > 0) {
			const prevMonth = m === 0 ? 11 : m - 1;
			const prevYear = m === 0 ? y - 1 : y;
			const prevDays = getDaysInMonth(prevYear, prevMonth);
			for (let i = firstDow - 1; i >= 0; i--) {
				const day = prevDays - i;
				allDays.push({ day, inMonth: false, year: prevYear, month: prevMonth, dateKey: formatDateKey(prevYear, prevMonth, day) });
			}
		}

		for (let d = 1; d <= daysInMonth; d++) {
			allDays.push({ day: d, inMonth: true, year: y, month: m, dateKey: formatDateKey(y, m, d) });
		}

		const remaining = 7 - (allDays.length % 7);
		if (remaining < 7) {
			const nextMonth = m === 11 ? 0 : m + 1;
			const nextYear = m === 11 ? y + 1 : y;
			for (let d = 1; d <= remaining; d++) {
				allDays.push({ day: d, inMonth: false, year: nextYear, month: nextMonth, dateKey: formatDateKey(nextYear, nextMonth, d) });
			}
		}

		const weeks: CalendarWeek[] = [];
		for (let i = 0; i < allDays.length; i += 7) {
			const days = allDays.slice(i, i + 7);
			const tripBars = calculateWeekBars(days);
			const maxRow = Math.max(-1, ...tripBars.map(b => b.row));
			weeks.push({ days, tripBars, maxRow });
		}

		return weeks;
	}

	function calculateWeekBars(days: CalendarDay[]): WeekTripBar[] {
		const bars: WeekTripBar[] = [];
		const weekStart = new Date(days[0].year, days[0].month, days[0].day);
		const weekEnd = new Date(days[6].year, days[6].month, days[6].day);

		for (const trip of trips) {
			if (!trip.startDate || !trip.endDate) continue;

			const tripStart = new Date(trip.startDate);
			const tripEnd = new Date(trip.endDate);

			if (tripEnd < weekStart || tripStart > weekEnd) continue;

			let startCol = 0;
			let endCol = 6;

			for (let c = 0; c < 7; c++) {
				const cellDate = new Date(days[c].year, days[c].month, days[c].day);
				if (cellDate < tripStart) startCol = c + 1;
				if (cellDate <= tripEnd) endCol = c;
			}

			if (startCol > 6) continue;
			startCol = Math.max(0, startCol);
			endCol = Math.min(6, endCol);

			bars.push({ trip, startCol, span: endCol - startCol + 1, row: 0 });
		}

		bars.sort((a, b) => {
			if (a.startCol !== b.startCol) return a.startCol - b.startCol;
			return b.span - a.span;
		});

		const rowEnds: number[] = [];
		for (const bar of bars) {
			let row = 0;
			while (rowEnds[row] !== undefined && rowEnds[row] >= bar.startCol) {
				row++;
			}
			bar.row = row;
			rowEnds[row] = bar.startCol + bar.span - 1;
		}

		return bars;
	}

	$: weeks = buildCalendar(year, month);
	$: entryMap = buildEntryMap(dayEntries);
</script>

<div class="bg-white rounded-lg border overflow-hidden">
	<!-- Month header -->
	<div class="px-4 py-3 bg-gray-50 border-b">
		<h2 class="text-lg font-semibold text-gray-800">{monthNames[month]} {year}</h2>
	</div>

	<!-- Day-of-week header -->
	<div class="grid grid-cols-7 border-b bg-gray-50">
		{#each dayNames as name}
			<div class="px-2 py-2 text-xs font-medium text-gray-500 text-center">{name}</div>
		{/each}
	</div>

	<!-- Week rows -->
	{#each weeks as week}
		<div class="grid grid-cols-7 border-b last:border-b-0">
			{#each week.days as day}
				{@const dayEntryList = entryMap.get(day.dateKey) || []}
				<button
					type="button"
					class="border-r last:border-r-0 px-1.5 pt-1 pb-1 min-h-[4rem] text-left cursor-pointer hover:bg-blue-50/50 transition-colors
						{day.inMonth ? '' : 'bg-gray-50'}
						{day.inMonth && (new Date(day.year, day.month, day.day).getDay() === 0 || new Date(day.year, day.month, day.day).getDay() === 6) ? 'bg-gray-50/50' : ''}"
					on:click={(e) => day.inMonth && onDayClick(day.dateKey, e)}
				>
					<div class="text-right">
						{#if isToday(day.year, day.month, day.day)}
							<span class="inline-flex items-center justify-center w-6 h-6 bg-blue-500 text-white rounded-full text-xs font-medium">
								{day.day}
							</span>
						{:else}
							<span class="text-xs {day.inMonth ? 'text-gray-700' : 'text-gray-300'}">
								{day.day}
							</span>
						{/if}
					</div>
					<!-- Day entry locations -->
					{#if dayEntryList.length > 0}
						<div class="mt-0.5 space-y-0.5">
							{#each dayEntryList.slice(0, 2) as entry}
								<div class="text-[10px] leading-tight text-gray-500 truncate" title="{entry.location}{entry.description ? ': ' + entry.description : ''}">
									{entry.location}
								</div>
							{/each}
							{#if dayEntryList.length > 2}
								<div class="text-[10px] text-gray-400">+{dayEntryList.length - 2} more</div>
							{/if}
						</div>
					{/if}
				</button>
			{/each}

			<!-- Trip bars overlay for this week -->
			{#if week.tripBars.length > 0}
				<div class="col-span-7 relative px-0" style="height: {(week.maxRow + 1) * 22 + 2}px; margin-top: -2px;">
					{#each week.tripBars as bar}
						{@const leftPercent = (bar.startCol / 7) * 100}
						{@const widthPercent = (bar.span / 7) * 100}
						<button
							type="button"
							class="trip-bar trip-bar-{bar.trip.purpose} absolute text-xs truncate px-1.5"
							style="left: {leftPercent}%; width: {widthPercent}%; top: {bar.row * 22}px; height: 18px; line-height: 18px;"
							on:click|stopPropagation={() => onTripClick(bar.trip)}
							title="{bar.trip.name}"
						>
							{bar.trip.name}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	{/each}
</div>
