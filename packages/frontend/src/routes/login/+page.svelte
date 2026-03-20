<script lang="ts">
	import { page } from '$app/stores';
	import { authStore } from '$lib/stores/auth';

	let loading = false;
	let error = '';

	// Check for error param from failed login
	$: {
		const err = $page.url.searchParams.get('error');
		if (err) {
			error = err;
		}
	}

	async function handleLogin() {
		loading = true;
		error = '';
		try {
			await authStore.login();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to start login';
			loading = false;
		}
	}
</script>

<div class="min-h-screen bg-gray-50 flex items-center justify-center">
	<div class="max-w-sm w-full mx-4">
		<div class="bg-white rounded-lg shadow p-8 text-center">
			<h1 class="text-2xl font-bold text-gray-900 mb-2">Travel Calendar</h1>
			<p class="text-gray-600 mb-6">Sign in to manage your trips</p>

			{#if error}
				<div class="mb-4 p-3 bg-red-50 text-red-700 rounded-md text-sm">
					{error}
				</div>
			{/if}

			<button
				class="w-full inline-flex items-center justify-center gap-2 px-4 py-2 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 disabled:opacity-50"
				disabled={loading}
				on:click={handleLogin}
			>
				{#if loading}
					<svg class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
				{:else}
					<svg class="w-5 h-5" viewBox="0 0 24 24">
						<path
							fill="#4285F4"
							d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"
						/>
						<path
							fill="#34A853"
							d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
						/>
						<path
							fill="#FBBC05"
							d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
						/>
						<path
							fill="#EA4335"
							d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
						/>
					</svg>
				{/if}
				<span class="text-gray-700 font-medium">Sign in with Google</span>
			</button>

			<p class="mt-4 text-xs text-gray-400">
				Calendar access is requested to detect trips from your events
			</p>
		</div>
	</div>
</div>
