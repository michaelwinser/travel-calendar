<script lang="ts">
	import type { Item } from '@travel-calendar/shared';

	export let item: Item;
	export let onDelete: ((item: Item) => void) | undefined = undefined;
	export let onMove: ((item: Item) => void) | undefined = undefined;
</script>

<div class="item-card item-card-drive bg-white rounded-lg shadow-sm p-4">
	<div class="flex items-start gap-3">
		<!-- Icon -->
		<div class="w-8 h-8 bg-gray-100 rounded-full flex items-center justify-center flex-shrink-0">
			<svg class="w-4 h-4 text-gray-600" fill="currentColor" viewBox="0 0 24 24">
				<path
					d="M18.92 6.01C18.72 5.42 18.16 5 17.5 5h-11c-.66 0-1.21.42-1.42 1.01L3 12v8c0 .55.45 1 1 1h1c.55 0 1-.45 1-1v-1h12v1c0 .55.45 1 1 1h1c.55 0 1-.45 1-1v-8l-2.08-5.99zM6.5 16c-.83 0-1.5-.67-1.5-1.5S5.67 13 6.5 13s1.5.67 1.5 1.5S7.33 16 6.5 16zm11 0c-.83 0-1.5-.67-1.5-1.5s.67-1.5 1.5-1.5 1.5.67 1.5 1.5-.67 1.5-1.5 1.5zM5 11l1.5-4.5h11L19 11H5z"
				/>
			</svg>
		</div>

		<!-- Drive info -->
		<div class="flex-1 min-w-0">
			<div class="flex items-center gap-2 mb-1">
				<span class="font-medium">
					{#if item.from && item.to}
						{item.from} → {item.to}
					{:else}
						Drive
					{/if}
				</span>
			</div>
			<div class="text-sm text-gray-600">
				{#if item.time}
					Depart {item.time}
				{/if}
				{#if item.carrier}
					{item.time ? ' · ' : ''}{item.carrier}
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
