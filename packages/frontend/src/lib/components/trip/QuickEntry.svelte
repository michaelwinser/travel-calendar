<script lang="ts">
	import { parseTrip } from '$lib/utils/tripParser';
	import type { ParsedTrip } from '$lib/utils/tripParser';

	export let onSubmit: (parsed: ParsedTrip) => void;

	let input = '';
	let preview: ParsedTrip | null = null;

	$: preview = input.length > 2 ? parseTrip(input) : null;

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && preview) {
			e.preventDefault();
			onSubmit(preview);
			input = '';
			preview = null;
		}
		if (e.key === 'Escape') {
			input = '';
			preview = null;
		}
	}

	function formatDateDisplay(dateStr: string): string {
		if (!dateStr) return '';
		const d = new Date(dateStr + 'T00:00:00');
		return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}
</script>

<div class="relative">
	<input
		type="text"
		bind:value={input}
		on:keydown={handleKeydown}
		placeholder="Quick add: Milan Jan 23-27 or London next week business"
		class="w-full px-4 py-2.5 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
	/>

	{#if preview}
		<div class="absolute z-10 left-0 right-0 mt-1 bg-white border rounded-lg shadow-lg p-3">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-3 text-sm">
					<span class="font-medium text-gray-900">{preview.name}</span>
					{#if preview.startDate}
						<span class="text-gray-500">
							{formatDateDisplay(preview.startDate)}{preview.endDate && preview.endDate !== preview.startDate ? ' – ' + formatDateDisplay(preview.endDate) : ''}
						</span>
					{/if}
					{#if preview.purpose}
						<span class="px-2 py-0.5 text-xs rounded-full bg-gray-100 text-gray-600">
							{preview.purpose}
						</span>
					{/if}
				</div>
				<span class="text-xs text-gray-400">Enter to create</span>
			</div>
		</div>
	{/if}
</div>
