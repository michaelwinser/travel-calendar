<script lang="ts">
  import { onMount } from 'svelte';

  interface Props {
    name: string;
    color: string;
    onsubmit: (data: { name: string; color: string }) => void;
    ondelete: () => void;
    oncancel: () => void;
  }

  let props: Props = $props();

  let name = $state(props.name);
  let color = $state(props.color);
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
    props.onsubmit({ name: name.trim(), color });
  }
</script>

<!-- svelte-ignore a11y_interactive_supports_focus -->
<div class="overlay" onclick={props.oncancel} onkeydown={(e) => e.key === 'Escape' && props.oncancel()} role="dialog">
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_noninteractive_element_interactions -->
  <div class="modal" onclick={(e) => e.stopPropagation()} role="document">
    <h2>Edit Trip</h2>

    <form onsubmit={handleSubmit}>
      <label>
        <span>Name</span>
        <input type="text" bind:value={name} bind:this={nameInput} />
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
            ></button>
          {/each}
        </div>
      </label>

      <div class="actions">
        {#if confirmingDelete}
          <span class="delete-confirm">
            Delete this trip?
            <button type="button" class="delete-yes" onclick={props.ondelete}>Yes, delete</button>
            <button type="button" class="cancel-btn" onclick={() => confirmingDelete = false}>No</button>
          </span>
        {:else}
          <button type="button" class="delete-btn" onclick={() => confirmingDelete = true}>Delete trip</button>
        {/if}
        <div class="spacer"></div>
        {#if !confirmingDelete}
          <button type="button" class="cancel-btn" onclick={props.oncancel}>Cancel</button>
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
</style>
