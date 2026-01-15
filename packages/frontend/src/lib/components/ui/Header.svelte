<script lang="ts">
	import { page } from '$app/stores';

	export let title: string = 'Travel Calendar';
	export let showBackButton: boolean = false;
	export let backHref: string = '/';
	export let onNewTrip: (() => void) | undefined = undefined;
</script>

<header class="bg-white border-b sticky top-0 z-10">
	<div class="max-w-4xl mx-auto px-4 py-3 flex items-center gap-4">
		{#if showBackButton}
			<a href={backHref} class="text-gray-500 hover:text-gray-700">
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
				</svg>
			</a>
		{/if}

		<h1 class="text-xl font-semibold flex-1">{title}</h1>

		<nav class="flex items-center gap-2">
			<a
				href="/calendar"
				class="px-3 py-1.5 text-sm rounded {$page.url.pathname === '/calendar'
					? 'bg-gray-100'
					: 'hover:bg-gray-100'}"
			>
				Calendar
			</a>
			<a
				href="/trips"
				class="px-3 py-1.5 text-sm rounded {$page.url.pathname.startsWith('/trips')
					? 'bg-gray-100'
					: 'hover:bg-gray-100'}"
			>
				Trips
			</a>
			<a
				href="/settings"
				class="px-2 py-1.5 text-sm rounded {$page.url.pathname === '/settings'
					? 'bg-gray-100'
					: 'hover:bg-gray-100'}"
				title="Settings"
			>
				<svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
				</svg>
			</a>
		</nav>

		{#if onNewTrip}
			<button
				on:click={onNewTrip}
				class="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded"
			>
				+ New Trip
			</button>
		{:else}
			<a
				href="/trips/new"
				class="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded"
			>
				+ New Trip
			</a>
		{/if}
	</div>
</header>
