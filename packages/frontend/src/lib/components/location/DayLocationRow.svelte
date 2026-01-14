<script lang="ts">
	import TagInput from '../ui/TagInput.svelte';

	export let date: Date;
	export let locations: string[] = [];
	export let suggestions: string[] = [];
	export let onUpdate: (locations: string[]) => void = () => {};

	function formatDateLabel(d: Date): string {
		return d.toLocaleDateString('en-US', {
			weekday: 'short',
			month: 'short',
			day: 'numeric'
		});
	}

	function handleAdd(tag: string) {
		onUpdate([...locations, tag]);
	}

	function handleRemove(tag: string) {
		onUpdate(locations.filter((l) => l !== tag));
	}

	$: dateLabel = formatDateLabel(date);
</script>

<div class="flex items-start gap-4 py-2">
	<div class="w-28 flex-shrink-0 text-sm text-gray-600 pt-2">
		{dateLabel}
	</div>
	<div class="flex-1">
		<TagInput
			tags={locations}
			{suggestions}
			placeholder="Add location..."
			onAdd={handleAdd}
			onRemove={handleRemove}
		/>
	</div>
</div>
