<script lang="ts">
	import type { Item } from '@travel-calendar/shared';

	export let item: Item;
	export let onDelete: ((item: Item) => void) | undefined = undefined;
	export let onMove: ((item: Item) => void) | undefined = undefined;

	function getNights(checkIn: string | undefined, checkOut: string | undefined): number {
		if (!checkIn || !checkOut) return 0;
		const start = new Date(checkIn);
		const end = new Date(checkOut);
		return Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24));
	}

	$: nights = getNights(item.checkIn, item.checkOut);
</script>

<div class="item-card item-card-hotel bg-white rounded-lg shadow-sm p-4">
	<div class="flex items-start gap-3">
		<!-- Icon -->
		<div class="w-8 h-8 bg-purple-100 rounded-full flex items-center justify-center flex-shrink-0">
			<svg class="w-4 h-4 text-purple-600" fill="currentColor" viewBox="0 0 24 24">
				<path
					d="M7 14c1.66 0 3-1.34 3-3S8.66 8 7 8s-3 1.34-3 3 1.34 3 3 3zm0-4c.55 0 1 .45 1 1s-.45 1-1 1-1-.45-1-1 .45-1 1-1zm12-3h-8v8H3V5H1v15h2v-3h18v3h2v-9c0-2.21-1.79-4-4-4z"
				/>
			</svg>
		</div>

		<!-- Hotel info -->
		<div class="flex-1 min-w-0">
			<div class="flex items-center gap-2 mb-1">
				<span class="font-medium">{item.name || 'Hotel'}</span>
				{#if nights > 0}
					<span class="px-1.5 py-0.5 bg-purple-100 text-purple-700 text-xs rounded">
						{nights} night{nights === 1 ? '' : 's'}
					</span>
				{/if}
			</div>
			<div class="text-sm text-gray-600">
				{#if item.location}
					{item.location}
				{:else if item.checkIn}
					Check-in {item.checkIn}
				{/if}
			</div>
		</div>

		<!-- Confirmation -->
		<div class="text-right text-sm">
			{#if item.confirmation}
				<div class="font-mono text-xs">{item.confirmation}</div>
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
