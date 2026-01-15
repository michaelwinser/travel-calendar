<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';

	let status: 'loading' | 'success' | 'error' = 'loading';
	let errorMessage = '';

	onMount(async () => {
		const code = $page.url.searchParams.get('code');
		const error = $page.url.searchParams.get('error');

		if (error) {
			status = 'error';
			errorMessage = error === 'access_denied'
				? 'You denied access to your Google Calendar.'
				: `Google returned an error: ${error}`;
			return;
		}

		if (!code) {
			status = 'error';
			errorMessage = 'No authorization code received from Google.';
			return;
		}

		try {
			// Exchange the code for tokens via our backend
			const response = await fetch(`/api/auth/google/callback?code=${encodeURIComponent(code)}`);

			if (!response.ok) {
				const data = await response.json().catch(() => ({ error: 'Unknown error' }));
				throw new Error(data.error || `HTTP ${response.status}`);
			}

			status = 'success';
			// Redirect to settings after a brief delay
			setTimeout(() => goto('/settings'), 1500);
		} catch (err) {
			status = 'error';
			errorMessage = err instanceof Error ? err.message : 'Failed to complete authentication';
		}
	});
</script>

<div class="min-h-screen bg-gray-50 flex items-center justify-center">
	<div class="bg-white rounded-lg shadow-lg p-8 max-w-md w-full mx-4">
		{#if status === 'loading'}
			<div class="text-center">
				<svg class="w-12 h-12 animate-spin text-blue-600 mx-auto mb-4" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
				<h2 class="text-xl font-semibold text-gray-900">Connecting to Google Calendar</h2>
				<p class="text-gray-500 mt-2">Please wait while we complete the authentication...</p>
			</div>
		{:else if status === 'success'}
			<div class="text-center">
				<svg class="w-12 h-12 text-green-500 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
				</svg>
				<h2 class="text-xl font-semibold text-gray-900">Connected!</h2>
				<p class="text-gray-500 mt-2">Your Google Calendar is now connected. Redirecting to settings...</p>
			</div>
		{:else}
			<div class="text-center">
				<svg class="w-12 h-12 text-red-500 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
				<h2 class="text-xl font-semibold text-gray-900">Connection Failed</h2>
				<p class="text-red-600 mt-2">{errorMessage}</p>
				<div class="mt-6 space-x-4">
					<button
						class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
						on:click={() => goto('/settings')}
					>
						Back to Settings
					</button>
				</div>
			</div>
		{/if}
	</div>
</div>
