<script lang="ts">
	import { goto } from '$app/navigation';
	import type { CreateTripRequest } from '@travel-calendar/shared';
	import { trips } from '$lib/stores';
	import TripForm from '$lib/components/trip/TripForm.svelte';

	let tripForm: TripForm;
	let saving = false;
	let error: string | null = null;

	async function handleSubmit(data: CreateTripRequest) {
		saving = true;
		error = null;

		try {
			const trip = await trips.create(data);
			goto(`/trips/${trip.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create trip';
			saving = false;
		}
	}

	function handleCancel() {
		goto('/trips');
	}

	function handleSaveClick() {
		tripForm?.submit();
	}
</script>

<svelte:head>
	<title>New Trip - Travel Calendar</title>
</svelte:head>

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
		<h1 class="font-semibold text-lg flex-1">New Trip</h1>
		<button
			type="button"
			on:click={handleSaveClick}
			disabled={saving}
			class="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded disabled:opacity-50"
		>
			{saving ? 'Saving...' : 'Save Trip'}
		</button>
	</div>
</header>

<main class="max-w-2xl mx-auto px-4 py-6">
	<TripForm
		bind:this={tripForm}
		mode="create"
		{saving}
		{error}
		onSubmit={handleSubmit}
		onCancel={handleCancel}
	/>
</main>
