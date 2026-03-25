<script lang="ts">
  import type { OverlayCalendar } from '../lib/api';

  interface Props {
    overlays: OverlayCalendar[];
    ontoggle: (email: string) => void;
  }

  let { overlays, ontoggle }: Props = $props();
</script>

{#if overlays.length > 0}
  <div class="sidebar">
    <div class="sidebar-header">Other calendars</div>
    {#each overlays as cal (cal.email)}
      <label class="cal-entry">
        <input
          type="checkbox"
          checked={cal.visible}
          onchange={() => ontoggle(cal.email)}
        />
        <span class="cal-dot" style="background: {cal.color}"></span>
        <span class="cal-email">{cal.email}</span>
      </label>
    {/each}
  </div>
{/if}

<style>
  .sidebar {
    margin-bottom: 1rem;
    padding: 0.5rem 0.75rem;
    background: white;
    border: 1px solid #eee;
    border-radius: 8px;
    font-size: 0.8rem;
  }

  .sidebar-header {
    font-size: 0.7rem;
    font-weight: 600;
    color: #999;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.35rem;
  }

  .cal-entry {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.2rem 0;
    cursor: pointer;
  }

  .cal-entry input[type="checkbox"] {
    margin: 0;
  }

  .cal-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .cal-email {
    color: #555;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
