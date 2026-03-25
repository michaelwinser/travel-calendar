import { mount } from 'svelte';
import SharedCalendarView from './components/SharedCalendarView.svelte';

// Extract token from URL: /shared/{token}[/view[/date]]
const match = window.location.pathname.match(/^\/shared\/([^/.]+)/);
const token = match?.[1] ?? '';

if (!token) {
  document.getElementById('app')!.innerHTML = '<p style="text-align:center;color:#999;padding:3rem">Invalid share link.</p>';
} else {
  mount(SharedCalendarView, { target: document.getElementById('app')!, props: { token } });
}
