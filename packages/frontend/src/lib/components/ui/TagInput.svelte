<script lang="ts">
	export let tags: string[] = [];
	export let suggestions: string[] = [];
	export let placeholder: string = 'Add location...';
	export let onAdd: (tag: string) => void = () => {};
	export let onRemove: (tag: string) => void = () => {};

	let inputValue = '';
	let showSuggestions = false;
	let selectedSuggestionIndex = -1;
	let inputElement: HTMLInputElement;

	$: filteredSuggestions = suggestions.filter(
		(s) =>
			s.toLowerCase().includes(inputValue.toLowerCase()) &&
			!tags.includes(s) &&
			inputValue.length > 0
	);

	function addTag(value: string) {
		const trimmed = value.trim();
		if (trimmed && !tags.includes(trimmed)) {
			onAdd(trimmed);
		}
		inputValue = '';
		showSuggestions = false;
		selectedSuggestionIndex = -1;
	}

	function removeTag(tag: string) {
		onRemove(tag);
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' || event.key === ',') {
			event.preventDefault();
			if (selectedSuggestionIndex >= 0 && filteredSuggestions[selectedSuggestionIndex]) {
				addTag(filteredSuggestions[selectedSuggestionIndex]);
			} else if (inputValue.trim()) {
				addTag(inputValue);
			}
		} else if (event.key === 'Tab' && filteredSuggestions.length > 0) {
			event.preventDefault();
			const suggestion = filteredSuggestions[selectedSuggestionIndex >= 0 ? selectedSuggestionIndex : 0];
			if (suggestion) {
				addTag(suggestion);
			}
		} else if (event.key === 'Backspace' && inputValue === '' && tags.length > 0) {
			removeTag(tags[tags.length - 1]);
		} else if (event.key === 'ArrowDown' && showSuggestions) {
			event.preventDefault();
			selectedSuggestionIndex = Math.min(selectedSuggestionIndex + 1, filteredSuggestions.length - 1);
		} else if (event.key === 'ArrowUp' && showSuggestions) {
			event.preventDefault();
			selectedSuggestionIndex = Math.max(selectedSuggestionIndex - 1, -1);
		} else if (event.key === 'Escape') {
			showSuggestions = false;
			selectedSuggestionIndex = -1;
		}
	}

	function handleInput() {
		showSuggestions = inputValue.length > 0 && filteredSuggestions.length > 0;
		selectedSuggestionIndex = -1;
	}

	function handleFocus() {
		if (inputValue.length > 0 && filteredSuggestions.length > 0) {
			showSuggestions = true;
		}
	}

	function handleBlur() {
		// Delay hiding to allow click on suggestion
		setTimeout(() => {
			showSuggestions = false;
		}, 150);
	}

	function selectSuggestion(suggestion: string) {
		addTag(suggestion);
		inputElement?.focus();
	}
</script>

<div class="tag-input-container relative">
	<div
		class="flex flex-wrap items-center gap-1.5 p-2 border rounded-md bg-white min-h-[42px] focus-within:ring-2 focus-within:ring-blue-500 focus-within:border-blue-500"
	>
		{#each tags as tag}
			<span class="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-100 text-blue-800 text-sm rounded">
				{tag}
				<button
					type="button"
					class="hover:text-blue-600 focus:outline-none"
					on:click={() => removeTag(tag)}
					aria-label="Remove {tag}"
				>
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</span>
		{/each}
		<input
			bind:this={inputElement}
			bind:value={inputValue}
			type="text"
			class="flex-1 min-w-[120px] outline-none text-sm"
			{placeholder}
			on:keydown={handleKeydown}
			on:input={handleInput}
			on:focus={handleFocus}
			on:blur={handleBlur}
		/>
	</div>

	{#if showSuggestions && filteredSuggestions.length > 0}
		<ul class="absolute z-10 w-full mt-1 bg-white border rounded-md shadow-lg max-h-48 overflow-auto">
			{#each filteredSuggestions as suggestion, index}
				<li>
					<button
						type="button"
						class="w-full text-left px-3 py-2 text-sm hover:bg-blue-50 {index === selectedSuggestionIndex
							? 'bg-blue-100'
							: ''}"
						on:mousedown|preventDefault={() => selectSuggestion(suggestion)}
					>
						{suggestion}
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>
