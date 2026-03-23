<script lang="ts">
  import { parseActivity, type ParseResult, type ActivityType } from '../lib/api';
  import ActivityForm from './ActivityForm.svelte';

  interface Props {
    oncreate: (data: {
      title: string;
      type: ActivityType;
      startDate: string;
      endDate: string;
      location: string;
      notes: string;
      parseHistoryId?: string;
    }) => void;
  }

  let { oncreate }: Props = $props();

  let text = $state('');
  let parseResult = $state<ParseResult | null>(null);
  let showForm = $state(false);
  let debounceTimer: ReturnType<typeof setTimeout> | undefined;

  function handleInput() {
    if (!text.trim()) {
      parseResult = null;
      showForm = false;
      return;
    }
    showForm = true;

    // Debounce parse calls
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(async () => {
      try {
        parseResult = await parseActivity(text);
      } catch {
        // Parse failed — leave form with whatever it has
      }
    }, 300);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      text = '';
      parseResult = null;
      showForm = false;
    } else if (e.key === 'Enter' && showForm && parseResult) {
      e.preventDefault();
      const a = parseResult.activity;
      if (a.title && a.startDate) {
        handleSubmit({
          title: a.title,
          type: a.type ?? 'stay',
          startDate: a.startDate,
          endDate: a.endDate ?? a.startDate,
          location: a.location ?? '',
          notes: '',
        });
      }
    }
  }

  function handleSubmit(data: {
    title: string;
    type: ActivityType;
    startDate: string;
    endDate: string;
    location: string;
    notes: string;
  }) {
    oncreate({
      ...data,
      parseHistoryId: parseResult?.id,
    });
    text = '';
    parseResult = null;
    showForm = false;
  }

  function handleCancel() {
    text = '';
    parseResult = null;
    showForm = false;
  }
</script>

<div class="quick-add">
  <input
    type="text"
    bind:value={text}
    oninput={handleInput}
    onkeydown={handleKeydown}
    placeholder="Quick add: FOSDEM Jan 22 - Feb 3 in Brussels"
  />
</div>

{#if showForm}
  <ActivityForm
    mode="create"
    title={parseResult?.activity?.title ?? ''}
    type={parseResult?.activity?.type ?? 'stay'}
    startDate={parseResult?.activity?.startDate ?? ''}
    endDate={parseResult?.activity?.endDate ?? ''}
    location={parseResult?.activity?.location ?? ''}
    confidence={parseResult?.confidence ?? {}}
    onsubmit={handleSubmit}
    oncancel={handleCancel}
  />
{/if}

<style>
  .quick-add {
    margin-bottom: 1rem;
  }

  .quick-add input {
    width: 100%;
    padding: 0.7rem 1rem;
    border: 1px solid #ddd;
    border-radius: 8px;
    font-size: 1rem;
    font-family: inherit;
    box-sizing: border-box;
  }

  .quick-add input:focus {
    outline: none;
    border-color: #333;
    box-shadow: 0 0 0 3px rgba(0, 0, 0, 0.05);
  }

  .quick-add input::placeholder {
    color: #aaa;
  }
</style>
