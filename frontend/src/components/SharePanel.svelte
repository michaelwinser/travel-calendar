<script lang="ts">
  import { onMount } from 'svelte';
  import {
    listShareLinks, createShareLink, deleteShareLink,
    listShares, createShare, deleteShare,
    listSharedWithMe,
    getPublicProfile, updatePublicProfile,
    type ShareLink, type Share, type SharedWithMeEntry, type PublicProfile,
  } from '../lib/api';

  interface Props {
    onclose: () => void;
  }

  let { onclose }: Props = $props();

  // Share links state
  let links = $state<ShareLink[]>([]);
  let linkLabel = $state('');
  let linkShowTitles = $state(false);
  let linkLoading = $state(false);
  let linkCopied = $state('');

  // User shares state
  let shares = $state<Share[]>([]);
  let shareEmail = $state('');
  let shareShowTitles = $state(false);
  let shareLoading = $state(false);

  // Shared with me state
  let sharedWithMe = $state<SharedWithMeEntry[]>([]);

  // Public profile state
  let publicProfile = $state<PublicProfile>({ handle: '', enabled: false });
  let profileHandle = $state('');
  let profileSaving = $state(false);

  let error = $state('');

  onMount(async () => {
    await refresh();
  });

  async function refresh() {
    try {
      const [l, s, swm, pp] = await Promise.all([listShareLinks(), listShares(), listSharedWithMe(), getPublicProfile()]);
      links = l; shares = s; sharedWithMe = swm;
      publicProfile = pp;
      profileHandle = pp.handle;
      error = '';
    } catch (e: any) {
      error = e.message;
    }
  }

  async function handleCreateLink() {
    linkLoading = true;
    try {
      await createShareLink({
        label: linkLabel || undefined,
        showTitle: linkShowTitles || undefined,
      });
      linkLabel = '';
      linkShowTitles = false;
      await refresh();
    } catch (e: any) {
      error = e.message;
    }
    linkLoading = false;
  }

  async function handleDeleteLink(id: string) {
    try {
      await deleteShareLink(id);
      await refresh();
    } catch (e: any) {
      error = e.message;
    }
  }

  function copyLinkURL(token: string) {
    const url = `${window.location.origin}/shared/${token}`;
    navigator.clipboard.writeText(url);
    linkCopied = token;
    setTimeout(() => { if (linkCopied === token) linkCopied = ''; }, 2000);
  }

  async function handleCreateShare() {
    if (!shareEmail) return;
    shareLoading = true;
    try {
      await createShare({
        email: shareEmail,
        showTitle: shareShowTitles || undefined,
      });
      shareEmail = '';
      shareShowTitles = false;
      await refresh();
    } catch (e: any) {
      error = e.message;
    }
    shareLoading = false;
  }

  async function handleDeleteShare(id: string) {
    try {
      await deleteShare(id);
      await refresh();
    } catch (e: any) {
      error = e.message;
    }
  }

  async function handleSaveProfile(enabled: boolean) {
    if (!profileHandle) return;
    profileSaving = true;
    try {
      publicProfile = await updatePublicProfile({ handle: profileHandle, enabled });
      error = '';
    } catch (e: any) {
      error = e.message;
    }
    profileSaving = false;
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="overlay" onclick={onclose}>
  <div class="panel" onclick={(e) => e.stopPropagation()}>
    <div class="panel-header">
      <h2>Sharing</h2>
      <button class="close-btn" onclick={onclose}>&times;</button>
    </div>

    {#if error}
      <p class="error">{error}</p>
    {/if}

    <!-- Public Profile Section -->
    <section>
      <h3>Public Dashboard</h3>
      <p class="hint">A public "Where am I?" page showing your next 30 days — location and type only, no titles or notes.</p>

      <div class="create-form">
        <div class="handle-input">
          <span class="handle-prefix">/public/</span>
          <input
            type="text"
            placeholder="your-handle"
            bind:value={profileHandle}
            onkeydown={(e) => e.key === 'Enter' && handleSaveProfile(true)}
          />
        </div>
        {#if publicProfile.enabled}
          <button class="btn-primary" onclick={() => handleSaveProfile(true)} disabled={profileSaving || !profileHandle}>
            Save
          </button>
          <button class="btn-small btn-danger" onclick={() => handleSaveProfile(false)} disabled={profileSaving}>
            Disable
          </button>
        {:else}
          <button class="btn-primary" onclick={() => handleSaveProfile(true)} disabled={profileSaving || !profileHandle}>
            Enable
          </button>
        {/if}
      </div>

      {#if publicProfile.enabled && publicProfile.handle}
        <div class="profile-status">
          <span class="status-active">Active</span>
          <a href="/public/{publicProfile.handle}" target="_blank" rel="noopener" class="profile-link">
            /public/{publicProfile.handle}
          </a>
        </div>
      {/if}
    </section>

    <!-- Share Links Section -->
    <section>
      <h3>Share Links</h3>
      <p class="hint">Anyone with the link can view your calendar (read-only).</p>

      <div class="create-form">
        <input
          type="text"
          placeholder="Label (optional)"
          bind:value={linkLabel}
          onkeydown={(e) => e.key === 'Enter' && handleCreateLink()}
        />
        <label class="checkbox">
          <input type="checkbox" bind:checked={linkShowTitles} />
          Show titles
        </label>
        <button class="btn-primary" onclick={handleCreateLink} disabled={linkLoading}>
          Create link
        </button>
      </div>

      {#if links.length > 0}
        <ul class="item-list">
          {#each links as link (link.id)}
            <li>
              <div class="item-main">
                <span class="item-label">{link.label}</span>
                {#if link.showTitle}
                  <span class="tag">titles visible</span>
                {/if}
                {#if link.expiresAt}
                  <span class="tag">expires {link.expiresAt.slice(0, 10)}</span>
                {/if}
              </div>
              <div class="item-actions">
                <button
                  class="btn-small"
                  onclick={() => copyLinkURL(link.token)}
                >
                  {linkCopied === link.token ? 'Copied!' : 'Copy URL'}
                </button>
                <button class="btn-small btn-danger" onclick={() => handleDeleteLink(link.id)}>
                  Revoke
                </button>
              </div>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="empty">No share links yet.</p>
      {/if}
    </section>

    <!-- User Shares Section -->
    <section>
      <h3>Share with People</h3>
      <p class="hint">Share your calendar with specific users by email.</p>

      <div class="create-form">
        <input
          type="email"
          placeholder="Email address"
          bind:value={shareEmail}
          onkeydown={(e) => e.key === 'Enter' && handleCreateShare()}
        />
        <label class="checkbox">
          <input type="checkbox" bind:checked={shareShowTitles} />
          Show titles
        </label>
        <button class="btn-primary" onclick={handleCreateShare} disabled={shareLoading || !shareEmail}>
          Share
        </button>
      </div>

      {#if shares.length > 0}
        <ul class="item-list">
          {#each shares as share (share.id)}
            <li>
              <div class="item-main">
                <span class="item-label">{share.sharedWith}</span>
                {#if share.showTitle}
                  <span class="tag">titles visible</span>
                {/if}
              </div>
              <div class="item-actions">
                <button class="btn-small btn-danger" onclick={() => handleDeleteShare(share.id)}>
                  Revoke
                </button>
              </div>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="empty">Not shared with anyone yet.</p>
      {/if}
    </section>

    <!-- Shared with me Section -->
    <section>
      <h3>Shared with Me</h3>
      <p class="hint">Calendars other people have shared with you.</p>

      {#if sharedWithMe.length > 0}
        <ul class="item-list">
          {#each sharedWithMe as entry (entry.shareId)}
            <li>
              <div class="item-main">
                <span class="item-label">{entry.ownerEmail}</span>
              </div>
              <div class="item-actions">
                <a
                  class="btn-small"
                  href="/view/{encodeURIComponent(entry.ownerEmail)}"
                  target="_blank"
                  rel="noopener"
                >View</a>
              </div>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="empty">No one has shared their calendar with you yet.</p>
      {/if}
    </section>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    z-index: 100;
    background: rgba(0, 0, 0, 0.3);
    display: flex;
    justify-content: flex-end;
  }

  .panel {
    width: 420px;
    max-width: 90vw;
    background: white;
    height: 100%;
    overflow-y: auto;
    padding: 1.25rem;
    box-shadow: -4px 0 20px rgba(0, 0, 0, 0.15);
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1rem;
  }

  .panel-header h2 {
    margin: 0;
    font-size: 1.2rem;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    color: #888;
    padding: 0 0.25rem;
    line-height: 1;
  }

  .close-btn:hover { color: #333; }

  section {
    margin-bottom: 1.5rem;
  }

  h3 {
    font-size: 0.95rem;
    margin: 0 0 0.25rem;
    color: #333;
  }

  .hint {
    font-size: 0.8rem;
    color: #888;
    margin: 0 0 0.75rem;
  }

  .create-form {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    flex-wrap: wrap;
    margin-bottom: 0.75rem;
  }

  .create-form input[type="text"],
  .create-form input[type="email"] {
    flex: 1;
    min-width: 140px;
    padding: 0.4rem 0.6rem;
    border: 1px solid #ddd;
    border-radius: 6px;
    font-size: 0.85rem;
    font-family: inherit;
  }

  .create-form input:focus {
    outline: none;
    border-color: #333;
  }

  .checkbox {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    font-size: 0.8rem;
    color: #666;
    white-space: nowrap;
    cursor: pointer;
  }

  .checkbox input[type="checkbox"] {
    margin: 0;
  }

  .btn-primary {
    padding: 0.4rem 0.75rem;
    border: none;
    border-radius: 6px;
    background: #333;
    color: white;
    font-size: 0.8rem;
    cursor: pointer;
    white-space: nowrap;
  }

  .btn-primary:hover { background: #555; }
  .btn-primary:disabled { opacity: 0.4; cursor: default; }

  .item-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
    background: #eee;
    border-radius: 6px;
    overflow: hidden;
  }

  .item-list li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 0.65rem;
    background: white;
    gap: 0.5rem;
  }

  .item-main {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    min-width: 0;
    flex: 1;
  }

  .item-label {
    font-size: 0.85rem;
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tag {
    font-size: 0.65rem;
    color: #888;
    background: #f3f4f6;
    border-radius: 3px;
    padding: 0.1rem 0.35rem;
    white-space: nowrap;
  }

  .item-actions {
    display: flex;
    gap: 0.35rem;
    flex-shrink: 0;
  }

  .btn-small {
    padding: 0.2rem 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    background: white;
    font-size: 0.7rem;
    cursor: pointer;
    color: #555;
    white-space: nowrap;
  }

  .btn-small:hover {
    border-color: #999;
    color: #333;
  }

  .btn-danger {
    color: #dc2626;
    border-color: #fecaca;
  }

  .btn-danger:hover {
    color: #b91c1c;
    border-color: #dc2626;
    background: #fef2f2;
  }

  .handle-input {
    display: flex;
    align-items: center;
    flex: 1;
    min-width: 140px;
    border: 1px solid #ddd;
    border-radius: 6px;
    overflow: hidden;
  }

  .handle-prefix {
    padding: 0.4rem 0 0.4rem 0.6rem;
    font-size: 0.8rem;
    color: #999;
    white-space: nowrap;
  }

  .handle-input input {
    border: none;
    outline: none;
    padding: 0.4rem 0.6rem 0.4rem 0;
    font-size: 0.85rem;
    font-family: inherit;
    flex: 1;
    min-width: 0;
  }

  .handle-input:focus-within {
    border-color: #333;
  }

  .profile-status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-top: 0.5rem;
    font-size: 0.8rem;
  }

  .status-active {
    color: #22c55e;
    font-weight: 600;
    font-size: 0.75rem;
  }

  .profile-link {
    color: #0066cc;
    text-decoration: none;
  }

  .profile-link:hover {
    text-decoration: underline;
  }

  .empty {
    font-size: 0.8rem;
    color: #aaa;
    padding: 0.5rem 0;
  }

  .error {
    color: #dc2626;
    font-size: 0.8rem;
    margin: 0 0 0.75rem;
    padding: 0.4rem 0.6rem;
    background: #fef2f2;
    border-radius: 6px;
  }
</style>
