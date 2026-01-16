<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import type { Item, Trip } from '@travel-calendar/shared';
	import { currentTrip, trips } from '$lib/stores';
	import { api } from '$lib/api';
	import Header from '$lib/components/ui/Header.svelte';
	import TripBadge from '$lib/components/trip/TripBadge.svelte';
	import ItemCard from '$lib/components/item/ItemCard.svelte';
	import ItemForm from '$lib/components/item/ItemForm.svelte';
	import LocationEditor from '$lib/components/location/LocationEditor.svelte';
	import TripPickerModal from '$lib/components/trip/TripPickerModal.svelte';

	let loading = true;
	let error: string | null = null;
	let deleting = false;
	let showLocations = false;
	let showAddItem = false;
	let showMergeModal = false;
	let showMoveModal = false;
	let itemToMove: Item | null = null;

	$: tripId = $page.params.id;

	onMount(async () => {
		try {
			await currentTrip.load(tripId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load trip';
		} finally {
			loading = false;
		}
	});

	async function handleDelete() {
		if (!$currentTrip) return;
		if (!confirm('Are you sure you want to delete this trip?')) return;

		deleting = true;
		try {
			await api.trips.delete($currentTrip.id);
			goto('/trips');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete trip';
			deleting = false;
		}
	}

	async function handleDeleteItem(item: Item) {
		if (!confirm('Delete this item?')) return;
		try {
			await api.items.delete(item.id);
			// Reload the trip to get updated items
			await currentTrip.load(tripId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete item';
		}
	}

	async function handleItemAdded() {
		// Reload the trip to get updated items
		await currentTrip.load(tripId);
		showAddItem = false;
	}

	async function handleMerge(targetTrip: Trip) {
		if (!$currentTrip) return;
		if (!confirm(`Merge "${$currentTrip.name}" into "${targetTrip.name}"? All items will be moved and this trip will be deleted.`)) {
			showMergeModal = false;
			return;
		}

		try {
			await trips.merge($currentTrip.id, targetTrip.id);
			goto(`/trips/${targetTrip.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to merge trips';
			showMergeModal = false;
		}
	}

	function handleMoveItem(item: Item) {
		itemToMove = item;
		showMoveModal = true;
	}

	async function handleMoveToTrip(targetTrip: Trip) {
		if (!itemToMove) return;

		try {
			await api.items.move(itemToMove.id, { targetTripId: targetTrip.id });
			// Reload the current trip to reflect the moved item
			await currentTrip.load(tripId);
			showMoveModal = false;
			itemToMove = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to move item';
			showMoveModal = false;
			itemToMove = null;
		}
	}

	// Group items by date
	function groupItemsByDate(items: Item[]): Map<string, Item[]> {
		const groups = new Map<string, Item[]>();

		// Sort items by date and time
		const sorted = [...items].sort((a, b) => {
			const dateA = a.date || a.checkIn || '';
			const dateB = b.date || b.checkIn || '';
			if (dateA !== dateB) return dateA.localeCompare(dateB);
			const timeA = a.time || '00:00';
			const timeB = b.time || '00:00';
			return timeA.localeCompare(timeB);
		});

		for (const item of sorted) {
			const date = item.date || item.checkIn || 'No date';
			if (!groups.has(date)) {
				groups.set(date, []);
			}
			groups.get(date)!.push(item);
		}

		return groups;
	}

	function formatDateHeader(dateStr: string): { day: string; weekday: string; monthYear: string } {
		if (dateStr === 'No date') {
			return { day: '--', weekday: '', monthYear: '' };
		}
		const date = new Date(dateStr);
		return {
			day: date.getDate().toString(),
			weekday: date.toLocaleDateString('en-US', { weekday: 'short' }),
			monthYear: date.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })
		};
	}

	function formatDateRange(start: string | undefined, end: string | undefined): string {
		if (!start) return '';
		const startDate = new Date(start);
		const endDate = end ? new Date(end) : null;

		const startStr = startDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
		if (!endDate) return startStr;

		const endStr = endDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
		return `${startStr} – ${endStr}`;
	}

	function getDuration(start: string | undefined, end: string | undefined): string {
		if (!start || !end) return '';
		const startDate = new Date(start);
		const endDate = new Date(end);
		const days = Math.ceil((endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60 * 24)) + 1;
		return `${days} day${days === 1 ? '' : 's'}`;
	}

	function getStatusLabel(status: string): string {
		switch (status) {
			case 'in_progress':
				return 'In Progress';
			default:
				return status.charAt(0).toUpperCase() + status.slice(1);
		}
	}

	function getStatusColor(status: string): string {
		switch (status) {
			case 'confirmed':
				return 'text-green-600';
			case 'planning':
				return 'text-yellow-600';
			case 'in_progress':
				return 'text-blue-600';
			case 'completed':
				return 'text-gray-500';
			case 'cancelled':
				return 'text-red-600';
			default:
				return 'text-gray-600';
		}
	}

	$: itemGroups = $currentTrip?.items ? groupItemsByDate($currentTrip.items) : new Map();
</script>

<svelte:head>
	<title>{$currentTrip?.name || 'Trip'} - Travel Calendar</title>
</svelte:head>

{#if loading}
	<Header title="Loading..." showBackButton backHref="/trips" />
	<main class="max-w-4xl mx-auto px-4 py-6">
		<div class="text-center py-12">
			<p class="text-gray-500">Loading trip...</p>
		</div>
	</main>
{:else if error}
	<Header title="Error" showBackButton backHref="/trips" />
	<main class="max-w-4xl mx-auto px-4 py-6">
		<div class="text-center py-12">
			<p class="text-red-500">{error}</p>
			<a
				href="/trips"
				class="inline-block mt-4 px-4 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-700"
			>
				Back to trips
			</a>
		</div>
	</main>
{:else if $currentTrip}
	<header class="bg-white border-b sticky top-0 z-10">
		<div class="max-w-4xl mx-auto px-4 py-3 flex items-center gap-4">
			<a href="/trips" class="text-gray-500 hover:text-gray-700">
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M15 19l-7-7 7-7"
					/>
				</svg>
			</a>
			<div class="flex-1">
				<h1 class="font-semibold text-lg">{$currentTrip.name}</h1>
				<p class="text-sm text-gray-500">
					{formatDateRange($currentTrip.startDate, $currentTrip.endDate)}
				</p>
			</div>
			<TripBadge purpose={$currentTrip.purpose} />
			<button
				type="button"
				class="text-gray-400 hover:text-gray-600 p-2"
				on:click={() => (showMergeModal = true)}
				title="Merge with another trip"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
					/>
				</svg>
			</button>
			<a
				href="/trips/{$currentTrip.id}/edit"
				class="text-gray-400 hover:text-gray-600 p-2"
				title="Edit trip"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
					/>
				</svg>
			</a>
			<button
				type="button"
				class="text-gray-400 hover:text-gray-600 p-2"
				on:click={handleDelete}
				disabled={deleting}
				title="Delete trip"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
					/>
				</svg>
			</button>
		</div>
	</header>

	<main class="max-w-4xl mx-auto px-4 py-6">
		<!-- Trip Summary -->
		<div class="bg-white rounded-lg shadow-sm border p-4 mb-6">
			<div class="flex items-start gap-4">
				<div class="flex-1">
					{#if $currentTrip.notes}
						<p class="text-gray-600 mb-3">{$currentTrip.notes}</p>
					{/if}
					<div class="flex gap-6 text-sm">
						<div>
							<span class="text-gray-500">Duration</span>
							<p class="font-medium">{getDuration($currentTrip.startDate, $currentTrip.endDate) || 'Not set'}</p>
						</div>
						<div>
							<span class="text-gray-500">Status</span>
							<p class="font-medium {getStatusColor($currentTrip.status)}">
								{getStatusLabel($currentTrip.status)}
							</p>
						</div>
						<div>
							<span class="text-gray-500">Items</span>
							<p class="font-medium">{$currentTrip.items?.length || 0}</p>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Locations Section -->
		{#if $currentTrip.startDate && $currentTrip.endDate}
			<div class="bg-white rounded-lg shadow-sm border mb-6">
				<button
					type="button"
					class="w-full flex items-center justify-between p-4 text-left hover:bg-gray-50"
					on:click={() => (showLocations = !showLocations)}
				>
					<div class="flex items-center gap-2">
						<svg
							class="w-5 h-5 text-gray-500 transform transition-transform {showLocations ? 'rotate-90' : ''}"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
						</svg>
						<span class="font-medium">Locations</span>
					</div>
					<span class="text-sm text-gray-500">Where will you be each day?</span>
				</button>
				{#if showLocations}
					<div class="p-4 border-t">
						<LocationEditor
							tripId={$currentTrip.id}
							startDate={$currentTrip.startDate}
							endDate={$currentTrip.endDate}
						/>
					</div>
				{/if}
			</div>
		{/if}

		<!-- Add Item Section -->
		<div class="bg-white rounded-lg shadow-sm border mb-6">
			<button
				type="button"
				class="w-full flex items-center justify-between p-4 text-left hover:bg-gray-50"
				on:click={() => (showAddItem = !showAddItem)}
			>
				<div class="flex items-center gap-2">
					<svg
						class="w-5 h-5 text-gray-500 transform transition-transform {showAddItem ? 'rotate-90' : ''}"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
					<span class="font-medium">Add Item</span>
				</div>
				<span class="text-sm text-gray-500">Flights, hotels, events, and more</span>
			</button>
			{#if showAddItem && $currentTrip}
				<div class="p-4 border-t">
					<ItemForm
						tripId={$currentTrip.id}
						onSave={handleItemAdded}
						onCancel={() => (showAddItem = false)}
					/>
				</div>
			{/if}
		</div>

		<!-- Timeline of Items -->
		{#if itemGroups.size === 0}
			<div class="text-center py-12 bg-white rounded-lg shadow-sm border">
				<svg
					class="w-12 h-12 mx-auto mb-3 text-gray-300"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="1.5"
						d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
					/>
				</svg>
				<p class="text-gray-500">No items yet</p>
				<p class="text-xs text-gray-400 mt-1">Add flights, hotels, events, and more</p>
			</div>
		{:else}
			<div class="space-y-4">
				{#each [...itemGroups.entries()] as [date, items]}
					{@const dateInfo = formatDateHeader(date)}
					<!-- Day Header -->
					<div class="flex items-center gap-3 pt-4">
						<div class="text-center">
							<div class="text-2xl font-bold">{dateInfo.day}</div>
							<div class="text-xs text-gray-500 uppercase">{dateInfo.weekday}</div>
						</div>
						<div class="flex-1 border-t border-gray-200"></div>
						<div class="text-sm text-gray-500">{dateInfo.monthYear}</div>
					</div>

					<!-- Items for this day -->
					{#each items as item (item.id)}
						<div class="ml-12">
							<ItemCard {item} onDelete={handleDeleteItem} onMove={handleMoveItem} />
						</div>
					{/each}
				{/each}
			</div>
		{/if}
	</main>
{/if}

<!-- Merge Trip Modal -->
{#if showMergeModal && $currentTrip}
	<TripPickerModal
		excludeTripId={$currentTrip.id}
		title="Merge into..."
		onSelect={handleMerge}
		onCancel={() => (showMergeModal = false)}
	/>
{/if}

<!-- Move Item Modal -->
{#if showMoveModal && itemToMove && $currentTrip}
	<TripPickerModal
		excludeTripId={$currentTrip.id}
		title="Move item to..."
		onSelect={handleMoveToTrip}
		onCancel={() => {
			showMoveModal = false;
			itemToMove = null;
		}}
	/>
{/if}
