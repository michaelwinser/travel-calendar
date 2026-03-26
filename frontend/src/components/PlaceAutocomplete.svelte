<script lang="ts">
  import { resolvePlaces, createPlace, type PlaceSuggestion, type PlaceResolveResponse } from '../lib/api';

  interface Props {
    value: string;
    placeId: string;
    onchange: (value: string, placeId: string) => void;
    placeholder?: string;
  }

  let { value, placeId, onchange, placeholder = 'Location' }: Props = $props();

  let inputValue = $state(value);
  let suggestions = $state<PlaceSuggestion[]>([]);
  let showDropdown = $state(false);
  let selectedIndex = $state(-1);
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let inputEl: HTMLInputElement;
  let lastPropValue = value;

  // Sync only when the parent *prop* changes (not when inputValue changes)
  $effect(() => {
    if (value !== lastPropValue) {
      lastPropValue = value;
      inputValue = value;
    }
  });

  function handleInput() {
    // Always propagate text changes to parent; clear place link when editing
    onchange(inputValue, placeId && inputValue !== value ? '' : placeId);

    if (debounceTimer) clearTimeout(debounceTimer);

    if (inputValue.length < 2) {
      suggestions = [];
      showDropdown = false;
      return;
    }

    debounceTimer = setTimeout(async () => {
      try {
        const result = await resolvePlaces(inputValue);

        // Build suggestion list: exact match first (if any), then other suggestions
        const allSuggestions: PlaceSuggestion[] = [];
        if (result.exact) {
          allSuggestions.push({
            source: 'user',
            place: result.exact,
            name: result.exact.name,
            score: 1.0,
          });
        }
        for (const s of result.suggestions ?? []) {
          // Skip duplicate of exact match
          if (result.exact && s.source === 'user' && s.place?.id === result.exact.id) continue;
          allSuggestions.push(s);
        }

        suggestions = allSuggestions;
        showDropdown = suggestions.length > 0;
        selectedIndex = -1;
      } catch {
        suggestions = [];
        showDropdown = false;
      }
    }, 200);
  }

  async function selectSuggestion(sug: PlaceSuggestion) {
    showDropdown = false;

    // User explicitly selected this suggestion — use the canonical name
    const displayName = formatSuggestion(sug);

    if (sug.source === 'user' && sug.place) {
      // Existing user place — link directly
      inputValue = sug.place.name;
      onchange(inputValue, sug.place.id);
    } else {
      // Gazetteer suggestion — create a new place with the canonical name
      inputValue = displayName;
      try {
        const newPlace = await createPlace({
          name: displayName,
          city: sug.name,
          country: sug.country ?? undefined,
          latitude: sug.latitude ?? undefined,
          longitude: sug.longitude ?? undefined,
          timezone: sug.timezone ?? undefined,
          kind: 'city',
        });
        onchange(displayName, newPlace.id);
      } catch {
        onchange(displayName, '');
      }
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (!showDropdown) return;

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      selectedIndex = Math.min(selectedIndex + 1, suggestions.length - 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      selectedIndex = Math.max(selectedIndex - 1, -1);
    } else if (e.key === 'Enter' && selectedIndex >= 0) {
      e.preventDefault();
      selectSuggestion(suggestions[selectedIndex]);
    } else if (e.key === 'Escape') {
      showDropdown = false;
    }
  }

  function handleBlur() {
    // Delay to allow click on dropdown item
    setTimeout(() => {
      showDropdown = false;
    }, 150);
  }

  function formatSuggestion(sug: PlaceSuggestion): string {
    let label = sug.name;
    if (sug.country) label += `, ${sug.country}`;
    return label;
  }

  function formatMeta(sug: PlaceSuggestion): string {
    const parts: string[] = [];
    if (sug.timezone) parts.push(sug.timezone);
    if (sug.population) parts.push(`pop ${(sug.population / 1000).toFixed(0)}k`);
    return parts.join(' · ');
  }
</script>

<div class="place-autocomplete">
  <input
    type="text"
    bind:value={inputValue}
    bind:this={inputEl}
    oninput={handleInput}
    onkeydown={handleKeydown}
    onblur={handleBlur}
    onfocus={() => { if (suggestions.length > 0) showDropdown = true; }}
    {placeholder}
    autocomplete="off"
  />
  {#if placeId}
    <span class="linked-badge" title="Linked to a place">●</span>
  {/if}

  {#if showDropdown && suggestions.length > 0}
    <div class="dropdown">
      {#each suggestions as sug, i}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div
          class="suggestion"
          class:selected={i === selectedIndex}
          class:is-user={sug.source === 'user'}
          onmousedown={() => selectSuggestion(sug)}
        >
          <span class="sug-name">{formatSuggestion(sug)}</span>
          <span class="sug-source">{sug.source === 'user' ? 'your place' : formatMeta(sug)}</span>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .place-autocomplete {
    position: relative;
    flex: 1;
  }

  input {
    width: 100%;
    padding: 0.5rem 0.6rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    font-size: 0.9rem;
    font-family: inherit;
  }

  input:focus {
    outline: none;
    border-color: #333;
  }

  .linked-badge {
    position: absolute;
    right: 8px;
    top: 50%;
    transform: translateY(-50%);
    color: #22c55e;
    font-size: 0.6rem;
    pointer-events: none;
  }

  .dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: white;
    border: 1px solid #ddd;
    border-radius: 0 0 6px 6px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    z-index: 50;
    max-height: 200px;
    overflow-y: auto;
  }

  .suggestion {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.4rem 0.6rem;
    cursor: pointer;
    font-size: 0.85rem;
  }

  .suggestion:hover,
  .suggestion.selected {
    background: #f3f4f6;
  }

  .suggestion.is-user {
    border-left: 3px solid #22c55e;
  }

  .sug-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sug-source {
    font-size: 0.7rem;
    color: #999;
    white-space: nowrap;
    margin-left: 0.5rem;
    flex-shrink: 0;
  }
</style>
