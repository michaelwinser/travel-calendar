<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { DayEntry } from '$lib/stores/dayEntries';

	export let date: string; // YYYY-MM-DD
	export let entries: DayEntry[] = [];
	export let x: number = 0;
	export let y: number = 0;

	const dispatch = createEventDispatcher<{
		create: { date: string; text: string };
		delete: { id: string };
		close: void;
	}>();

	let input = '';

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && input.trim()) {
			e.preventDefault();
			dispatch('create', { date, text: input.trim() });
			input = '';
		}
		if (e.key === 'Escape') {
			dispatch('close');
		}
	}

	function handleDelete(id: string) {
		dispatch('delete', { id });
	}

	function formatDate(dateStr: string): string {
		const d = new Date(dateStr + 'T00:00:00');
		return d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
	}

	// Auto-focus the input
	function focusInput(node: HTMLInputElement) {
		node.focus();
	}
</script>

<div
	class="fixed z-50 bg-white border rounded-lg shadow-xl p-3 w-72"
	style="left: {x}px; top: {y}px;"
>
	<div class="flex items-center justify-between mb-2">
		<span class="text-sm font-medium text-gray-700">{formatDate(date)}</span>
		<button
			class="text-gray-400 hover:text-gray-600"
			on:click={() => dispatch('close')}
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
			</svg>
		</button>
	</div>

	<!-- Existing entries for this day -->
	{#if entries.length > 0}
		<div class="space-y-1 mb-2">
			{#each entries as entry (entry.id)}
				<div class="flex items-center justify-between px-2 py-1 bg-gray-50 rounded text-sm group">
					<div>
						<span class="text-gray-900">{entry.location}</span>
						{#if entry.description}
							<span class="text-gray-500"> — {entry.description}</span>
						{/if}
					</div>
					<button
						class="text-gray-300 hover:text-red-500 opacity-0 group-hover:opacity-100"
						on:click={() => handleDelete(entry.id)}
					>
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Add new entry -->
	<input
		type="text"
		bind:value={input}
		on:keydown={handleKeydown}
		use:focusInput
		placeholder="dentist in Fairfield"
		class="w-full px-2 py-1.5 text-sm border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
	/>
	<p class="text-xs text-gray-400 mt-1">Enter to add</p>
</div>

<!-- Backdrop to close on click-outside -->
<button
	class="fixed inset-0 z-40 cursor-default"
	on:click={() => dispatch('close')}
	tabindex="-1"
></button>
