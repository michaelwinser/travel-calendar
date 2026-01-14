<script lang="ts">
	import type { Item } from '@travel-calendar/shared';

	export let item: Item;
	export let onDelete: ((item: Item) => void) | undefined = undefined;
</script>

<div class="item-card item-card-train bg-white rounded-lg shadow-sm p-4">
	<div class="flex items-start gap-3">
		<!-- Icon -->
		<div
			class="w-8 h-8 bg-emerald-100 rounded-full flex items-center justify-center flex-shrink-0"
		>
			<svg class="w-4 h-4 text-emerald-600" fill="currentColor" viewBox="0 0 24 24">
				<path
					d="M12 2c-4 0-8 .5-8 4v9.5C4 17.43 5.57 19 7.5 19L6 20.5v.5h2.23l2-2H14l2 2h2v-.5L16.5 19c1.93 0 3.5-1.57 3.5-3.5V6c0-3.5-3.58-4-8-4zM7.5 17c-.83 0-1.5-.67-1.5-1.5S6.67 14 7.5 14s1.5.67 1.5 1.5S8.33 17 7.5 17zm3.5-7H6V6h5v4zm2 0V6h5v4h-5zm3.5 7c-.83 0-1.5-.67-1.5-1.5s.67-1.5 1.5-1.5 1.5.67 1.5 1.5-.67 1.5-1.5 1.5z"
				/>
			</svg>
		</div>

		<!-- Train info -->
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
