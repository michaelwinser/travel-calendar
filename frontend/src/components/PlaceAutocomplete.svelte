<script lang="ts">
  import { onMount } from 'svelte';
  import { resolvePlaces, createPlace, getPlace, type PlaceSuggestion, type PlaceResolveResponse, type Place } from '../lib/api';

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
  let chipSuggestions = $state<PlaceSuggestion[]>([]);
  let resolvedPlace = $state<Place | null>(null);

  // Load suggestion chips on mount if there's text but no placeId
  onMount(async () => {
    // Fetch linked place details for display
    if (placeId) {
      try {
        resolvedPlace = await getPlace(placeId);
      } catch { /* ignore */ }
    }

    // Load suggestion chips for unresolved locations
    if (value && !placeId) {
      try {
        const result = await resolvePlaces(value);
        const chips: PlaceSuggestion[] = [];
        if (result.exact) {
          chips.push({ source: 'user', place: result.exact, name: result.exact.name, score: 1.0 });
        }
        for (const s of result.suggestions ?? []) {
          if (result.exact && s.source === 'user' && s.place?.id === result.exact.id) continue;
          chips.push(s);
          if (chips.length >= 3) break;
        }
        chipSuggestions = chips;
      } catch { /* ignore */ }
    }
  });

  // Sync only when the parent *prop* changes (not when inputValue changes)
  $effect(() => {
    if (value !== lastPropValue) {
      lastPropValue = value;
      inputValue = value;
    }
  });

  function handleInput() {
    chipSuggestions = []; // Clear chips when user edits
    resolvedPlace = null; // Clear resolved display
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
    chipSuggestions = [];

    // User explicitly selected this suggestion — use the canonical name
    const displayName = formatSuggestion(sug);

    if (sug.source === 'user' && sug.place) {
      // Existing user place — link directly
      inputValue = sug.place.name;
      resolvedPlace = sug.place;
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
        resolvedPlace = newPlace;
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

  const usStates: Record<string, string> = {
    AL:'Alabama',AK:'Alaska',AZ:'Arizona',AR:'Arkansas',CA:'California',
    CO:'Colorado',CT:'Connecticut',DE:'Delaware',FL:'Florida',GA:'Georgia',
    HI:'Hawaii',ID:'Idaho',IL:'Illinois',IN:'Indiana',IA:'Iowa',KS:'Kansas',
    KY:'Kentucky',LA:'Louisiana',ME:'Maine',MD:'Maryland',MA:'Massachusetts',
    MI:'Michigan',MN:'Minnesota',MS:'Mississippi',MO:'Missouri',MT:'Montana',
    NE:'Nebraska',NV:'Nevada',NH:'New Hampshire',NJ:'New Jersey',NM:'New Mexico',
    NY:'New York',NC:'North Carolina',ND:'North Dakota',OH:'Ohio',OK:'Oklahoma',
    OR:'Oregon',PA:'Pennsylvania',RI:'Rhode Island',SC:'South Carolina',
    SD:'South Dakota',TN:'Tennessee',TX:'Texas',UT:'Utah',VT:'Vermont',
    VA:'Virginia',WA:'Washington',WV:'West Virginia',WI:'Wisconsin',WY:'Wyoming',
    DC:'District of Columbia',
  };

  const countryNames: Record<string, string> = {
    US:'United States',GB:'United Kingdom',CA:'Canada',FR:'France',DE:'Germany',
    IT:'Italy',ES:'Spain',NL:'Netherlands',BE:'Belgium',CH:'Switzerland',
    JP:'Japan',AU:'Australia',NZ:'New Zealand',BR:'Brazil',MX:'Mexico',
    IE:'Ireland',SE:'Sweden',NO:'Norway',DK:'Denmark',AT:'Austria',
  };

  const wellKnown = new Set(['london','paris','tokyo','berlin','rome','amsterdam','brussels','vienna','sydney','dublin','singapore']);

  function formatSuggestion(sug: PlaceSuggestion): string {
    const parts = [sug.name];
    if (sug.admin1 && sug.country === 'US') {
      parts.push(usStates[sug.admin1] ?? sug.admin1);
    } else if (sug.admin1) {
      parts.push(sug.admin1);
    }
    if (sug.country && !wellKnown.has(sug.name.toLowerCase())) {
      parts.push(countryNames[sug.country] ?? sug.country);
    }
    return parts.join(', ');
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
  {#if placeId && resolvedPlace && resolvedPlace.timezone}
    <span class="resolved-info">{resolvedPlace.timezone}</span>
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

  {#if !showDropdown && !placeId && chipSuggestions.length > 0}
    <div class="chips">
      {#each chipSuggestions as chip}
        <button class="chip" onclick={() => selectSuggestion(chip)}>
          {formatSuggestion(chip)}
        </button>
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

  .resolved-info {
    font-size: 0.7rem;
    color: #22c55e;
    margin-top: 0.15rem;
    display: block;
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

  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
    margin-top: 0.3rem;
  }

  .chip {
    font-size: 0.7rem;
    padding: 0.15rem 0.5rem;
    border: 1px solid #ddd;
    border-radius: 12px;
    background: #f8f9fa;
    color: #555;
    cursor: pointer;
    font-family: inherit;
  }

  .chip:hover {
    background: #e5e7eb;
    border-color: #999;
    color: #333;
  }
</style>
