import { mount } from 'svelte';
import PublicDashboardView from './components/PublicDashboardView.svelte';

// Extract handle from URL: /public/{handle}
const match = window.location.pathname.match(/^\/public\/([^/.]+)/);
const handle = match?.[1] ?? '';

if (!handle) {
  document.getElementById('app')!.innerHTML = '<p style="text-align:center;color:#999;padding:3rem">Invalid public profile link.</p>';
} else {
  mount(PublicDashboardView, { target: document.getElementById('app')!, props: { handle } });
}
