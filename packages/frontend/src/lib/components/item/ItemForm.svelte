<script lang="ts">
	import type { ItemType, CreateItemRequest } from '@travel-calendar/shared';
	import { api } from '$lib/api';
	import ItemTypeSelector from './ItemTypeSelector.svelte';

	export let tripId: string;
	export let onSave: () => void;
	export let onCancel: () => void;

	let selectedType: ItemType | null = null;
	let saving = false;
	let error: string | null = null;

	// Common fields
	let date = '';
	let time = '';
	let confirmation = '';
	let notes = '';

	// Transport fields (flight, train, drive)
	let from = '';
	let to = '';
	let carrier = '';
	let flightNumber = '';

	// Hotel/Event fields
	let name = '';
	let location = '';
	let checkIn = '';
	let checkOut = '';

	function handleTypeSelect(type: ItemType) {
		selectedType = type;
		resetForm();
	}

	function resetForm() {
		date = '';
		time = '';
		confirmation = '';
		notes = '';
		from = '';
		to = '';
		carrier = '';
		flightNumber = '';
		name = '';
		location = '';
		checkIn = '';
		checkOut = '';
		error = null;
	}

	function handleCancel() {
		selectedType = null;
		resetForm();
		onCancel();
	}

	async function handleSubmit() {
		if (!selectedType) return;

		saving = true;
		error = null;

		try {
			const request: CreateItemRequest = {
				type: selectedType,
				date: date || undefined,
				time: time || undefined,
				confirmation: confirmation.trim() || undefined,
				notes: notes.trim() || undefined,
				from: from.trim() || undefined,
				to: to.trim() || undefined,
				carrier: carrier.trim() || undefined,
				flightNumber: flightNumber.trim() || undefined,
				name: name.trim() || undefined,
				location: location.trim() || undefined,
				checkIn: checkIn || undefined,
				checkOut: checkOut || undefined
			};

			await api.items.create(tripId, request);
			selectedType = null;
			resetForm();
			onSave();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to add item';
		} finally {
			saving = false;
		}
	}

	// Field labels by type
	const typeLabels: Record<ItemType, string> = {
		flight: 'Flight',
		hotel: 'Hotel',
		train: 'Train',
		drive: 'Drive',
		event: 'Event'
	};

	$: showTransportFields = selectedType === 'flight' || selectedType === 'train' || selectedType === 'drive';
	$: showHotelFields = selectedType === 'hotel';
	$: showEventFields = selectedType === 'event';
	$: showDateTimeFields = selectedType !== 'hotel';
	$: showConfirmation = selectedType !== 'event';
</script>

{#if !selectedType}
	<div>
		<p class="text-sm text-gray-600 mb-3">Select the type of item to add:</p>
		<ItemTypeSelector onSelect={handleTypeSelect} />
	</div>
{:else}
	{#if error}
		<div class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
			{error}
		</div>
	{/if}

	<form on:submit|preventDefault={handleSubmit} class="space-y-4">
		<div class="flex items-center gap-2 mb-4">
			<button
				type="button"
				class="text-gray-400 hover:text-gray-600"
				on:click={() => { selectedType = null; resetForm(); }}
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
				</svg>
			</button>
			<h3 class="font-medium">Add {typeLabels[selectedType]}</h3>
		</div>

		<!-- Transport fields (flight, train, drive) -->
		{#if showTransportFields}
			<div class="grid grid-cols-2 gap-4">
				<div>
					<label for="from" class="block text-sm font-medium text-gray-700 mb-1">
						{selectedType === 'flight' ? 'From (Airport)' : 'From'}
					</label>
					<input
						type="text"
						id="from"
						bind:value={from}
						placeholder={selectedType === 'flight' ? 'e.g., JFK' : 'Origin'}
						class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
				</div>
				<div>
					<label for="to" class="block text-sm font-medium text-gray-700 mb-1">
						{selectedType === 'flight' ? 'To (Airport)' : 'To'}
					</label>
					<input
						type="text"
						id="to"
						bind:value={to}
						placeholder={selectedType === 'flight' ? 'e.g., LAX' : 'Destination'}
						class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
				</div>
			</div>

			{#if selectedType === 'flight' || selectedType === 'train'}
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="carrier" class="block text-sm font-medium text-gray-700 mb-1">
							{selectedType === 'flight' ? 'Airline' : 'Operator'}
						</label>
						<input
							type="text"
							id="carrier"
							bind:value={carrier}
							placeholder={selectedType === 'flight' ? 'e.g., United' : 'e.g., Amtrak'}
							class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
						/>
					</div>
					<div>
						<label for="flightNumber" class="block text-sm font-medium text-gray-700 mb-1">
							{selectedType === 'flight' ? 'Flight Number' : 'Train Number'}
						</label>
						<input
							type="text"
							id="flightNumber"
							bind:value={flightNumber}
							placeholder={selectedType === 'flight' ? 'e.g., UA123' : 'e.g., 2151'}
							class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
						/>
					</div>
				</div>
			{/if}

			{#if selectedType === 'drive'}
				<div>
					<label for="carrier" class="block text-sm font-medium text-gray-700 mb-1">
						Vehicle / Rental Company
					</label>
					<input
						type="text"
						id="carrier"
						bind:value={carrier}
						placeholder="e.g., Hertz, Personal car"
						class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
				</div>
			{/if}
		{/if}

		<!-- Hotel fields -->
		{#if showHotelFields}
			<div>
				<label for="name" class="block text-sm font-medium text-gray-700 mb-1">
					Hotel Name
				</label>
				<input
					type="text"
					id="name"
					bind:value={name}
					placeholder="e.g., Hilton Garden Inn"
					class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
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
					placeholder="e.g., Downtown NYC"
					class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
				/>
			</div>
			<div class="grid grid-cols-2 gap-4">
				<div>
					<label for="checkIn" class="block text-sm font-medium text-gray-700 mb-1">
						Check-in Date
					</label>
					<input
						type="date"
						id="checkIn"
						bind:value={checkIn}
						class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
				</div>
				<div>
					<label for="checkOut" class="block text-sm font-medium text-gray-700 mb-1">
						Check-out Date
					</label>
					<input
						type="date"
						id="checkOut"
						bind:value={checkOut}
						class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
				</div>
			</div>
		{/if}

		<!-- Event fields -->
		{#if showEventFields}
			<div>
				<label for="name" class="block text-sm font-medium text-gray-700 mb-1">
					Event Name
				</label>
				<input
					type="text"
					id="name"
					bind:value={name}
					placeholder="e.g., Conference Keynote"
					class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
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
					placeholder="e.g., Convention Center, Room 101"
					class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
				/>
			</div>
		{/if}

		<!-- Date/Time fields (not for hotel) -->
		{#if showDateTimeFields}
			<div class="grid grid-cols-2 gap-4">
				<div>
					<label for="date" class="block text-sm font-medium text-gray-700 mb-1">
						Date
					</label>
					<input
						type="date"
						id="date"
						bind:value={date}
						class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
				</div>
				<div>
					<label for="time" class="block text-sm font-medium text-gray-700 mb-1">
						Time
					</label>
					<input
						type="time"
						id="time"
						bind:value={time}
						class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
				</div>
			</div>
		{/if}

		<!-- Confirmation (not for event) -->
		{#if showConfirmation}
			<div>
				<label for="confirmation" class="block text-sm font-medium text-gray-700 mb-1">
					Confirmation Number
				</label>
				<input
					type="text"
					id="confirmation"
					bind:value={confirmation}
					placeholder="e.g., ABC123"
					class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
				/>
			</div>
		{/if}

		<!-- Notes (all types) -->
		<div>
			<label for="notes" class="block text-sm font-medium text-gray-700 mb-1">
				Notes
			</label>
			<textarea
				id="notes"
				bind:value={notes}
				rows="2"
				placeholder="Additional notes..."
				class="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
			></textarea>
		</div>

		<!-- Actions -->
		<div class="flex justify-end gap-3 pt-2">
			<button
				type="button"
				class="px-4 py-2 text-sm text-gray-600 hover:text-gray-800"
				on:click={handleCancel}
				disabled={saving}
			>
				Cancel
			</button>
			<button
				type="submit"
				class="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
				disabled={saving}
			>
				{saving ? 'Adding...' : 'Add Item'}
			</button>
		</div>
	</form>
{/if}
