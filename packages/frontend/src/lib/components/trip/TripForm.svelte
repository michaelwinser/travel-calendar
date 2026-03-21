<script lang="ts">
	import type { Trip, TripPurpose, TripStatus, CreateTripRequest, UpdateTripRequest } from '@travel-calendar/shared';

	export let trip: Partial<Trip> = {};
	export let mode: 'create' | 'edit' = 'create';
	export let saving: boolean = false;
	export let error: string | null = null;
	export let onSubmit: (data: CreateTripRequest | UpdateTripRequest) => void;
	export let onCancel: () => void;

	let name = trip.name || '';
	let location = trip.location || '';
	let purpose: TripPurpose = trip.purpose || 'conference';
	let startDate = trip.startDate || '';
	let endDate = trip.endDate || '';
	let status: TripStatus = trip.status || 'planning';
	let notes = trip.notes || '';

	const purposeSelectedClasses: Record<TripPurpose, string> = {
		conference: 'border-orange-500 bg-orange-50',
		business: 'border-yellow-500 bg-yellow-50',
		vacation: 'border-blue-500 bg-blue-50',
		family: 'border-green-500 bg-green-50',
		other: 'border-purple-500 bg-purple-50'
	};

	const purposeDotClasses: Record<TripPurpose, string> = {
		conference: 'bg-orange-300',
		business: 'bg-yellow-300',
		vacation: 'bg-blue-300',
		family: 'bg-green-300',
		other: 'bg-purple-300'
	};

	const purposes: { value: TripPurpose; label: string }[] = [
		{ value: 'conference', label: 'Conference' },
		{ value: 'business', label: 'Business' },
		{ value: 'vacation', label: 'Vacation' },
		{ value: 'family', label: 'Family' },
		{ value: 'other', label: 'Other' }
	];

	$: isValid = name.trim().length > 0;

	export function getIsValid(): boolean {
		return isValid;
	}

	export function submit(): boolean {
		if (!name.trim()) {
			return false;
		}

		const data: CreateTripRequest | UpdateTripRequest = {
			name: name.trim(),
			location: location.trim() || undefined,
			purpose,
			startDate: startDate || undefined,
			endDate: endDate || undefined,
			status,
			notes: notes.trim() || undefined
		};

		onSubmit(data);
		return true;
	}

	function handleSubmit() {
		submit();
	}
</script>

{#if error}
	<div class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
		{error}
	</div>
{/if}

<form on:submit|preventDefault={handleSubmit} class="space-y-6">
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
					required
				/>
			</div>

			<div>
				<label for="location" class="block text-sm font-medium text-gray-700 mb-1">
					Location
				</label>
				<input
					type="text"
					id="location"
					bind:value={location}
					placeholder="e.g., Milan, London, Tokyo"
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
					{#each purposes as { value, label }}
						<label
							class="flex items-center gap-2 px-3 py-2 border rounded-lg cursor-pointer hover:bg-gray-50 transition-colors
								{purpose === value ? purposeSelectedClasses[value] : ''}"
						>
							<input
								type="radio"
								name="purpose"
								{value}
								bind:group={purpose}
								class="sr-only"
							/>
							<span class="w-3 h-3 rounded-full {purposeDotClasses[value]}"></span>
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

	{#if mode === 'create'}
		<div class="bg-gray-50 rounded-lg border p-4 text-sm text-gray-600">
			<p>
				After creating the trip, you can add flights, hotels, events, and other items from the trip detail page.
			</p>
		</div>
	{/if}

</form>
