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
