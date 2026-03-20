<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { authStore, isLoggedIn, authChecked } from '$lib/stores/auth';

	// Pages that don't require authentication
	const publicPaths = ['/login', '/oauth/google/callback'];

	function isPublicPage(pathname: string): boolean {
		return publicPaths.some((p) => pathname.startsWith(p));
	}

	onMount(async () => {
		// Don't check auth on public pages — let them handle it
		if (isPublicPage($page.url.pathname)) {
			return;
		}

		const loggedIn = await authStore.check();
		if (!loggedIn) {
			goto('/login');
		}
	});

	// Watch for auth changes after initial check (only on protected pages)
	$: if ($authChecked && !$isLoggedIn && !isPublicPage($page.url.pathname)) {
		goto('/login');
	}

	async function handleLogout() {
		await authStore.logout();
		goto('/login');
	}
</script>

{#if isPublicPage($page.url.pathname)}
	<slot />
{:else if !$authChecked}
	<div class="flex items-center justify-center min-h-screen">
		<p class="text-gray-500">Loading...</p>
	</div>
{:else if $isLoggedIn}
	<header class="bg-white border-b border-gray-200 px-4 py-2 flex items-center justify-between">
		<nav class="flex items-center gap-4">
			<a href="/calendar" class="text-sm font-medium text-gray-700 hover:text-gray-900">Calendar</a>
			<a href="/trips" class="text-sm font-medium text-gray-700 hover:text-gray-900">Trips</a>
			<a href="/settings" class="text-sm font-medium text-gray-700 hover:text-gray-900">Settings</a>
		</nav>
		<div class="flex items-center gap-3">
			<span class="text-sm text-gray-500">{$authStore.email}</span>
			<button
				class="text-sm text-gray-500 hover:text-gray-700"
				on:click={handleLogout}
			>
				Sign out
			</button>
		</div>
	</header>
	<slot />
{/if}
