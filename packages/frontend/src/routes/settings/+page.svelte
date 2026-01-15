<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { calendarStore, isCalendarConnected } from '$lib/stores/calendar';
	import type { GoogleCalendar } from '@travel-calendar/shared';

	let loading = true;
	let connecting = false;
	let availableCalendars: GoogleCalendar[] = [];
	let selectedIds: Set<string> = new Set();
	let saving = false;

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
