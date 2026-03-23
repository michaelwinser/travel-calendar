<script lang="ts">
  import { onMount } from 'svelte';
  import { ACTIVITY_TYPES, ACTIVITY_COLORS, type ActivityType } from '../lib/api';

  interface Props {
    title?: string;
    type?: ActivityType;
    startDate?: string;
    endDate?: string;
    location?: string;
    notes?: string;
    confidence?: Record<string, string>;
    mode: 'create' | 'edit';
    onsubmit: (data: {
      title: string;
      type: ActivityType;
      startDate: string;
      endDate: string;
      location: string;
      notes: string;
    }) => void;
    oncancel: () => void;
    ondelete?: () => void;
  }

  let props: Props = $props();

  let title = $state(props.title ?? '');
  let type = $state<ActivityType>(props.type ?? 'stay');
  let startDate = $state(props.startDate ?? '');
  let endDate = $state(props.endDate ?? '');
  let location = $state(props.location ?? '');
  let notes = $state(props.notes ?? '');
  let error = $state('');
  let confirmingDelete = $state(false);
  let titleInput: HTMLInputElement;

  onMount(() => { titleInput?.focus(); });

  // Sync from props when they change (live parse updates)
  $effect(() => { if (props.title) title = props.title; });
  $effect(() => { if (props.type) type = props.type; });
  $effect(() => { if (props.startDate) startDate = props.startDate; });
  $effect(() => { if (props.endDate) endDate = props.endDate; });
  $effect(() => { if (props.location) location = props.location; });

  function handleSubmit(e: Event) {
    e.preventDefault();
    error = '';
    if (!title.trim()) { error = 'Title is required'; return; }
    if (!startDate) { error = 'Start date is required'; return; }
    props.onsubmit({
      title: title.trim(),
      type,
      startDate,
      endDate: endDate || startDate,
      location: location.trim(),
      notes: notes.trim(),
    });
  }

  let canSubmit = $derived(title.trim() !== '' && startDate !== '');

  function confClass(field: string): string {
    const c = (props.confidence ?? {})[field];
    if (c === 'low') return 'conf-low';
    if (c === 'medium') return 'conf-medium';
    return '';
  }
</script>

{#snippet formContent()}
  {#if error}
    <p class="error">{error}</p>
  {/if}

  <form onsubmit={handleSubmit}>
    <label class={confClass('title')}>
      <span>Title</span>
      <input type="text" bind:value={title} bind:this={titleInput} />
    </label>

    <label class={confClass('type')}>
      <span>Type</span>
      <div class="type-picker">
        {#each ACTIVITY_TYPES as t}
          <button
            type="button"
            class="type-btn"
            class:selected={type === t}
            style="--color: {ACTIVITY_COLORS[t]}"
            onclick={() => type = t}
          >{t}</button>
        {/each}
      </div>
    </label>

    <div class="date-row">
      <label class={confClass('startDate')}>
        <span>Start</span>
        <input type="date" bind:value={startDate} />
      </label>
      <label class={confClass('endDate')}>
        <span>End</span>
        <input type="date" bind:value={endDate} />
      </label>
    </div>

    <label class={confClass('location')}>
      <span>Location</span>
      <input type="text" bind:value={location} placeholder="e.g. Brussels, Home" />
    </label>

    <label>
      <span>Notes</span>
      <textarea bind:value={notes} rows="2"></textarea>
    </label>

    <div class="actions">
      {#if props.mode === 'edit' && props.ondelete}
        {#if confirmingDelete}
          <span class="delete-confirm">
            Delete this activity?
            <button type="button" class="delete-yes" onclick={props.ondelete}>Yes, delete</button>
            <button type="button" class="cancel-btn" onclick={() => confirmingDelete = false}>No</button>
          </span>
        {:else}
          <button type="button" class="delete-btn" onclick={() => confirmingDelete = true}>Delete</button>
        {/if}
      {/if}
      <div class="spacer"></div>
      {#if !confirmingDelete}
        <button type="button" class="cancel-btn" onclick={props.oncancel}>Cancel</button>
        <button type="submit" class="submit-btn" disabled={!canSubmit}>
          {props.mode === 'create' ? 'Create' : 'Save'}
        </button>
      {/if}
    </div>
  </form>
{/snippet}

{#if props.mode === 'edit'}
  <!-- Modal overlay for editing existing activities -->
  <!-- svelte-ignore a11y_interactive_supports_focus -->
  <div class="overlay" onclick={props.oncancel} onkeydown={(e) => e.key === 'Escape' && props.oncancel()} role="dialog">
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_noninteractive_element_interactions -->
    <div class="modal" onclick={(e) => e.stopPropagation()} role="document">
      <h2>Edit Activity</h2>
      {@render formContent()}
    </div>
  </div>
{:else}
  <!-- Inline form for quick add -->
  <div class="inline-form">
    {@render formContent()}
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .modal {
    background: white;
    border-radius: 12px;
    padding: 1.5rem;
    width: 90%;
    max-width: 480px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  }

  .inline-form {
    background: white;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    padding: 1rem;
    margin-bottom: 1rem;
  }

  h2 {
    margin: 0 0 1rem;
    font-size: 1.2rem;
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  label {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  label span {
    font-size: 0.8rem;
    font-weight: 600;
    color: #555;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  input, textarea {
    padding: 0.5rem 0.6rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    font-size: 0.95rem;
    font-family: inherit;
  }

  input:focus, textarea:focus {
    outline: none;
    border-color: #333;
  }

  .conf-low input {
    border-color: #f59e0b;
    background: #fffbeb;
  }

  .conf-medium input {
    border-color: #3b82f6;
    background: #eff6ff;
  }

  .date-row {
    display: flex;
    gap: 0.75rem;
  }

  .date-row label {
    flex: 1;
  }

  .type-picker {
    display: flex;
    gap: 0.4rem;
    flex-wrap: wrap;
  }

  .type-btn {
    padding: 0.3rem 0.7rem;
    border: 2px solid transparent;
    border-radius: 16px;
    background: #f3f4f6;
    font-size: 0.8rem;
    cursor: pointer;
    text-transform: capitalize;
  }

  .type-btn.selected {
    background: var(--color);
    color: white;
    border-color: var(--color);
  }

  .type-btn:not(.selected):hover {
    border-color: var(--color);
  }

  .actions {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.5rem;
    align-items: center;
  }

  .spacer { flex: 1; }

  .submit-btn {
    padding: 0.5rem 1.2rem;
    border: none;
    border-radius: 8px;
    background: #333;
    color: white;
    font-size: 0.95rem;
    cursor: pointer;
  }

  .submit-btn:hover:not(:disabled) { background: #555; }

  .submit-btn:disabled {
    opacity: 0.4;
    cursor: default;
  }

  .cancel-btn {
    padding: 0.5rem 1rem;
    border: 1px solid #ddd;
    border-radius: 8px;
    background: white;
    font-size: 0.95rem;
    cursor: pointer;
  }

  .cancel-btn:hover { background: #f5f5f5; }

  .delete-btn {
    padding: 0.5rem 1rem;
    border: 1px solid #fca5a5;
    border-radius: 8px;
    background: white;
    color: #dc2626;
    font-size: 0.95rem;
    cursor: pointer;
  }

  .delete-btn:hover { background: #fef2f2; }

  .delete-confirm {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.85rem;
    color: #dc2626;
  }

  .delete-yes {
    padding: 0.35rem 0.75rem;
    border: none;
    border-radius: 6px;
    background: #dc2626;
    color: white;
    font-size: 0.85rem;
    cursor: pointer;
  }

  .delete-yes:hover { background: #b91c1c; }

  .error {
    color: #dc2626;
    font-size: 0.85rem;
    margin: 0;
  }
</style>
