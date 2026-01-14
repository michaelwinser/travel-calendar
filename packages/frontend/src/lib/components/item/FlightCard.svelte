<script lang="ts">
	import type { Item } from '@travel-calendar/shared';

	export let item: Item;
	export let onDelete: ((item: Item) => void) | undefined = undefined;
</script>

<div class="item-card item-card-flight bg-white rounded-lg shadow-sm p-4">
	<div class="flex items-start gap-3">
		<!-- Icon -->
		<div class="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center flex-shrink-0">
			<svg class="w-4 h-4 text-blue-600" fill="currentColor" viewBox="0 0 24 24">
				<path
					d="M21 16v-2l-8-5V3.5c0-.83-.67-1.5-1.5-1.5S10 2.67 10 3.5V9l-8 5v2l8-2.5V19l-2 1.5V22l3.5-1 3.5 1v-1.5L13 19v-5.5l8 2.5z"
				/>
			</svg>
		</div>

		<!-- Flight info -->
		<div class="flex-1 min-w-0">
			<div class="flex items-center gap-2 mb-1">
				<span class="font-medium">{item.from || '???'} → {item.to || '???'}</span>
				{#if item.carrier || item.flightNumber}
					<span class="text-xs text-gray-500">
						{item.carrier || ''} {item.flightNumber || ''}
					</span>
				{/if}
			</div>
			<div class="text-sm text-gray-600">
				{#if item.time}
					Depart {item.time}
				{/if}
			</div>
		</div>

		<!-- Confirmation -->
		<div class="text-right text-sm">
			{#if item.confirmation}
				<div class="text-gray-500">Confirmation</div>
				<div class="font-mono text-xs">{item.confirmation}</div>
			{/if}
		</div>

		{#if onDelete}
			<button
				type="button"
				on:click={() => onDelete?.(item)}
				class="text-gray-400 hover:text-red-500"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M6 18L18 6M6 6l12 12"
					/>
				</svg>
			</button>
		{/if}
	</div>
</div>
