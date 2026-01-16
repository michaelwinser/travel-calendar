<script lang="ts">
	import type { Item } from '@travel-calendar/shared';
	import FlightCard from './FlightCard.svelte';
	import HotelCard from './HotelCard.svelte';
	import TrainCard from './TrainCard.svelte';
	import EventCard from './EventCard.svelte';
	import DriveCard from './DriveCard.svelte';

	export let item: Item;
	export let onDelete: ((item: Item) => void) | undefined = undefined;
	export let onMove: ((item: Item) => void) | undefined = undefined;
</script>

{#if item.type === 'flight'}
	<FlightCard {item} {onDelete} {onMove} />
{:else if item.type === 'hotel'}
	<HotelCard {item} {onDelete} {onMove} />
{:else if item.type === 'train'}
	<TrainCard {item} {onDelete} {onMove} />
{:else if item.type === 'event'}
	<EventCard {item} {onDelete} {onMove} />
{:else if item.type === 'drive'}
	<DriveCard {item} {onDelete} {onMove} />
{:else}
	<!-- Fallback for unknown types -->
	<div class="bg-white rounded-lg shadow-sm border p-4">
		<div class="text-sm text-gray-500">Unknown item type: {item.type}</div>
	</div>
{/if}
