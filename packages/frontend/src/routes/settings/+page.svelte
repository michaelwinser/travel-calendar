<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { calendarStore, isCalendarConnected } from '$lib/stores/calendar';
	import { api } from '$lib/api/client';
	import type { GoogleCalendar, TripSuggestion, Trip, SuggestedItem } from '@travel-calendar/shared';

	let loading = true;
	let connecting = false;
	let availableCalendars: GoogleCalendar[] = [];
	let selectedIds: Set<string> = new Set();
	let saving = false;

	// Trip suggestions state
	let suggestions: TripSuggestion[] = [];
	let loadingSuggestions = false;
	let suggestionsError = '';
	let expandedSuggestions: Set<string> = new Set();
	let importingId: string | null = null;
	let dismissingId: string | null = null;
	let mergingId: string | null = null;
	let importedTrip: Trip | null = null;
	let mergedTrip: Trip | null = null;
	let selectedMergeTarget: Record<string, string> = {};

	// Date range for suggestions (default: next 90 days)
	const today = new Date().toISOString().split('T')[0];
	const ninetyDaysFromNow = new Date(Date.now() + 90 * 24 * 60 * 60 * 1000).toISOString().split('T')[0];
	let suggestionsFrom = today;
	let suggestionsTo = ninetyDaysFromNow;

	// Reset processed events state
	let resettingProcessedEvents = false;
	let resetSuccess = false;

	onMount(async () => {
		await calendarStore.loadAuthStatus();
		loading = false;

		// If connected, load calendars
		if ($calendarStore.authStatus?.connected) {
			await calendarStore.loadCalendars();
			availableCalendars = $calendarStore.availableCalendars;
			selectedIds = new Set($calendarStore.selectedCalendars.map((c) => c.calendarId));
		}
	});

	async function handleConnect() {
		connecting = true;
		await calendarStore.startOAuth();
		// Will redirect to Google
	}

	async function handleDisconnect() {
		if (confirm('Are you sure you want to disconnect Google Calendar?')) {
			await calendarStore.disconnect();
			availableCalendars = [];
			selectedIds = new Set();
			suggestions = [];
		}
	}

	function toggleCalendar(id: string) {
		if (selectedIds.has(id)) {
			selectedIds.delete(id);
		} else {
			selectedIds.add(id);
		}
		selectedIds = selectedIds; // Trigger reactivity
	}

	async function saveCalendarSelection() {
		saving = true;
		await calendarStore.setSelectedCalendars(Array.from(selectedIds));
		saving = false;
	}

	async function findTravelEvents() {
		loadingSuggestions = true;
		suggestionsError = '';
		importedTrip = null;

		try {
			suggestions = await api.calendar.getTripSuggestions(suggestionsFrom, suggestionsTo);
		} catch (err) {
			suggestionsError = err instanceof Error ? err.message : 'Failed to load suggestions';
		} finally {
			loadingSuggestions = false;
		}
	}

	function toggleSuggestionExpanded(id: string) {
		if (expandedSuggestions.has(id)) {
			expandedSuggestions.delete(id);
		} else {
			expandedSuggestions.add(id);
		}
		expandedSuggestions = expandedSuggestions; // Trigger reactivity
	}

	async function importSuggestion(suggestionId: string) {
		importingId = suggestionId;
		importedTrip = null;
		mergedTrip = null;
		suggestionsError = '';

		try {
			importedTrip = await api.calendar.importSuggestion(suggestionId);
			// Remove the imported suggestion from the list
			suggestions = suggestions.filter((s) => s.id !== suggestionId);
		} catch (err) {
			suggestionsError = err instanceof Error ? err.message : 'Failed to import trip';
		} finally {
			importingId = null;
		}
	}

	async function dismissSuggestion(suggestionId: string) {
		dismissingId = suggestionId;
		suggestionsError = '';

		try {
			await api.calendar.dismissSuggestion(suggestionId);
			// Remove the dismissed suggestion from the list
			suggestions = suggestions.filter((s) => s.id !== suggestionId);
		} catch (err) {
			suggestionsError = err instanceof Error ? err.message : 'Failed to dismiss suggestion';
		} finally {
			dismissingId = null;
		}
	}

	async function mergeSuggestion(suggestionId: string) {
		const tripId = selectedMergeTarget[suggestionId];
		if (!tripId) return;

		mergingId = suggestionId;
		importedTrip = null;
		mergedTrip = null;
		suggestionsError = '';

		try {
			mergedTrip = await api.calendar.mergeSuggestion(suggestionId, tripId);
			// Remove the merged suggestion from the list
			suggestions = suggestions.filter((s) => s.id !== suggestionId);
			// Clear the selection
			delete selectedMergeTarget[suggestionId];
			selectedMergeTarget = selectedMergeTarget;
		} catch (err) {
			suggestionsError = err instanceof Error ? err.message : 'Failed to merge suggestion';
		} finally {
			mergingId = null;
		}
	}

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function formatEventTime(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric'
		}) + ' at ' + date.toLocaleTimeString('en-US', {
			hour: 'numeric',
			minute: '2-digit'
		});
	}

	function getItemIcon(type: SuggestedItem['type']): string {
		switch (type) {
			case 'flight': return '✈️';
			case 'hotel': return '🏨';
			case 'train': return '🚆';
			case 'drive': return '🚗';
			case 'event': return '📍';
			default: return '📌';
		}
	}

	function formatItemSummary(item: SuggestedItem): string {
		switch (item.type) {
			case 'flight':
				if (item.carrier && item.flightNumber) {
					return `${item.carrier}${item.flightNumber}${item.from && item.to ? `: ${item.from} → ${item.to}` : ''}`;
				}
				return item.from && item.to ? `${item.from} → ${item.to}` : item.name || 'Flight';
			case 'hotel':
				return item.name || item.location || 'Hotel';
			case 'train':
				return item.from && item.to ? `${item.from} → ${item.to}` : item.name || 'Train';
			case 'drive':
				return item.from && item.to ? `${item.from} → ${item.to}` : item.name || 'Drive';
			case 'event':
				return item.name || item.location || 'Event';
			default:
				return item.name || 'Item';
		}
	}

	async function resetProcessedEvents() {
		if (!confirm('Are you sure you want to reset processed events? This will allow previously dismissed or imported suggestions to reappear.')) {
			return;
		}

		resettingProcessedEvents = true;
		resetSuccess = false;
		suggestionsError = '';

		try {
			await api.calendar.resetProcessedEvents();
			resetSuccess = true;
			// Clear current suggestions so user can re-fetch
			suggestions = [];
		} catch (err) {
			suggestionsError = err instanceof Error ? err.message : 'Failed to reset processed events';
		} finally {
			resettingProcessedEvents = false;
		}
	}
</script>

<div class="min-h-screen bg-gray-50 py-8">
	<div class="max-w-2xl mx-auto px-4">
		<header class="mb-8">
			<button
				class="text-blue-600 hover:text-blue-800 mb-4 flex items-center gap-1"
				on:click={() => goto('/calendar')}
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M15 19l-7-7 7-7"
					/>
				</svg>
				Back to Calendar
			</button>
			<h1 class="text-2xl font-bold text-gray-900">Settings</h1>
		</header>

		{#if loading}
			<div class="bg-white rounded-lg shadow p-6">
				<p class="text-gray-500">Loading...</p>
			</div>
		{:else}
			<!-- Google Calendar Section -->
			<section class="bg-white rounded-lg shadow mb-6">
				<div class="p-6 border-b">
					<h2 class="text-lg font-semibold text-gray-900 flex items-center gap-2">
						<svg class="w-6 h-6 text-blue-500" fill="currentColor" viewBox="0 0 24 24">
							<path
								d="M19 4h-1V2h-2v2H8V2H6v2H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 16H5V10h14v10zm0-12H5V6h14v2z"
							/>
						</svg>
						Google Calendar
					</h2>
				</div>

				<div class="p-6">
					{#if $calendarStore.error}
						<div class="mb-4 p-3 bg-red-50 text-red-700 rounded-md">
							{$calendarStore.error}
							<button
								class="ml-2 text-red-500 hover:text-red-700"
								on:click={() => calendarStore.clearError()}
							>
								Dismiss
							</button>
						</div>
					{/if}

					{#if !$isCalendarConnected}
						<div class="text-center py-4">
							<p class="text-gray-600 mb-4">
								Connect your Google Calendar to detect conflicts and sync your trips.
							</p>
							<button
								class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
								disabled={connecting}
								on:click={handleConnect}
							>
								{#if connecting}
									<svg class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
										<circle
											class="opacity-25"
											cx="12"
											cy="12"
											r="10"
											stroke="currentColor"
											stroke-width="4"
										></circle>
										<path
											class="opacity-75"
											fill="currentColor"
											d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
										></path>
									</svg>
									Connecting...
								{:else}
									<svg class="w-5 h-5" viewBox="0 0 24 24">
										<path
											fill="currentColor"
											d="M12.545,10.239v3.821h5.445c-0.712,2.315-2.647,3.972-5.445,3.972c-3.332,0-6.033-2.701-6.033-6.032s2.701-6.032,6.033-6.032c1.498,0,2.866,0.549,3.921,1.453l2.814-2.814C17.503,2.988,15.139,2,12.545,2C7.021,2,2.543,6.477,2.543,12s4.478,10,10.002,10c8.396,0,10.249-7.85,9.426-11.748L12.545,10.239z"
										/>
									</svg>
									Connect Google Calendar
								{/if}
							</button>
						</div>
					{:else}
						<div class="space-y-4">
							<!-- Connected status -->
							<div class="flex items-center justify-between p-3 bg-green-50 rounded-md">
								<div class="flex items-center gap-2">
									<svg
										class="w-5 h-5 text-green-600"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M5 13l4 4L19 7"
										/>
									</svg>
									<span class="text-green-800">
										Connected as {$calendarStore.authStatus?.email || 'Unknown'}
									</span>
								</div>
								<button
									class="text-sm text-red-600 hover:text-red-800"
									on:click={handleDisconnect}
								>
									Disconnect
								</button>
							</div>

							<!-- Calendar selection -->
							{#if availableCalendars.length > 0}
								<div>
									<h3 class="font-medium text-gray-900 mb-2">Select calendars to monitor</h3>
									<p class="text-sm text-gray-500 mb-3">
										Choose which calendars to check for conflicts with your trips.
									</p>
									<div class="space-y-2 max-h-64 overflow-y-auto border rounded-md p-2">
										{#each availableCalendars as calendar (calendar.id)}
											<label
												class="flex items-center gap-3 p-2 hover:bg-gray-50 rounded cursor-pointer"
											>
												<input
													type="checkbox"
													checked={selectedIds.has(calendar.id)}
													on:change={() => toggleCalendar(calendar.id)}
													class="w-4 h-4 text-blue-600 rounded"
												/>
												<span
													class="w-3 h-3 rounded-full flex-shrink-0"
													style="background-color: {calendar.backgroundColor || '#4285f4'}"
												></span>
												<span class="text-gray-900">
													{calendar.name}
													{#if calendar.primary}
														<span class="text-xs text-gray-500">(Primary)</span>
													{/if}
												</span>
											</label>
										{/each}
									</div>
									<button
										class="mt-3 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
										disabled={saving}
										on:click={saveCalendarSelection}
									>
										{saving ? 'Saving...' : 'Save Selection'}
									</button>
								</div>
							{:else if $calendarStore.loading}
								<p class="text-gray-500">Loading calendars...</p>
							{:else}
								<p class="text-gray-500">No calendars found. Please check your Google Calendar settings.</p>
							{/if}
						</div>
					{/if}
				</div>
			</section>

			<!-- Trip Suggestions Section -->
			{#if $isCalendarConnected && $calendarStore.selectedCalendars.length > 0}
				<section class="bg-white rounded-lg shadow mb-6">
					<div class="p-6 border-b">
						<h2 class="text-lg font-semibold text-gray-900 flex items-center gap-2">
							<svg class="w-6 h-6 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
							</svg>
							Trip Suggestions
						</h2>
					</div>

					<div class="p-6">
						<p class="text-sm text-gray-600 mb-4">
							Find travel-related events in your calendar (events with locations or travel keywords like
							Flight, Hotel, Train, etc.) and import them as trips.
						</p>

						<!-- Date range selector -->
						<div class="flex flex-wrap gap-4 mb-4">
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-1">From</label>
								<input
									type="date"
									bind:value={suggestionsFrom}
									class="px-3 py-2 border rounded-md text-sm"
								/>
							</div>
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-1">To</label>
								<input
									type="date"
									bind:value={suggestionsTo}
									class="px-3 py-2 border rounded-md text-sm"
								/>
							</div>
							<div class="flex items-end">
								<button
									class="px-4 py-2 bg-purple-600 text-white rounded-md hover:bg-purple-700 disabled:opacity-50 flex items-center gap-2"
									disabled={loadingSuggestions}
									on:click={findTravelEvents}
								>
									{#if loadingSuggestions}
										<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
											<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
											<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
										</svg>
										Searching...
									{:else}
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
										</svg>
										Find Travel Events
									{/if}
								</button>
							</div>
						</div>

						<!-- Reset processed events -->
						<div class="flex items-center justify-between p-3 bg-gray-50 rounded-md mb-4">
							<div>
								<p class="text-sm font-medium text-gray-700">Reset processed events</p>
								<p class="text-xs text-gray-500">Allow previously dismissed or imported suggestions to reappear</p>
							</div>
							<button
								class="px-3 py-1.5 text-sm text-red-600 border border-red-200 rounded-md hover:bg-red-50 disabled:opacity-50"
								disabled={resettingProcessedEvents}
								on:click={resetProcessedEvents}
							>
								{resettingProcessedEvents ? 'Resetting...' : 'Reset'}
							</button>
						</div>

						{#if resetSuccess}
							<div class="mb-4 p-3 bg-green-50 text-green-700 rounded-md">
								Processed events reset. Click "Find Travel Events" to see all suggestions again.
							</div>
						{/if}

						{#if suggestionsError}
							<div class="mb-4 p-3 bg-red-50 text-red-700 rounded-md">
								{suggestionsError}
							</div>
						{/if}

						{#if importedTrip}
							<div class="mb-4 p-3 bg-green-50 text-green-700 rounded-md flex items-center justify-between">
								<span>
									Trip "{importedTrip.name}" created{importedTrip.items && importedTrip.items.length > 0 ? ` with ${importedTrip.items.length} item${importedTrip.items.length !== 1 ? 's' : ''}` : ''}!
								</span>
								<a
									href="/trips/{importedTrip.id}"
									class="text-green-800 underline hover:no-underline"
								>
									View Trip
								</a>
							</div>
						{/if}

						{#if mergedTrip}
							<div class="mb-4 p-3 bg-blue-50 text-blue-700 rounded-md flex items-center justify-between">
								<span>
									Items merged into "{mergedTrip.name}"{mergedTrip.items && mergedTrip.items.length > 0 ? ` (now has ${mergedTrip.items.length} item${mergedTrip.items.length !== 1 ? 's' : ''})` : ''}!
								</span>
								<a
									href="/trips/{mergedTrip.id}"
									class="text-blue-800 underline hover:no-underline"
								>
									View Trip
								</a>
							</div>
						{/if}

						<!-- Suggestions list -->
						{#if suggestions.length > 0}
							<div class="space-y-3">
								{#each suggestions as suggestion (suggestion.id)}
									<div class="border rounded-lg overflow-hidden">
										<div class="p-4 bg-gray-50">
											<div class="flex items-start justify-between">
												<div class="flex-1">
													<div class="flex items-center gap-2 flex-wrap">
														<h3 class="font-medium text-gray-900">{suggestion.name}</h3>
														{#if suggestion.source === 'tripit'}
															<span class="px-2 py-0.5 text-xs font-medium bg-indigo-100 text-indigo-700 rounded-full">
																TripIt
															</span>
														{/if}
													</div>
													<p class="text-sm text-gray-600">{suggestion.location}</p>
													<p class="text-sm text-gray-500 mt-1">
														{formatDate(suggestion.startDate)} - {formatDate(suggestion.endDate)}
													</p>

													{#if suggestion.mergeCandidates && suggestion.mergeCandidates.length > 0}
														<div class="mt-2">
															{#each suggestion.mergeCandidates as candidate}
																<div class="flex items-center gap-1.5 text-sm text-amber-700 bg-amber-50 px-2 py-1 rounded-md inline-flex">
																	<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
																		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
																	</svg>
																	<span>Similar: <a href="/trips/{candidate.tripId}" class="underline hover:no-underline">{candidate.tripName}</a></span>
																	<span class="text-amber-600">({candidate.matchReason})</span>
																</div>
															{/each}
														</div>
													{/if}
												</div>
												<!-- Action buttons -->
												<div class="ml-4 flex items-center gap-2">
													<!-- Dismiss button -->
													<button
														class="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-md disabled:opacity-50"
														title="Dismiss suggestion"
														disabled={dismissingId === suggestion.id}
														on:click={() => dismissSuggestion(suggestion.id)}
													>
														{#if dismissingId === suggestion.id}
															<svg class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
																<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
																<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
															</svg>
														{:else}
															<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
															</svg>
														{/if}
													</button>

													{#if suggestion.mergeCandidates && suggestion.mergeCandidates.length > 0}
														<!-- Merge dropdown -->
														<div class="flex items-center gap-1">
															<select
																class="px-2 py-1.5 text-sm border rounded-md bg-white"
																bind:value={selectedMergeTarget[suggestion.id]}
															>
																<option value="">Merge into...</option>
																{#each suggestion.mergeCandidates as candidate}
																	<option value={candidate.tripId}>{candidate.tripName}</option>
																{/each}
															</select>
															{#if selectedMergeTarget[suggestion.id]}
																<button
																	class="px-3 py-1.5 bg-amber-600 text-white text-sm rounded-md hover:bg-amber-700 disabled:opacity-50"
																	disabled={mergingId === suggestion.id}
																	on:click={() => mergeSuggestion(suggestion.id)}
																>
																	{mergingId === suggestion.id ? 'Merging...' : 'Merge'}
																</button>
															{/if}
														</div>
														<span class="text-gray-300">|</span>
													{/if}

													<!-- Import button -->
													<button
														class="px-3 py-1.5 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 disabled:opacity-50"
														disabled={importingId === suggestion.id}
														on:click={() => importSuggestion(suggestion.id)}
													>
														{importingId === suggestion.id ? 'Importing...' : 'Import as New'}
													</button>
												</div>
											</div>

											{#if suggestion.suggestedItems && suggestion.suggestedItems.length > 0}
												<div class="mt-3 pt-3 border-t border-gray-200">
													<p class="text-xs font-medium text-gray-500 uppercase mb-2">Items to create</p>
													<div class="flex flex-wrap gap-2">
														{#each suggestion.suggestedItems as item}
															<span class="inline-flex items-center gap-1 px-2 py-1 bg-white border border-gray-200 rounded text-sm text-gray-700">
																<span>{getItemIcon(item.type)}</span>
																<span>{formatItemSummary(item)}</span>
															</span>
														{/each}
													</div>
												</div>
											{/if}

											<!-- Toggle source events -->
											<button
												class="mt-3 text-sm text-gray-500 hover:text-gray-700 flex items-center gap-1"
												on:click={() => toggleSuggestionExpanded(suggestion.id)}
											>
												<svg
													class="w-4 h-4 transition-transform {expandedSuggestions.has(suggestion.id) ? 'rotate-90' : ''}"
													fill="none"
													stroke="currentColor"
													viewBox="0 0 24 24"
												>
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
												</svg>
												{suggestion.sourceEvents.length} source event{suggestion.sourceEvents.length !== 1 ? 's' : ''}
											</button>
										</div>

										<!-- Source events (collapsible) -->
										{#if expandedSuggestions.has(suggestion.id)}
											<div class="border-t bg-white divide-y">
												{#each suggestion.sourceEvents as event (event.id)}
													<div class="p-3 text-sm">
														<div class="font-medium text-gray-900">{event.summary}</div>
														{#if event.location}
															<div class="text-gray-600">{event.location}</div>
														{/if}
														<div class="text-gray-500">{formatEventTime(event.start)}</div>
													</div>
												{/each}
											</div>
										{/if}
									</div>
								{/each}
							</div>
						{:else if !loadingSuggestions && suggestions.length === 0}
							<p class="text-gray-500 text-sm">
								Click "Find Travel Events" to search for travel-related events in your selected calendars.
							</p>
						{/if}
					</div>
				</section>
			{/if}

			<!-- Info section -->
			<section class="bg-white rounded-lg shadow">
				<div class="p-6 border-b">
					<h2 class="text-lg font-semibold text-gray-900">About Calendar Integration</h2>
				</div>
				<div class="p-6 text-sm text-gray-600 space-y-2">
					<p>When connected, Travel Calendar can:</p>
					<ul class="list-disc list-inside space-y-1 ml-2">
						<li>Detect conflicts between your calendar events and trips</li>
						<li>Suggest new trips based on travel-related calendar events</li>
						<li>Sync your trips to Google Calendar</li>
					</ul>
					<p class="mt-4 text-gray-500">
						Your calendar data is only used to provide these features and is never shared with third
						parties.
					</p>
				</div>
			</section>
		{/if}
	</div>
</div>
