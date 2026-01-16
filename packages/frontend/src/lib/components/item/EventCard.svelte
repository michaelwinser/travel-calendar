<script lang="ts">
	import type { Item } from '@travel-calendar/shared';

	export let item: Item;
	export let onDelete: ((item: Item) => void) | undefined = undefined;
	export let onMove: ((item: Item) => void) | undefined = undefined;
</script>

<div class="item-card item-card-event bg-white rounded-lg shadow-sm p-4">
	<div class="flex items-start gap-3">
		<!-- Icon -->
		<div class="w-8 h-8 bg-orange-100 rounded-full flex items-center justify-center flex-shrink-0">
			<svg class="w-4 h-4 text-orange-600" fill="currentColor" viewBox="0 0 24 24">
				<path
					d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM9 10H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2zm-8 4H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2z"
				/>
			</svg>
		</div>

		<!-- Event info -->
		<div class="flex-1 min-w-0">
			<div class="flex items-center gap-2 mb-1">
				<span class="font-medium">{item.name || 'Event'}</span>
			</div>
			<div class="text-sm text-gray-600">
				{#if item.time}
					{item.time}
				{/if}
				{#if item.location}
					{item.time ? ' · ' : ''}{item.location}
				{/if}
			</div>
			{#if item.notes}
				<div class="mt-2 text-sm text-gray-500">
					{item.notes}
				</div>
			{/if}
		</div>

		{#if onMove || onDelete}
			<div class="flex gap-1">
				{#if onMove}
					<button
						type="button"
						on:click={() => onMove?.(item)}
						class="text-gray-400 hover:text-blue-500"
						title="Move to another trip"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
							/>
						</svg>
					</button>
				{/if}
				{#if onDelete}
					<button
						type="button"
						on:click={() => onDelete?.(item)}
						class="text-gray-400 hover:text-red-500"
						title="Delete"
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
		{/if}
	</div>
</div>
