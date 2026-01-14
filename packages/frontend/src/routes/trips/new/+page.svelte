<script lang="ts">
	import { goto } from '$app/navigation';
	import type { TripPurpose, TripStatus } from '@travel-calendar/shared';
	import { trips } from '$lib/stores';

	let name = '';
	let purpose: TripPurpose = 'conference';
	let startDate = '';
	let endDate = '';
	let status: TripStatus = 'planning';
	let notes = '';

	let saving = false;
	let error: string | null = null;

	const purposes: { value: TripPurpose; label: string; color: string }[] = [
		{ value: 'conference', label: 'Conference', color: 'orange' },
		{ value: 'business', label: 'Business', color: 'yellow' },
		{ value: 'vacation', label: 'Vacation', color: 'blue' },
		{ value: 'family', label: 'Family', color: 'green' },
		{ value: 'other', label: 'Other', color: 'purple' }
	];

	async function handleSubmit() {
		if (!name.trim()) {
			error = 'Trip name is required';
			return;
		}

		saving = true;
		error = null;

		try {
			const trip = await trips.create({
				name: name.trim(),
				purpose,
				startDate: startDate || undefined,
				endDate: endDate || undefined,
				status,
				notes: notes.trim() || undefined
			});
			goto(`/trips/${trip.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create trip';
			saving = false;
		}
	}

	function handleCancel() {
		goto('/trips');
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
			on:click={handleSubmit}
			disabled={saving}
			class="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded disabled:opacity-50"
		>
			{saving ? 'Saving...' : 'Save Trip'}
		</button>
	</div>
</header>

<main class="max-w-2xl mx-auto px-4 py-6">
	{#if error}
		<div class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
			{error}
		</div>
	{/if}

	<form on:submit|preventDefault={handleSubmit} class="space-y-6">
		<!-- Basic Info -->
		<div class="bg-white rounded-lg shadow-sm border p-4">
			<h2 class="font-medium text-gray-900 mb-4">Trip Details</h2>

			<div class="space-y-4">
				<div>
					<label for="name" class="block text-sm font-medium text-gray-700 mb-1">
						Trip Name
					</label>
					<input
						type="text"
						id="name"
						bind:value={name}
						placeholder="e.g., FOSDEM 2025, Summer Vacation"
						class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
				</div>

				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="startDate" class="block text-sm font-medium text-gray-700 mb-1">
							Start Date
						</label>
						<input
							type="date"
							id="startDate"
							bind:value={startDate}
							class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
						/>
					</div>
					<div>
						<label for="endDate" class="block text-sm font-medium text-gray-700 mb-1">
							End Date
						</label>
						<input
							type="date"
							id="endDate"
							bind:value={endDate}
							class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
						/>
					</div>
				</div>

				<div>
					<label class="block text-sm font-medium text-gray-700 mb-1">Type</label>
					<div class="flex flex-wrap gap-2">
						{#each purposes as { value, label, color }}
							<label
								class="flex items-center gap-2 px-3 py-2 border rounded-lg cursor-pointer hover:bg-gray-50 transition-colors
									{purpose === value ? `border-${color}-500 bg-${color}-50` : ''}"
							>
								<input
									type="radio"
									name="purpose"
									{value}
									bind:group={purpose}
									class="sr-only"
								/>
								<span class="w-3 h-3 rounded-full bg-{color}-300"></span>
								<span class="text-sm">{label}</span>
							</label>
						{/each}
					</div>
				</div>

				<div>
					<label for="status" class="block text-sm font-medium text-gray-700 mb-1">
						Status
					</label>
					<select
						id="status"
						bind:value={status}
						class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
					>
						<option value="planning">Planning</option>
						<option value="confirmed">Confirmed</option>
						<option value="in_progress">In Progress</option>
						<option value="completed">Completed</option>
						<option value="cancelled">Cancelled</option>
					</select>
				</div>

				<div>
					<label for="notes" class="block text-sm font-medium text-gray-700 mb-1">
						Description
					</label>
					<textarea
						id="notes"
						bind:value={notes}
						rows="3"
						placeholder="What's this trip about?"
						class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
					></textarea>
				</div>
			</div>
		</div>

		<!-- Info about adding items -->
		<div class="bg-gray-50 rounded-lg border p-4 text-sm text-gray-600">
			<p>
				After creating the trip, you can add flights, hotels, events, and other items from the trip detail page.
			</p>
		</div>
	</form>
</main>
