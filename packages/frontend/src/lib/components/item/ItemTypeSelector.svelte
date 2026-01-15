<script lang="ts">
	import type { ItemType } from '@travel-calendar/shared';

	export let onSelect: (type: ItemType) => void;

	const itemTypes: { type: ItemType; label: string; bgColor: string; iconColor: string }[] = [
		{ type: 'flight', label: 'Flight', bgColor: 'bg-blue-100 hover:bg-blue-200', iconColor: 'text-blue-600' },
		{ type: 'hotel', label: 'Hotel', bgColor: 'bg-purple-100 hover:bg-purple-200', iconColor: 'text-purple-600' },
		{ type: 'train', label: 'Train', bgColor: 'bg-emerald-100 hover:bg-emerald-200', iconColor: 'text-emerald-600' },
		{ type: 'drive', label: 'Drive', bgColor: 'bg-gray-100 hover:bg-gray-200', iconColor: 'text-gray-600' },
		{ type: 'event', label: 'Event', bgColor: 'bg-orange-100 hover:bg-orange-200', iconColor: 'text-orange-600' }
	];
</script>

<div class="grid grid-cols-5 gap-3">
	{#each itemTypes as { type, label, bgColor, iconColor }}
		<button
			type="button"
			class="flex flex-col items-center gap-2 p-3 rounded-lg {bgColor} transition-colors"
			on:click={() => onSelect(type)}
		>
			<div class="w-10 h-10 rounded-full bg-white/50 flex items-center justify-center">
				{#if type === 'flight'}
					<svg class="w-5 h-5 {iconColor}" fill="currentColor" viewBox="0 0 24 24">
						<path d="M21 16v-2l-8-5V3.5c0-.83-.67-1.5-1.5-1.5S10 2.67 10 3.5V9l-8 5v2l8-2.5V19l-2 1.5V22l3.5-1 3.5 1v-1.5L13 19v-5.5l8 2.5z" />
					</svg>
				{:else if type === 'hotel'}
					<svg class="w-5 h-5 {iconColor}" fill="currentColor" viewBox="0 0 24 24">
						<path d="M7 14c1.66 0 3-1.34 3-3S8.66 8 7 8s-3 1.34-3 3 1.34 3 3 3zm0-4c.55 0 1 .45 1 1s-.45 1-1 1-1-.45-1-1 .45-1 1-1zm12-3h-8v8H3V5H1v15h2v-3h18v3h2v-9c0-2.21-1.79-4-4-4z" />
					</svg>
				{:else if type === 'train'}
					<svg class="w-5 h-5 {iconColor}" fill="currentColor" viewBox="0 0 24 24">
						<path d="M12 2c-4 0-8 .5-8 4v9.5C4 17.43 5.57 19 7.5 19L6 20.5v.5h2.23l2-2H14l2 2h2v-.5L16.5 19c1.93 0 3.5-1.57 3.5-3.5V6c0-3.5-3.58-4-8-4zM7.5 17c-.83 0-1.5-.67-1.5-1.5S6.67 14 7.5 14s1.5.67 1.5 1.5S8.33 17 7.5 17zm3.5-7H6V6h5v4zm2 0V6h5v4h-5zm3.5 7c-.83 0-1.5-.67-1.5-1.5s.67-1.5 1.5-1.5 1.5.67 1.5 1.5-.67 1.5-1.5 1.5z" />
					</svg>
				{:else if type === 'drive'}
					<svg class="w-5 h-5 {iconColor}" fill="currentColor" viewBox="0 0 24 24">
						<path d="M18.92 6.01C18.72 5.42 18.16 5 17.5 5h-11c-.66 0-1.21.42-1.42 1.01L3 12v8c0 .55.45 1 1 1h1c.55 0 1-.45 1-1v-1h12v1c0 .55.45 1 1 1h1c.55 0 1-.45 1-1v-8l-2.08-5.99zM6.5 16c-.83 0-1.5-.67-1.5-1.5S5.67 13 6.5 13s1.5.67 1.5 1.5S7.33 16 6.5 16zm11 0c-.83 0-1.5-.67-1.5-1.5s.67-1.5 1.5-1.5 1.5.67 1.5 1.5-.67 1.5-1.5 1.5zM5 11l1.5-4.5h11L19 11H5z" />
					</svg>
				{:else if type === 'event'}
					<svg class="w-5 h-5 {iconColor}" fill="currentColor" viewBox="0 0 24 24">
						<path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM9 10H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2zm-8 4H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2z" />
					</svg>
				{/if}
			</div>
			<span class="text-sm font-medium {iconColor}">{label}</span>
		</button>
	{/each}
</div>
