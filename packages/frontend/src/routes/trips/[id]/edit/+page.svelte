<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import type { UpdateTripRequest } from '@travel-calendar/shared';
	import { trips, currentTrip } from '$lib/stores';
	import TripForm from '$lib/components/trip/TripForm.svelte';

	let tripForm: TripForm;
	let loading = true;
	let saving = false;
	let error: string | null = null;

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

	async function handleSubmit(data: UpdateTripRequest) {
		saving = true;
		error = null;

		try {
			await trips.update(tripId, data);
			await currentTrip.load(tripId);
			goto(`/trips/${tripId}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update trip';
			saving = false;
		}
	}

	function handleCancel() {
		goto(`/trips/${tripId}`);
	}

	function handleSaveClick() {
		tripForm?.submit();
	}
</script>

<svelte:head>
	<title>{$currentTrip?.name ? `Edit ${$currentTrip.name}` : 'Edit Trip'} - Travel Calendar</title>
</svelte:head>

{#if loading}
	<header class="bg-white border-b sticky top-0 z-10">
		<div class="max-w-2xl mx-auto px-4 py-3 flex items-center gap-4">
			<button
				type="button"
				on:click={handleCancel}
				class="text-gray-500 hover:text-gray-700"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M6 18L18 6M6 6l12 12"
					/>
				</svg>
			</button>
			<h1 class="font-semibold text-lg flex-1">Loading...</h1>
		</div>
	</header>
	<main class="max-w-2xl mx-auto px-4 py-6">
		<div class="text-center py-12">
			<p class="text-gray-500">Loading trip...</p>
		</div>
	</main>
{:else if error && !$currentTrip}
	<header class="bg-white border-b sticky top-0 z-10">
		<div class="max-w-2xl mx-auto px-4 py-3 flex items-center gap-4">
			<button
				type="button"
				on:click={handleCancel}
				class="text-gray-500 hover:text-gray-700"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M6 18L18 6M6 6l12 12"
					/>
				</svg>
			</button>
			<h1 class="font-semibold text-lg flex-1">Error</h1>
		</div>
	</header>
	<main class="max-w-2xl mx-auto px-4 py-6">
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
		<div class="max-w-2xl mx-auto px-4 py-3 flex items-center gap-4">
			<button
				type="button"
				on:click={handleCancel}
				class="text-gray-500 hover:text-gray-700"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M6 18L18 6M6 6l12 12"
					/>
				</svg>
			</button>
			<h1 class="font-semibold text-lg flex-1">Edit Trip</h1>
			<button
				type="button"
				on:click={handleSaveClick}
				disabled={saving}
				class="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded disabled:opacity-50"
			>
				{saving ? 'Saving...' : 'Save Changes'}
			</button>
		</div>
	</header>

	<main class="max-w-2xl mx-auto px-4 py-6">
		<TripForm
			bind:this={tripForm}
			trip={$currentTrip}
			mode="edit"
			{saving}
			{error}
			onSubmit={handleSubmit}
			onCancel={handleCancel}
		/>
	</main>
{/if}
