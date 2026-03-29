<script lang="ts">
  import { onMount } from 'svelte';
  import { ACTIVITY_COLORS, type Activity } from '../lib/api';
  import { formatDateRange } from '../lib/date-utils';

  type TripStatus = 'planned' | 'confirmed' | 'tentative';

  interface Props {
    name: string;
    color: string;
    startDate?: string;
    endDate?: string;
    status?: string;
    unassignedActivities?: Activity[];
    onsubmit: (data: { name: string; color: string; startDate?: string; endDate?: string; status?: TripStatus }) => void;
    ondelete: () => void;
    oncancel: () => void;
    onassign?: (activityId: string) => void;
  }

  let {
    name: initialName, color: initialColor,
    startDate: initialStartDate, endDate: initialEndDate,
    status: initialStatus,
    unassignedActivities, onsubmit, ondelete, oncancel, onassign,
  }: Props = $props();

  let name = $state(initialName);
  let color = $state(initialColor);
  let startDate = $state(initialStartDate ?? '');
  let endDate = $state(initialEndDate ?? '');
  let status = $state<TripStatus>((initialStatus as TripStatus) ?? 'planned');
  let confirmingDelete = $state(false);
  let nameInput: HTMLInputElement;

  onMount(() => { nameInput?.focus(); });

  const COLORS = [
    '#4f86c6', '#e07b53', '#6bb86a', '#c75ca2',
    '#d4a843', '#5cbcb6', '#8b6cc1', '#c95454',
    '#5a8f5a', '#c4853d',
  ];

  function handleSubmit(e: Event) {
    e.preventDefault();
    if (!name.trim()) return;
    onsubmit({
      name: name.trim(),
      color,
      startDate: startDate || undefined,
      endDate: endDate || undefined,
      status,
    });
  }
</script>

<!-- svelte-ignore a11y_interactive_supports_focus -->
<div class="overlay" onclick={oncancel} onkeydown={(e) => e.key === 'Escape' && oncancel()} role="dialog">
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_noninteractive_element_interactions -->
  <div class="modal" onclick={(e) => e.stopPropagation()} role="document">
    <h2>Edit Trip</h2>

    <form onsubmit={handleSubmit}>
      <label>
        <span>Name</span>
        <input type="text" bind:value={name} bind:this={nameInput} />
      </label>

      <div class="date-row">
        <label>
          <span>Start</span>
          <input type="date" bind:value={startDate} />
        </label>
        <label>
          <span>End</span>
          <input type="date" bind:value={endDate} />
        </label>
      </div>

      <label>
        <span>Status</span>
        <div class="status-picker">
          {#each ['planned', 'confirmed', 'tentative'] as s}
            <button
              type="button"
              class="status-btn"
              class:selected={status === s}
              onclick={() => status = s as TripStatus}
            >{s}</button>
          {/each}
        </div>
      </label>

      <label>
        <span>Color</span>
        <div class="color-picker">
          {#each COLORS as c}
            <button
              type="button"
              class="color-swatch"
              class:selected={color === c}
              style="background: {c};"
              onclick={() => color = c}
              aria-label="Color {c}"
            ></button>
          {/each}
        </div>
      </label>

      {#if unassignedActivities && unassignedActivities.length > 0}
        <div class="suggestions">
          <span class="suggestions-label">Unassigned activities in range</span>
          <div class="suggestion-list">
            {#each unassignedActivities as activity (activity.id)}
              <button type="button" class="suggestion-chip" onclick={() => onassign?.(activity.id)}>
                <span class="suggestion-dot" style="background: {ACTIVITY_COLORS[activity.type]}"></span>
                <span class="suggestion-title">{activity.title}</span>
                <span class="suggestion-dates">{formatDateRange(activity.startDate, activity.endDate)}</span>
              </button>
            {/each}
          </div>
        </div>
      {/if}

      <div class="actions">
        {#if confirmingDelete}
          <span class="delete-confirm">
            Delete this trip?
            <button type="button" class="delete-yes" onclick={ondelete}>Yes, delete</button>
            <button type="button" class="cancel-btn" onclick={() => confirmingDelete = false}>No</button>
          </span>
        {:else}
          <button type="button" class="delete-btn" onclick={() => confirmingDelete = true}>Delete trip</button>
        {/if}
        <div class="spacer"></div>
        {#if !confirmingDelete}
          <button type="button" class="cancel-btn" onclick={oncancel}>Cancel</button>
          <button type="submit" class="submit-btn" disabled={!name.trim()}>Save</button>
        {/if}
      </div>
    </form>
  </div>
</div>

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
    max-width: 380px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  }

  h2 {
    margin: 0 0 1rem;
    font-size: 1.1rem;
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

  input {
    padding: 0.5rem 0.6rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    font-size: 0.95rem;
    font-family: inherit;
  }

  input:focus {
    outline: none;
    border-color: #333;
  }

  .date-row {
    display: flex;
    gap: 0.75rem;
  }

  .date-row label {
    flex: 1;
  }

  .status-picker {
    display: flex;
    gap: 0.4rem;
  }

  .status-btn {
    flex: 1;
    padding: 0.4rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    background: white;
    font-size: 0.8rem;
    font-family: inherit;
    cursor: pointer;
    text-transform: capitalize;
    color: #666;
  }

  .status-btn:hover {
    background: #f5f5f5;
  }

  .status-btn.selected {
    border-color: #333;
    background: #333;
    color: white;
  }

  input[type="date"] {
    padding: 0.5rem 0.6rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    font-size: 0.9rem;
    font-family: inherit;
    width: 100%;
    box-sizing: border-box;
  }

  .color-picker {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }

  .color-swatch {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: 3px solid transparent;
    cursor: pointer;
    padding: 0;
  }

  .color-swatch.selected {
    border-color: #333;
    box-shadow: 0 0 0 2px white, 0 0 0 4px #333;
  }

  .color-swatch:hover:not(.selected) {
    border-color: #aaa;
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
  .submit-btn:disabled { opacity: 0.4; cursor: default; }

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
    font-size: 0.85rem;
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

  .suggestions {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .suggestions-label {
    font-size: 0.8rem;
    font-weight: 600;
    color: #555;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .suggestion-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .suggestion-chip {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.35rem 0.5rem;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    background: #f9fafb;
    cursor: pointer;
    font-family: inherit;
    font-size: 0.8rem;
    text-align: left;
    width: 100%;
  }

  .suggestion-chip:hover {
    background: #eef2ff;
    border-color: #c7d2fe;
  }

  .suggestion-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .suggestion-title {
    flex: 1;
    font-weight: 500;
    color: #333;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .suggestion-dates {
    color: #888;
    font-size: 0.7rem;
    white-space: nowrap;
    flex-shrink: 0;
  }
</style>
